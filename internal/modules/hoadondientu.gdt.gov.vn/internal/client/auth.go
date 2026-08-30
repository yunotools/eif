package client

import (
	"context"

	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
)

func (c *hddtgdtClient) GetCaptcha(ctx context.Context) (
	*dto.CaptchaResponse, error,
) {
	var response dto.CaptchaResponse
	if err := c.httpClient.GetJSON(
		ctx,
		c.endpoint+"/api/captcha",
		acceptHeaders(),
		&response,
	); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *hddtgdtClient) Authenticate(
	ctx context.Context,
	req *dto.AuthenticationRequest,
) (*dto.AuthenticationTokenResponse, error) {
	var response dto.AuthenticationTokenResponse
	if err := c.httpClient.PostJSON(
		ctx,
		c.endpoint+"/api/security-taxpayer/authenticate",
		acceptHeaders(),
		req,
		&response,
	); err != nil {
		return nil, err
	}

	return &response, nil
}
