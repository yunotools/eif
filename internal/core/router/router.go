package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yunotools/eif/internal/core/buildinfo"
	"github.com/yunotools/eif/internal/core/config"
	"github.com/yunotools/eif/internal/core/middleware"
	coremodule "github.com/yunotools/eif/internal/core/module"
	appweb "github.com/yunotools/eif/web"
)

func New(
	cfg *config.Config,
	registrars ...coremodule.Registrar,
) *gin.Engine {
	r := gin.New()

	r.Use(
		middleware.RequestID(),
		middleware.Logger(),
		middleware.Recover(),
		middleware.CORS(cfg.CORSConfig.AllowedOrigins),
	)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":              "ok",
			"version":             buildinfo.Version,
			"backend_commit":      buildinfo.BackendCommit,
			"frontend_commit":     buildinfo.FrontendCommit,
			"orchestrator_commit": buildinfo.OrchestratorCommit,
		})
	})

	api := r.Group("/api/v1")
	for _, registrar := range registrars {
		registrar.RegisterRoutes(api)
	}

	r.NoRoute(staticFallback(cfg.ServerConfig.StaticDir, appweb.StaticFS()))
	return r
}
