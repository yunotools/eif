package hddtgdt

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/yunotools/eif/internal/core/config"
	corehttp "github.com/yunotools/eif/internal/core/protocol/httpclient"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/client"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/handler"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/service"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/session"
)

type Module struct {
	handler *handler.Handler
}

func New(
	httpClient *corehttp.Client,
	cfg config.HDDTGDTConfig,
) (
	*Module,
	error,
) {
	upstreamClient := client.New(httpClient, cfg.Endpoint)
	sessionManager, err := session.NewManager(
		cfg.SessionSkew,
		cfg.SessionStorePath,
		cfg.SessionEncryptionKey,
	)
	if err != nil {
		return nil, err
	}

	slog.Info(
		"HDDT GDT session store loaded",
		"sessions", sessionManager.Count(),
		"store_path", cfg.SessionStorePath,
	)

	svc := service.New(
		upstreamClient,
		sessionManager,
		cfg.MaxQueryDays,
		cfg.MaxExportDays,
		cfg.MinRequestInterval,
		cfg.RateLimitRetries,
		cfg.RateLimitBaseDelay,
		cfg.QueryCacheTTL,
	)
	return &Module{
		handler: handler.New(svc),
	}, nil
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/module/hoadondientu.gdt.gov.vn")
	group.GET("/captcha", m.handler.GetCaptcha)
	group.POST("/authenticate", m.handler.Authenticate)
	group.GET("/session", m.handler.GetSession)
	group.POST("/session/refresh", m.handler.RefreshSession)
	group.DELETE("/session", m.handler.DeleteSession)

	group.POST("/invoice/sold", m.handler.QueryInvoiceSold)
	group.POST("/invoice/purchase", m.handler.QueryInvoicePurchase)
	group.POST("/invoice/sco/sold", m.handler.QueryScoInvoiceSold)
	group.POST("/invoice/sco/purchase", m.handler.QueryScoInvoicePurchase)
	group.POST("/invoice/export", m.handler.ExportInvoice)
	group.POST("/invoice/export/merged", m.handler.ExportInvoiceMerged)

	group.POST("/invoice/wrapper/sold", m.handler.QueryInvoiceSoldWrapper)
	group.POST("/invoice/wrapper/purchase", m.handler.QueryInvoicePurchaseWrapper)
	group.POST("/invoice/wrapper/sold/export", m.handler.ExportInvoiceSoldWrapper)
	group.POST("/invoice/wrapper/purchase/export", m.handler.ExportInvoicePurchaseWrapper)
}
