package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xray-geocheck/internal/binaries"
	"xray-geocheck/internal/config"
	"xray-geocheck/internal/httpapi"
	"xray-geocheck/internal/hwid"
	"xray-geocheck/internal/metrics"
	"xray-geocheck/internal/runner"
	"xray-geocheck/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	id, err := hwid.LoadOrCreate(cfg.DataDir)
	if err != nil {
		slog.Error("hwid", "err", err)
		os.Exit(1)
	}
	cfg.HWID = id

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st := store.New()
	met := metrics.New()
	api := httpapi.New(ctx, cfg, st)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	metSrv := &http.Server{
		Addr:              cfg.MetricsAddr(),
		Handler:           api.MetricsHandler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("http listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server", "err", err)
			stop()
		}
	}()
	go func() {
		slog.Info("metrics listening", "addr", cfg.MetricsAddr(), "path", cfg.MetricsBasePath)
		if err := metSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server", "err", err)
			stop()
		}
	}()

	slog.Info("ensuring binaries", "deps", cfg.DepsFile, "bin", cfg.BinDir)
	bins, err := binaries.Ensure(ctx, cfg.DepsFile, cfg.BinDir, cfg.XrayCoreURL, cfg.GeocheckURL)
	if err != nil {
		slog.Error("binaries", "err", err)
		os.Exit(1)
	}
	slog.Info("binaries ready", "xray", bins.Xray, "geocheck", bins.Geocheck)

	run := runner.New(cfg, bins, st, met)
	api.SetRunner(run)
	defer run.Stop()
	run.StartLoops(ctx)

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
	_ = metSrv.Shutdown(shutdown)
	slog.Info("shutdown complete")
}
