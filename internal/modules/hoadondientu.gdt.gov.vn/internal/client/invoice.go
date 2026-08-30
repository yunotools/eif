package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	corehttp "github.com/yunotools/eif/internal/core/protocol/httpclient"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/model"
	moduleutils "github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/utils"
)

func (c *hddtgdtClient) QueryInvoices(
	ctx context.Context,
	token string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	opts model.QueryOptions,
) (
	*model.InvoiceQueryResult,
	error,
) {
	path, err := queryPath(channel, direction)
	if err != nil {
		return nil, err
	}

	search, err := moduleutils.BuildSearch(
		opts.From,
		opts.To,
		opts.Filter,
	)
	if err != nil {
		return nil, err
	}

	size := opts.Size
	if size <= 0 {
		size = 15
	}
	if size > 5000 {
		size = 5000
	}

	params := url.Values{}
	params.Set("sort", "tdlap:desc")
	params.Set("size", strconv.Itoa(size))
	params.Set("search", search)
	requestURL, err := corehttp.BuildURL(c.endpoint, path, params)
	if err != nil {
		return nil, err
	}

	var response model.InvoiceQueryResult
	if err := c.httpClient.GetJSON(
		ctx,
		requestURL,
		authHeaders(token),
		&response,
	); err != nil {
		return nil, err
	}
	return &response, nil
}

func queryPath(channel model.InvoiceChannel, direction model.InvoiceDirection) (string, error) {
	var prefix string

	switch channel {
	case model.InvoiceChannelStandard:
		prefix = "/api/query/invoices/"

	case model.InvoiceChannelSCO:
		prefix = "/api/sco-query/invoices/"

	default:
		return "", fmt.Errorf("unsupported invoice channel %q", channel)
	}

	switch direction {
	case model.InvoiceDirectionSold:
		return prefix + "sold", nil

	case model.InvoiceDirectionPurchase:
		return prefix + "purchase", nil

	default:
		return "", fmt.Errorf("unsupported invoice direction %q", direction)
	}
}
