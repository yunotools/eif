package service

import (
	"context"
	"log/slog"

	"github.com/yunotools/eif/internal/core/apperr"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/model"
	moduleutils "github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/utils"
)

func (s *service) QueryInvoiceSold(
	ctx context.Context,
	sessionID string,
	query *dto.HoaDonQuery,
) (
	*model.InvoiceQueryResult,
	error,
) {
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
) (
	*model.InvoiceQueryResult,
	error,
) {
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
) (
	*model.InvoiceQueryResult,
	error,
) {
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
	query *dto.HoaDonQuery) (
	*model.InvoiceQueryResult,
	error,
) {
	return s.queryInvoices(
		ctx,
		sessionID,
		model.InvoiceChannelSCO,
		model.InvoiceDirectionPurchase,
		query,
	)
}

func (s *service) queryInvoices(
	ctx context.Context,
	sessionID string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	query *dto.HoaDonQuery) (
	*model.InvoiceQueryResult,
	error,
) {
	if query == nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			nil,
		)
	}

	token, err := s.getTokenFromSession(sessionID)
	if err != nil {
		return nil, err
	}

	from, to, err := moduleutils.ParseDateRange(
		query.FromDate,
		query.ToDate,
	)
	if err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	filter := mapToFilter(query.InvoiceFilter)
	if _, err := moduleutils.BuildSearch(
		from,
		to,
		filter,
	); err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	ranges := moduleutils.SplitDateRangeDescending(
		from,
		to,
		s.maxQueryDays,
	)
	merged := &model.InvoiceQueryResult{}
	successfulRequests := 0
	failedRequests := 0
	var lastErr error
	for _, dateRange := range ranges {
		response, err := s.client.QueryInvoices(
			ctx,
			token,
			channel,
			direction,
			model.QueryOptions{
				From:   dateRange.From,
				To:     dateRange.To,
				Size:   query.Size,
				Filter: filter,
			})
		if err != nil {
			failedRequests++
			lastErr = err

			slog.Error(
				"HDDT GDT invoice request failed",
				"channel", channel,
				"direction", direction,
				"from", dateRange.From,
				"to", dateRange.To,
				"error", err,
			)

			// Request phía client đã bị hủy thì
			// không tiếp tục gọi HDDTGDT các range còn lại.
			if ctx.Err() != nil {
				return nil, s.mapToAppError(err)
			}

			// Một range lỗi không làm dừng toàn bộ query,
			// tiếp tục xử lý range tiếp theo.
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

	// Tất cả range đều lỗi thì vẫn trả lỗi thay vì trả response rỗng 200.
	if successfulRequests == 0 && lastErr != nil {
		return nil, s.mapToAppError(lastErr)
	}

	if failedRequests > 0 {
		slog.Warn(
			"HDDT GDT invoice query completed with failed ranges",
			"channel", channel,
			"direction", direction,
			"total_ranges", len(ranges),
			"successful_requests", successfulRequests,
			"failed_requests", failedRequests,
		)
	}

	return merged, nil
}
