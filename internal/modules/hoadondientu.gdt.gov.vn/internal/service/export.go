package service

import (
	"context"
	"fmt"

	"github.com/yunotools/eif/internal/core/apperr"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/model"
	moduleutils "github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/utils"
)

func (s *service) ExportInvoice(
	ctx context.Context,
	sessionID string,
	req *dto.ExportInvoiceRequest,
) (
	*model.File,
	error,
) {
	if req == nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			nil,
		)
	}

	token, err := s.getTokenFromSession(sessionID)
	if err != nil {
		return nil, err
	}

	direction, err := getDirectionFromString(req.Type)
	if err != nil {
		return nil, err
	}

	channel := model.InvoiceChannelStandard
	if req.Sco {
		channel = model.InvoiceChannelSCO
	}

	from, to, err := moduleutils.ParseDateRange(
		req.FromDate,
		req.ToDate,
	)
	if err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	if moduleutils.CalculateInclusiveDays(from, to) > s.maxExportDays {
		return nil, apperr.New(
			apperr.CodeExportRangeTooLarge,
			fmt.Errorf(
				"maximum is %d days",
				s.maxExportDays,
			),
		)
	}

	filter := mapToFilter(req.InvoiceFilter)
	if _, err := moduleutils.BuildSearch(from, to, filter); err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	file, err := s.client.ExportInvoices(
		ctx,
		token,
		channel,
		direction,
		model.ExportOptions{
			From:   from,
			To:     to,
			Filter: filter,
		},
	)
	if err != nil {
		return nil, s.mapToAppError(err)
	}

	if file.ContentType == "" {
		file.ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}

	file.Filename = fmt.Sprintf(
		"gdt-%s-%s-%s-%s.xlsx",
		channel,
		direction,
		req.FromDate,
		req.ToDate,
	)

	return file, nil
}
