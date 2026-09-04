package service

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/yunotools/eif/internal/core/apperr"
	corehttp "github.com/yunotools/eif/internal/core/protocol/httpclient"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/client"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/session"
)

type service struct {
	client             client.Client
	sessions           *session.Manager
	maxQueryDays       int
	maxExportDays      int
	upstreamLimiter    *upstreamLimiter
	rateLimitRetries   int
	rateLimitBaseDelay time.Duration
	queryCache         *invoiceQueryCache
}

func New(
	client client.Client,
	sessions *session.Manager,
	maxQueryDays,
	maxExportDays int,
	minRequestInterval time.Duration,
	rateLimitRetries int,
	rateLimitBaseDelay time.Duration,
	queryCacheTTL time.Duration,
) Service {
	return &service{
		client:             client,
		sessions:           sessions,
		maxQueryDays:       maxQueryDays,
		maxExportDays:      maxExportDays,
		upstreamLimiter:    newUpstreamLimiter(minRequestInterval),
		rateLimitRetries:   rateLimitRetries,
		rateLimitBaseDelay: rateLimitBaseDelay,
		queryCache:         newInvoiceQueryCache(queryCacheTTL),
	}
}

func (s *service) getTokenFromSession(
	sessionID string,
) (
	string,
	error,
) {
	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return "", apperr.New(
			apperr.CodeSessionExpired,
			err,
		)
	}

	return sess.Token, nil
}

func (s *service) mapToAppError(err error) error {
	if err == nil {
		return nil
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return apperr.New(
			apperr.CodeHDDTGDTTimeout,
			err,
		)
	}

	var httpErr *corehttp.HTTPError
	if errors.As(err, &httpErr) {
		if httpErr.StatusCode == http.StatusUnauthorized ||
			httpErr.StatusCode == http.StatusForbidden {
			return apperr.New(apperr.CodeSessionExpired, err)
		}
		if httpErr.StatusCode == http.StatusTooManyRequests {
			return apperr.New(apperr.CodeHDDTGDTRateLimited, err)
		}
		return apperr.New(apperr.CodeHDDTGDTBadGateway, err)
	}

	return apperr.New(apperr.CodeHDDTGDTInvalidResponse, err)
}
