package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/yunotools/eif/internal/core/apperr"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/model"
	moduleutils "github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/utils"
)

const (
	wrapperQueryPageSize = 50
	wrapperMaxPage       = 100
)

func (s *service) QueryInvoiceSold(
	ctx context.Context,
	sessionID string,
	query *dto.HoaDonQuery,
) (*model.InvoiceQueryResult, error) {
	return s.queryInvoices(
		ctx,
		sessionID,
		model.InvoiceChannelStandard,
		model.InvoiceDirectionSold,
		query,
	)
}

func (s *service) QueryInvoicePurchase(
	ctx context.Context,
	sessionID string,
	query *dto.HoaDonQuery,
) (*model.InvoiceQueryResult, error) {
	return s.queryInvoices(
		ctx,
		sessionID,
		model.InvoiceChannelStandard,
		model.InvoiceDirectionPurchase,
		query,
	)
}

func (s *service) QueryScoInvoiceSold(
	ctx context.Context,
	sessionID string,
	query *dto.HoaDonQuery,
) (*model.InvoiceQueryResult, error) {
	return s.queryInvoices(
		ctx,
		sessionID,
		model.InvoiceChannelSCO,
		model.InvoiceDirectionSold,
		query,
	)
}

func (s *service) QueryScoInvoicePurchase(
	ctx context.Context,
	sessionID string,
	query *dto.HoaDonQuery,
) (*model.InvoiceQueryResult, error) {
	return s.queryInvoices(
		ctx,
		sessionID,
		model.InvoiceChannelSCO,
		model.InvoiceDirectionPurchase,
		query,
	)
}

// QueryInvoiceSoldWrapper tổng hợp hóa đơn bán ra thường + hóa đơn bán ra MTT.
func (s *service) QueryInvoiceSoldWrapper(
	ctx context.Context,
	sessionID string,
	query *dto.HoaDonQuery,
) (*model.InvoiceQueryResult, error) {
	return s.queryInvoiceWrapper(
		ctx,
		sessionID,
		model.InvoiceDirectionSold,
		query,
	)
}

// QueryInvoicePurchaseWrapper tổng hợp hóa đơn mua vào thường + hóa đơn mua vào MTT.
func (s *service) QueryInvoicePurchaseWrapper(
	ctx context.Context,
	sessionID string,
	query *dto.HoaDonQuery,
) (*model.InvoiceQueryResult, error) {
	return s.queryInvoiceWrapper(
		ctx,
		sessionID,
		model.InvoiceDirectionPurchase,
		query,
	)
}

type preparedInvoiceQuery struct {
	fromDate string
	toDate   string
	ranges   []moduleutils.DateRange
	filter   model.InvoiceFilter
	size     int
}

type wrapperQueryResult struct {
	channel   model.InvoiceChannel
	result    *model.InvoiceQueryResult
	truncated bool
	err       error
}

type adaptiveRangeResult struct {
	datas     []json.RawMessage
	total     int
	time      int
	state     any
	truncated bool
}

func (s *service) queryInvoiceWrapper(
	ctx context.Context,
	sessionID string,
	direction model.InvoiceDirection,
	query *dto.HoaDonQuery,
) (*model.InvoiceQueryResult, error) {
	if query == nil {
		return nil, apperr.New(apperr.CodeInvalidRequest, nil)
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	if page > wrapperMaxPage {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			fmt.Errorf("page must be between 1 and %d", wrapperMaxPage),
		)
	}

	prepared, err := s.prepareInvoiceQuery(query)
	if err != nil {
		return nil, err
	}

	// Session/token chỉ resolve một lần cho toàn bộ wrapper request.
	token, err := s.getTokenFromSession(sessionID)
	if err != nil {
		return nil, err
	}

	targetCount := page * wrapperQueryPageSize
	channels := []model.InvoiceChannel{
		model.InvoiceChannelStandard,
		model.InvoiceChannelSCO,
	}
	results := make([]wrapperQueryResult, 0, len(channels))

	// Upstream HDDTGDT rate-limit khá chặt. Không fan-out đồng thời Standard/SCO;
	// limiter chung còn đảm bảo các request từ request khác cũng không burst.
	for _, channel := range channels {
		result, truncated, queryErr := s.queryInvoicesWindowWithToken(
			ctx,
			token,
			channel,
			direction,
			prepared,
			targetCount,
		)
		results = append(results, wrapperQueryResult{
			channel:   channel,
			result:    result,
			truncated: truncated,
			err:       queryErr,
		})
	}

	merged := &model.InvoiceQueryResult{
		FromDate:     query.FromDate,
		ToDate:       query.ToDate,
		FailedRanges: make([]model.InvoiceQueryFailedRange, 0),
		Datas:        make([]json.RawMessage, 0, targetCount*len(channels)),
	}

	successfulSources := 0
	paginationTruncated := false
	var firstErr error
	for _, source := range results {
		if source.truncated {
			paginationTruncated = true
		}
		if source.err != nil {
			if firstErr == nil {
				firstErr = source.err
			}
			merged.FailedRanges = appendUniqueFailedRange(
				merged.FailedRanges,
				model.InvoiceQueryFailedRange{
					FromDate: query.FromDate,
					ToDate:   query.ToDate,
				},
			)
			slog.Warn(
				"HDDT GDT wrapper source failed",
				"channel", source.channel,
				"direction", direction,
				"from", query.FromDate,
				"to", query.ToDate,
				"page", page,
				"error", source.err,
			)
			continue
		}
		if source.result == nil {
			continue
		}

		successfulSources++
		merged.Datas = append(merged.Datas, source.result.Datas...)
		merged.Total += source.result.Total
		merged.Time += source.result.Time
		if merged.State == nil {
			merged.State = source.result.State
		}
		for _, failedRange := range source.result.FailedRanges {
			merged.FailedRanges = appendUniqueFailedRange(
				merged.FailedRanges,
				failedRange,
			)
		}
	}

	if successfulSources == 0 && firstErr != nil {
		return nil, firstErr
	}

	sortInvoiceDatasByCreatedAt(merged.Datas)

	start := (page - 1) * wrapperQueryPageSize
	end := start + wrapperQueryPageSize
	if start >= len(merged.Datas) {
		merged.Datas = make([]json.RawMessage, 0)
	} else {
		if end > len(merged.Datas) {
			end = len(merged.Datas)
		}
		merged.Datas = merged.Datas[start:end]
	}

	actualTotalPages := 0
	if merged.Total > 0 {
		actualTotalPages = (merged.Total + wrapperQueryPageSize - 1) / wrapperQueryPageSize
	}
	totalPages := actualTotalPages
	if totalPages > wrapperMaxPage {
		totalPages = wrapperMaxPage
		paginationTruncated = true
	}

	merged.Pagination = &model.InvoicePagination{
		Page:        page,
		PageSize:    wrapperQueryPageSize,
		TotalPages:  totalPages,
		HasPrevious: page > 1,
		HasNext:     page < totalPages,
		Truncated:   paginationTruncated,
	}

	return merged, nil
}

func (s *service) queryInvoices(
	ctx context.Context,
	sessionID string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	query *dto.HoaDonQuery,
) (*model.InvoiceQueryResult, error) {
	prepared, err := s.prepareInvoiceQuery(query)
	if err != nil {
		return nil, err
	}

	token, err := s.getTokenFromSession(sessionID)
	if err != nil {
		return nil, err
	}

	return s.queryInvoicesWithToken(
		ctx,
		token,
		channel,
		direction,
		prepared,
	)
}

func (s *service) prepareInvoiceQuery(query *dto.HoaDonQuery) (*preparedInvoiceQuery, error) {
	if query == nil {
		return nil, apperr.New(apperr.CodeInvalidRequest, nil)
	}

	from, to, err := moduleutils.ParseDateRange(query.FromDate, query.ToDate)
	if err != nil {
		return nil, apperr.New(apperr.CodeInvalidRequest, err)
	}

	filter := mapToFilter(query.InvoiceFilter)
	if _, err := moduleutils.BuildSearch(from, to, filter); err != nil {
		return nil, apperr.New(apperr.CodeInvalidRequest, err)
	}

	size := query.Size
	if size <= 0 {
		size = wrapperQueryPageSize
	}
	if size > model.MaxInvoiceQuerySize {
		size = model.MaxInvoiceQuerySize
	}

	return &preparedInvoiceQuery{
		fromDate: query.FromDate,
		toDate:   query.ToDate,
		ranges: moduleutils.SplitDateRangeDescending(
			from,
			to,
			s.maxQueryDays,
		),
		filter: filter,
		size:   size,
	}, nil
}

// queryInvoicesWithToken giữ behavior endpoint legacy: mỗi range gọi một lần,
// nhưng tuyệt đối không vượt giới hạn size=50 của HDDTGDT.
func (s *service) queryInvoicesWithToken(
	ctx context.Context,
	token string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	prepared *preparedInvoiceQuery,
) (*model.InvoiceQueryResult, error) {
	if prepared == nil {
		return nil, apperr.New(apperr.CodeInvalidRequest, nil)
	}

	merged := &model.InvoiceQueryResult{
		FromDate:     prepared.fromDate,
		ToDate:       prepared.toDate,
		FailedRanges: make([]model.InvoiceQueryFailedRange, 0),
		Datas:        make([]json.RawMessage, 0),
	}
	successfulRequests := 0
	failedRequests := 0
	var lastErr error

	for _, dateRange := range prepared.ranges {
		response, err := s.queryInvoicesUpstream(
			ctx,
			token,
			channel,
			direction,
			model.QueryOptions{
				From:   dateRange.From,
				To:     dateRange.To,
				Size:   prepared.size,
				Filter: prepared.filter,
			},
		)
		if err != nil {
			failedRequests++
			lastErr = err
			merged.FailedRanges = append(
				merged.FailedRanges,
				model.InvoiceQueryFailedRange{
					FromDate: moduleutils.FormatInputDate(dateRange.From),
					ToDate:   moduleutils.FormatInputDate(dateRange.To),
				},
			)

			slog.Error(
				"HDDT GDT invoice request failed",
				"channel", channel,
				"direction", direction,
				"from", dateRange.From,
				"to", dateRange.To,
				"error", err,
			)

			if ctx.Err() != nil {
				return nil, s.mapToAppError(err)
			}
			continue
		}

		successfulRequests++
		merged.Datas = append(merged.Datas, response.Datas...)
		merged.Total += response.Total
		merged.Time += response.Time
		if merged.State == nil {
			merged.State = response.State
		}
	}

	if successfulRequests == 0 && lastErr != nil {
		return nil, s.mapToAppError(lastErr)
	}

	if failedRequests > 0 {
		slog.Warn(
			"HDDT GDT invoice query completed with failed ranges",
			"channel", channel,
			"direction", direction,
			"total_ranges", len(prepared.ranges),
			"successful_requests", successfulRequests,
			"failed_requests", failedRequests,
		)
	}

	return merged, nil
}

// queryInvoicesWindowWithToken lấy đủ top N record cho một channel mà vẫn giữ
// mọi request HDDTGDT ở size<=50. Nếu một range có >50 record, range được chia nhỏ
// theo thời gian cho tới khi có thể lấy chính xác phần dữ liệu cần thiết.
func (s *service) queryInvoicesWindowWithToken(
	ctx context.Context,
	token string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	prepared *preparedInvoiceQuery,
	targetCount int,
) (*model.InvoiceQueryResult, bool, error) {
	if prepared == nil || targetCount <= 0 {
		return nil, false, apperr.New(apperr.CodeInvalidRequest, nil)
	}

	merged := &model.InvoiceQueryResult{
		FromDate:     prepared.fromDate,
		ToDate:       prepared.toDate,
		FailedRanges: make([]model.InvoiceQueryFailedRange, 0),
		Datas:        make([]json.RawMessage, 0, targetCount),
	}

	successfulRequests := 0
	failedRequests := 0
	truncated := false
	var lastErr error

	for _, dateRange := range prepared.ranges {
		need := targetCount - len(merged.Datas)
		if need < 0 {
			need = 0
		}

		rangeResult, err := s.queryAdaptiveRange(
			ctx,
			token,
			channel,
			direction,
			dateRange,
			prepared.filter,
			need,
		)
		if err != nil {
			failedRequests++
			lastErr = err
			merged.FailedRanges = appendUniqueFailedRange(
				merged.FailedRanges,
				model.InvoiceQueryFailedRange{
					FromDate: moduleutils.FormatInputDate(dateRange.From),
					ToDate:   moduleutils.FormatInputDate(dateRange.To),
				},
			)

			slog.Error(
				"HDDT GDT adaptive invoice range failed",
				"channel", channel,
				"direction", direction,
				"from", dateRange.From,
				"to", dateRange.To,
				"error", err,
			)

			if ctx.Err() != nil {
				return nil, truncated, s.mapToAppError(err)
			}
			continue
		}

		successfulRequests++
		merged.Datas = append(merged.Datas, rangeResult.datas...)
		merged.Total += rangeResult.total
		merged.Time += rangeResult.time
		if merged.State == nil {
			merged.State = rangeResult.state
		}
		if rangeResult.truncated {
			truncated = true
		}
	}

	if successfulRequests == 0 && lastErr != nil {
		return nil, truncated, s.mapToAppError(lastErr)
	}

	if failedRequests > 0 {
		slog.Warn(
			"HDDT GDT adaptive invoice query completed with failed ranges",
			"channel", channel,
			"direction", direction,
			"total_ranges", len(prepared.ranges),
			"successful_requests", successfulRequests,
			"failed_requests", failedRequests,
		)
	}

	if len(merged.Datas) > targetCount {
		merged.Datas = merged.Datas[:targetCount]
	}
	return merged, truncated, nil
}

func (s *service) queryAdaptiveRange(
	ctx context.Context,
	token string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	dateRange moduleutils.DateRange,
	filter model.InvoiceFilter,
	need int,
) (*adaptiveRangeResult, error) {
	requestSize := model.MaxInvoiceQuerySize
	if need <= 0 {
		// Chỉ cần metadata total cho range cũ hơn.
		requestSize = 1
	}

	response, err := s.queryInvoicesUpstream(
		ctx,
		token,
		channel,
		direction,
		model.QueryOptions{
			From:   dateRange.From,
			To:     dateRange.To,
			Size:   requestSize,
			Filter: filter,
		},
	)
	if err != nil {
		return nil, err
	}

	result := &adaptiveRangeResult{
		total: response.Total,
		time:  response.Time,
		state: response.State,
	}

	if need <= 0 || response.Total <= 0 {
		return result, nil
	}

	// HDDTGDT trả tối đa 50 record. Nếu ta chỉ cần <=50 thì top 50 của chính
	// range hiện tại đã đủ và không cần chia nhỏ thêm.
	if need <= model.MaxInvoiceQuerySize || response.Total <= model.MaxInvoiceQuerySize {
		take := need
		if take > len(response.Datas) {
			take = len(response.Datas)
		}
		result.datas = append(result.datas, response.Datas[:take]...)
		return result, nil
	}

	newer, older, ok := splitAdaptiveRange(dateRange)
	if !ok {
		// Có >50 record cùng đúng một giây nên API công khai không thể cursor
		// sâu hơn. Trả phần chắc chắn có và đánh dấu truncated thay vì loop vô hạn.
		take := need
		if take > len(response.Datas) {
			take = len(response.Datas)
		}
		result.datas = append(result.datas, response.Datas[:take]...)
		result.truncated = response.Total > len(response.Datas)
		return result, nil
	}

	newerResult, err := s.queryAdaptiveRange(
		ctx,
		token,
		channel,
		direction,
		newer,
		filter,
		need,
	)
	if err != nil {
		return nil, err
	}
	result.datas = append(result.datas, newerResult.datas...)
	result.time += newerResult.time
	result.truncated = result.truncated || newerResult.truncated

	remaining := need - len(result.datas)
	if remaining > 0 {
		olderResult, err := s.queryAdaptiveRange(
			ctx,
			token,
			channel,
			direction,
			older,
			filter,
			remaining,
		)
		if err != nil {
			return nil, err
		}
		result.datas = append(result.datas, olderResult.datas...)
		result.time += olderResult.time
		result.truncated = result.truncated || olderResult.truncated
		if result.state == nil {
			result.state = olderResult.state
		}
	}

	if len(result.datas) > need {
		result.datas = result.datas[:need]
	}
	return result, nil
}

func splitAdaptiveRange(value moduleutils.DateRange) (
	newer moduleutils.DateRange,
	older moduleutils.DateRange,
	ok bool,
) {
	from := value.From.Truncate(time.Second)
	to := value.To.Truncate(time.Second)
	if !to.After(from) {
		return moduleutils.DateRange{}, moduleutils.DateRange{}, false
	}

	seconds := int64(to.Sub(from) / time.Second)
	midpoint := from.Add(time.Duration(seconds/2) * time.Second)
	newerFrom := midpoint.Add(time.Second)
	if newerFrom.After(to) {
		return moduleutils.DateRange{}, moduleutils.DateRange{}, false
	}

	return moduleutils.DateRange{
			From: newerFrom,
			To:   to,
		}, moduleutils.DateRange{
			From: from,
			To:   midpoint,
		}, true
}

func appendUniqueFailedRange(
	ranges []model.InvoiceQueryFailedRange,
	value model.InvoiceQueryFailedRange,
) []model.InvoiceQueryFailedRange {
	for _, current := range ranges {
		if current.FromDate == value.FromDate && current.ToDate == value.ToDate {
			return ranges
		}
	}
	return append(ranges, value)
}

func sortInvoiceDatasByCreatedAt(datas []json.RawMessage) {
	sort.SliceStable(datas, func(i, j int) bool {
		left, leftOK := invoiceCreatedAt(datas[i])
		right, rightOK := invoiceCreatedAt(datas[j])
		if leftOK != rightOK {
			return leftOK
		}
		if !leftOK {
			return false
		}
		return left.After(right)
	})
}

func invoiceCreatedAt(raw json.RawMessage) (time.Time, bool) {
	var value struct {
		CreatedAt string `json:"tdlap"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return time.Time{}, false
	}

	input := strings.TrimSpace(value.CreatedAt)
	if input == "" {
		return time.Time{}, false
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"02/01/2006T15:04:05",
		"02/01/2006 15:04:05",
		"02/01/2006",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, input); err == nil {
			return parsed, true
		}
	}

	return time.Time{}, false
}
