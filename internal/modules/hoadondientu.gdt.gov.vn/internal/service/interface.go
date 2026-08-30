package service

import (
	"context"

	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/model"
)

type Service interface {
	GetCaptcha(ctx context.Context) (
		*dto.CaptchaResponse,
		error,
	)

	Authenticate(
		ctx context.Context,
		req *dto.AuthenticationRequest,
	) (
		*dto.AuthenticationResponse,
		error,
	)

	DeleteSession(sessionID string)

	QueryInvoiceSold(
		ctx context.Context,
		sessionID string,
		query *dto.HoaDonQuery,
	) (
		*model.InvoiceQueryResult,
		error,
	)

	QueryInvoicePurchase(
		ctx context.Context,
		sessionID string,
		query *dto.HoaDonQuery,
	) (
		*model.InvoiceQueryResult,
		error,
	)

	QueryScoInvoiceSold(
		ctx context.Context,
		sessionID string,
		query *dto.HoaDonQuery,
	) (
		*model.InvoiceQueryResult,
		error,
	)

	QueryScoInvoicePurchase(
		ctx context.Context,
		sessionID string,
		query *dto.HoaDonQuery,
	) (
		*model.InvoiceQueryResult,
		error,
	)

	ExportInvoice(
		ctx context.Context,
		sessionID string,
		req *dto.ExportInvoiceRequest,
	) (
		*model.File,
		error,
	)
}
