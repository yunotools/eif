package service

import (
	"context"

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

	from, to, err := moduleutils.ParseDateRange(query.FromDate, query.ToDate)
	if err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	filter := mapToFilter(query.InvoiceFilter)
	if _, err := moduleutils.BuildSearch(from, to, filter); err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	ranges := moduleutils.SplitDateRangeDescending(from, to, s.maxQueryDays)
	merged := &model.InvoiceQueryResult{}
	for _, dateRange := range ranges {
		response, err := s.client.QueryInvoices(ctx, token, channel, direction, model.QueryOptions{
			From:   dateRange.From,
			To:     dateRange.To,
			Size:   query.Size,
			Filter: filter,
		})
		if err != nil {
			return nil, s.mapToAppError(err)
		}

		merged.Datas = append(merged.Datas, response.Datas...)
		merged.Total += response.Total
		merged.Time += response.Time
		if merged.State == nil {
			merged.State = response.State
		}
	}

	return merged, nil
}
