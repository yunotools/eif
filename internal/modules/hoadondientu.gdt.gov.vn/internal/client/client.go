package client

import corehttp "github.com/yunotools/eif/internal/core/protocol/httpclient"

type hddtgdtClient struct {
	httpClient *corehttp.Client
	endpoint   string
}

func New(httpClient *corehttp.Client, endpoint string) Client {
	return &hddtgdtClient{
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

func acceptHeaders() map[string]string {
	return map[string]string{
		"Accept": "application/json, text/plain, */*",
	}
}

func authHeaders(token string) map[string]string {
	headers := acceptHeaders()
	headers["Authorization"] = "Bearer " + token
	return headers
}
