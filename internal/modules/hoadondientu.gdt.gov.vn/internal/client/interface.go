package client

import (
	"context"

	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/model"
)

type Client interface {
	GetCaptcha(ctx context.Context) (
		*dto.CaptchaResponse, error,
	)

	Authenticate(
		ctx context.Context,
		req *dto.AuthenticationRequest,
	) (
		*dto.AuthenticationTokenResponse,
		error,
	)

	QueryInvoices(
		ctx context.Context,
		token string,
		channel model.InvoiceChannel,
		direction model.InvoiceDirection,
		opts model.QueryOptions,
	) (
		*model.InvoiceQueryResult,
		error,
	)

	ExportInvoices(
		ctx context.Context,
		token string,
		channel model.InvoiceChannel,
		direction model.InvoiceDirection,
		opts model.ExportOptions,
	) (
		*model.File,
		error,
	)
}
