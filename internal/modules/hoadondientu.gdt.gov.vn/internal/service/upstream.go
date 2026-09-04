package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	corehttp "github.com/yunotools/eif/internal/core/protocol/httpclient"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/model"
	moduleutils "github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/utils"
)

const maxRateLimitBackoff = 15 * time.Second

type upstreamLimiter struct {
	slot        chan struct{}
	mu          sync.Mutex
	nextStartAt time.Time
	minInterval time.Duration
}

func newUpstreamLimiter(minInterval time.Duration) *upstreamLimiter {
	if minInterval < 0 {
		minInterval = 0
	}
	return &upstreamLimiter{
		slot:        make(chan struct{}, 1),
		minInterval: minInterval,
	}
}

func (l *upstreamLimiter) acquire(ctx context.Context) (func(), error) {
	select {
	case l.slot <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	release := func() { <-l.slot }

	l.mu.Lock()
	wait := time.Until(l.nextStartAt)
	l.mu.Unlock()

	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			release()
			return nil, ctx.Err()
		}
	}

	l.mu.Lock()
	l.nextStartAt = time.Now().Add(l.minInterval)
	l.mu.Unlock()

	return release, nil
}

func (l *upstreamLimiter) cooldown(delay time.Duration) {
	if l == nil || delay <= 0 {
		return
	}

	until := time.Now().Add(delay)
	l.mu.Lock()
	if until.After(l.nextStartAt) {
		l.nextStartAt = until
	}
	l.mu.Unlock()
}

type invoiceQueryCacheEntry struct {
	expiresAt time.Time
	size      int
	result    *model.InvoiceQueryResult
}

type invoiceQueryCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]invoiceQueryCacheEntry
}

func newInvoiceQueryCache(ttl time.Duration) *invoiceQueryCache {
	return &invoiceQueryCache{
		ttl:   ttl,
		items: make(map[string]invoiceQueryCacheEntry),
	}
}

func (c *invoiceQueryCache) get(key string, requestedSize int) (*model.InvoiceQueryResult, bool) {
	if c == nil || c.ttl <= 0 {
		return nil, false
	}

	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.After(now) {
		delete(c.items, key)
		return nil, false
	}
	if entry.size < requestedSize {
		return nil, false
	}

	return cloneInvoiceQueryResult(entry.result), true
}

func (c *invoiceQueryCache) set(key string, size int, result *model.InvoiceQueryResult) {
	if c == nil || c.ttl <= 0 || result == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	previous, ok := c.items[key]
	if ok && previous.expiresAt.After(time.Now()) && previous.size > size {
		return
	}

	c.items[key] = invoiceQueryCacheEntry{
		expiresAt: time.Now().Add(c.ttl),
		size:      size,
		result:    cloneInvoiceQueryResult(result),
	}

	if len(c.items) > 512 {
		now := time.Now()
		for currentKey, entry := range c.items {
			if !entry.expiresAt.After(now) {
				delete(c.items, currentKey)
			}
		}
	}
}

func cloneInvoiceQueryResult(source *model.InvoiceQueryResult) *model.InvoiceQueryResult {
	if source == nil {
		return nil
	}

	cloned := *source
	cloned.Datas = append([]json.RawMessage(nil), source.Datas...)
	cloned.FailedRanges = append([]model.InvoiceQueryFailedRange(nil), source.FailedRanges...)
	if source.Pagination != nil {
		pagination := *source.Pagination
		cloned.Pagination = &pagination
	}
	return &cloned
}

func (s *service) queryInvoicesUpstream(
	ctx context.Context,
	token string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	opts model.QueryOptions,
) (*model.InvoiceQueryResult, error) {
	if opts.Size <= 0 {
		opts.Size = model.MaxInvoiceQuerySize
	}
	if opts.Size > model.MaxInvoiceQuerySize {
		opts.Size = model.MaxInvoiceQuerySize
	}

	cacheKey := invoiceQueryCacheKey(token, channel, direction, opts)
	if cached, ok := s.queryCache.get(cacheKey, opts.Size); ok {
		return cached, nil
	}

	var lastErr error
	for attempt := 0; attempt <= s.rateLimitRetries; attempt++ {
		release, err := s.upstreamLimiter.acquire(ctx)
		if err != nil {
			return nil, err
		}

		result, requestErr, delay := func() (*model.InvoiceQueryResult, error, time.Duration) {
			defer release()
			result, requestErr := s.client.QueryInvoices(
				ctx,
				token,
				channel,
				direction,
				opts,
			)
			if !isRateLimitError(requestErr) {
				return result, requestErr, 0
			}

			// Đặt global cooldown trước khi nhả slot để request đang chờ phía sau
			// không lọt vào khe giữa response 429 và lúc cập nhật backoff.
			delay := s.rateLimitDelay(requestErr, attempt)
			s.upstreamLimiter.cooldown(delay)
			return result, requestErr, delay
		}()

		if requestErr == nil {
			s.queryCache.set(cacheKey, opts.Size, result)
			return result, nil
		}

		lastErr = requestErr
		if !isRateLimitError(requestErr) || attempt >= s.rateLimitRetries {
			return nil, requestErr
		}

		slog.Warn(
			"HDDT GDT rate limited invoice query",
			"channel", channel,
			"direction", direction,
			"attempt", attempt+1,
			"retry_after", delay,
		)
		if err := waitContext(ctx, delay); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

func (s *service) exportInvoicesUpstream(
	ctx context.Context,
	token string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	opts model.ExportOptions,
) (*model.File, error) {
	var lastErr error
	for attempt := 0; attempt <= s.rateLimitRetries; attempt++ {
		release, err := s.upstreamLimiter.acquire(ctx)
		if err != nil {
			return nil, err
		}

		result, requestErr, delay := func() (*model.File, error, time.Duration) {
			defer release()
			result, requestErr := s.client.ExportInvoices(
				ctx,
				token,
				channel,
				direction,
				opts,
			)
			if !isRateLimitError(requestErr) {
				return result, requestErr, 0
			}

			delay := s.rateLimitDelay(requestErr, attempt)
			s.upstreamLimiter.cooldown(delay)
			return result, requestErr, delay
		}()

		if requestErr == nil {
			return result, nil
		}

		lastErr = requestErr
		if !isRateLimitError(requestErr) || attempt >= s.rateLimitRetries {
			return nil, requestErr
		}

		slog.Warn(
			"HDDT GDT rate limited export",
			"channel", channel,
			"direction", direction,
			"attempt", attempt+1,
			"retry_after", delay,
		)
		if err := waitContext(ctx, delay); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

func invoiceQueryCacheKey(
	token string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	opts model.QueryOptions,
) string {
	tokenHash := sha256.Sum256([]byte(token))
	search, _ := moduleutils.BuildSearch(opts.From, opts.To, opts.Filter)
	return fmt.Sprintf(
		"%x|%s|%s|%s",
		tokenHash,
		channel,
		direction,
		search,
	)
}

func isRateLimitError(err error) bool {
	var httpErr *corehttp.HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusTooManyRequests
}

func (s *service) rateLimitDelay(err error, attempt int) time.Duration {
	delay := s.rateLimitBaseDelay
	if delay <= 0 {
		delay = time.Second
	}

	for index := 0; index < attempt && delay < maxRateLimitBackoff; index++ {
		delay *= 2
	}
	if delay > maxRateLimitBackoff {
		delay = maxRateLimitBackoff
	}

	var httpErr *corehttp.HTTPError
	if errors.As(err, &httpErr) && httpErr.RetryAfter > delay {
		delay = httpErr.RetryAfter
	}
	return delay
}

func waitContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
