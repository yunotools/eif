package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yunotools/eif/internal/core/config"
	"github.com/yunotools/eif/internal/core/logger"
	coremodule "github.com/yunotools/eif/internal/core/module"
	corehttp "github.com/yunotools/eif/internal/core/protocol/httpclient"
	"github.com/yunotools/eif/internal/core/router"
	hddtgdt "github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	log := logger.Init(
		cfg.LoggerConfig.Level,
		cfg.LoggerConfig.Format,
	)

	httpClient := corehttp.New(cfg.HDDTGDTConfig.Timeout)
	hddtModule, err := hddtgdt.New(httpClient, cfg.HDDTGDTConfig)
	if err != nil {
		log.Error(
			"initialize HDDT GDT module",
			"error",
			err,
		)
		os.Exit(1)
	}

	registrars := []coremodule.Registrar{hddtModule}
	engine := router.New(cfg, registrars...)

	server := &http.Server{
		Addr:              cfg.ServerConfig.ServerAddress(),
		Handler:           engine,
		ReadHeaderTimeout: cfg.ServerConfig.ReadHeaderTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	sigCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErr:
		if err != nil {
			log.Error(
				"server stopped unexpectedly",
				"error",
				err,
			)
			os.Exit(1)
		}
	case <-sigCtx.Done():
		log.Info("shutdown signal received")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ServerConfig.ShutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error(
			"graceful shutdown failed",
			"error",
			err,
		)
	}

	time.Sleep(10 * time.Millisecond)
}
