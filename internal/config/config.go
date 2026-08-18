package config

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultUserAgent           = "xray-geocheck"
	DefaultPingCheckURL        = "https://www.gstatic.com/generate_204"
	DefaultGeoIntervalMinutes  = 720
	DefaultPingIntervalMinutes = 5
	DefaultSubIntervalMinutes  = 60
	DefaultListenAddr          = ":8080"
	DefaultMetricsBasePath     = "/metrics"
	DefaultMetricsHost         = "0.0.0.0"
	DefaultMetricsPort         = 3113
	DefaultSocksPortBase       = 11001
	DeviceModel                = "xray-geocheck"
)

// Config is populated from environment variables.
type Config struct {
	SubscriptionURL string
	UserAgent       string
	XrayCoreURL     string
	GeocheckURL     string
	PingCheckURL    string
	GeoInterval     time.Duration
	PingInterval    time.Duration
	SubInterval     time.Duration
	ListenAddr      string
	MetricsHost     string
	MetricsPort     int
	MetricsBasePath string
	MetricsUsername string
	MetricsPassword string
	DepsFile        string
	BinDir          string
	DataDir         string
	SocksPortBase   int
	HWID            string
	DeviceOS        string
	DeviceOSVersion string
}

func Load() (*Config, error) {
	sub := strings.TrimSpace(os.Getenv("SUBSCRIPTION_URL"))
	if sub == "" {
		return nil, fmt.Errorf("SUBSCRIPTION_URL is required")
	}

	geoMin := envInt("GEO_CHECK_INTERVAL_MINUTES", DefaultGeoIntervalMinutes)
	pingMin := envInt("PING_CHECK_INTERVAL_MINUTES", DefaultPingIntervalMinutes)
	subMin := envInt("SUBSCRIPTION_URL_UPDATE_INTERVAL_MINUTES", DefaultSubIntervalMinutes)
	if geoMin < 1 {
		return nil, fmt.Errorf("GEO_CHECK_INTERVAL_MINUTES must be >= 1")
	}
	if pingMin < 1 {
		return nil, fmt.Errorf("PING_CHECK_INTERVAL_MINUTES must be >= 1")
	}
	if subMin < 1 {
		return nil, fmt.Errorf("SUBSCRIPTION_URL_UPDATE_INTERVAL_MINUTES must be >= 1")
	}

	user := strings.TrimSpace(os.Getenv("METRICS_USERNAME"))
	pass := os.Getenv("METRICS_PASSWORD")
	if (user == "") != (pass == "") {
		return nil, fmt.Errorf("METRICS_USERNAME and METRICS_PASSWORD must both be set or both be empty")
	}

	metricsPort := envInt("METRICS_PORT", DefaultMetricsPort)
	if metricsPort < 1 || metricsPort > 65535 {
		return nil, fmt.Errorf("METRICS_PORT must be between 1 and 65535")
	}

	cfg := &Config{
		SubscriptionURL: sub,
		UserAgent:       envOr("USER_AGENT", DefaultUserAgent),
		XrayCoreURL:     strings.TrimSpace(os.Getenv("XRAY_CORE_URL")),
		GeocheckURL:     strings.TrimSpace(os.Getenv("GEOCHECK_URL")),
		PingCheckURL:    envOr("PING_CHECK_URL", DefaultPingCheckURL),
		GeoInterval:     time.Duration(geoMin) * time.Minute,
		PingInterval:    time.Duration(pingMin) * time.Minute,
		SubInterval:     time.Duration(subMin) * time.Minute,
		ListenAddr:      envOr("LISTEN_ADDR", DefaultListenAddr),
		MetricsHost:     envOr("METRICS_HOST", DefaultMetricsHost),
		MetricsPort:     metricsPort,
		MetricsBasePath: normalizePath(envOr("METRICS_BASE_PATH", DefaultMetricsBasePath)),
		MetricsUsername: user,
		MetricsPassword: pass,
		DepsFile:        envOr("DEPS_FILE", "deps.json"),
		BinDir:          envOr("BIN_DIR", "bin"),
		DataDir:         envOr("DATA_DIR", "data"),
		SocksPortBase:   DefaultSocksPortBase,
		DeviceOS:        runtime.GOOS,
		DeviceOSVersion: osVersion(),
	}
	return cfg, nil
}

func (c *Config) MetricsAddr() string {
	return net.JoinHostPort(c.MetricsHost, strconv.Itoa(c.MetricsPort))
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func osVersion() string {
	if b, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
			}
		}
	}
	return runtime.GOARCH
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return DefaultMetricsBasePath
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if p == "/" {
		return DefaultMetricsBasePath
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	return p
}
