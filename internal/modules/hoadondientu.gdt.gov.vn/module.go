package hddtgdt

import (
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

func New(httpClient *corehttp.Client, cfg config.HDDTGDTConfig) *Module {
	upstreamClient := client.New(httpClient, cfg.Endpoint)
	sessionManager := session.NewManager(cfg.SessionSkew)
	svc := service.New(upstreamClient, sessionManager, cfg.MaxQueryDays, cfg.MaxExportDays)
	return &Module{
		handler: handler.New(svc),
	}
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/module/hoadondientu.gdt.gov.vn")
	group.GET("/captcha", m.handler.GetCaptcha)
	group.POST("/authenticate", m.handler.Authenticate)
	group.DELETE("/session", m.handler.DeleteSession)

	group.POST("/invoice/sold", m.handler.QueryInvoiceSold)
	group.POST("/invoice/purchase", m.handler.QueryInvoicePurchase)
	group.POST("/invoice/sco/sold", m.handler.QueryScoInvoiceSold)
	group.POST("/invoice/sco/purchase", m.handler.QueryScoInvoicePurchase)
	group.POST("/invoice/export", m.handler.ExportInvoice)
}
