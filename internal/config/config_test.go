package config

import (
	"testing"
	"time"
)

func TestMetricsAuthMustBePaired(t *testing.T) {
	t.Setenv("SUBSCRIPTION_URL", "https://example.test/sub")
	t.Setenv("METRICS_USERNAME", "prom")
	t.Setenv("METRICS_PASSWORD", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when only METRICS_USERNAME is set")
	}
}

func TestMetricsBasePathNormalized(t *testing.T) {
	t.Setenv("SUBSCRIPTION_URL", "https://example.test/sub")
	t.Setenv("METRICS_USERNAME", "")
	t.Setenv("METRICS_PASSWORD", "")
	t.Setenv("METRICS_BASE_PATH", "internal/metrics")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MetricsBasePath != "/internal/metrics" {
		t.Fatalf("got %q", cfg.MetricsBasePath)
	}
}

func TestMetricsListenAddr(t *testing.T) {
	t.Setenv("SUBSCRIPTION_URL", "https://example.test/sub")
	t.Setenv("METRICS_USERNAME", "")
	t.Setenv("METRICS_PASSWORD", "")
	t.Setenv("METRICS_HOST", "")
	t.Setenv("METRICS_PORT", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MetricsHost != "0.0.0.0" || cfg.MetricsPort != 3113 {
		t.Fatalf("defaults host=%q port=%d", cfg.MetricsHost, cfg.MetricsPort)
	}
	if cfg.MetricsAddr() != "0.0.0.0:3113" {
		t.Fatalf("addr %q", cfg.MetricsAddr())
	}
}

func TestUIBasePath(t *testing.T) {
	t.Setenv("SUBSCRIPTION_URL", "https://example.test/sub")
	t.Setenv("METRICS_USERNAME", "")
	t.Setenv("METRICS_PASSWORD", "")
	t.Setenv("UI_BASE_PATH", "geocheck/")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UIBasePath != "/geocheck" {
		t.Fatalf("path %q", cfg.UIBasePath)
	}
	if cfg.UIBaseHref() != "/geocheck/" {
		t.Fatalf("href %q", cfg.UIBaseHref())
	}
}

func TestSubscriptionIntervalDefault(t *testing.T) {
	t.Setenv("SUBSCRIPTION_URL", "https://example.test/sub")
	t.Setenv("METRICS_USERNAME", "")
	t.Setenv("METRICS_PASSWORD", "")
	t.Setenv("SUBSCRIPTION_URL_UPDATE_INTERVAL_MINUTES", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SubInterval != 60*time.Minute {
		t.Fatalf("got %s", cfg.SubInterval)
	}
}
