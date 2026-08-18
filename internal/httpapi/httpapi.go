package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"xray-geocheck/internal/config"
	"xray-geocheck/internal/mask"
	"xray-geocheck/internal/model"
	"xray-geocheck/internal/runner"
	"xray-geocheck/internal/store"
	"xray-geocheck/webui"
)

type API struct {
	ctx   context.Context
	cfg   *config.Config
	store *store.Store
	run   atomic.Pointer[runner.Runner]
}

func New(ctx context.Context, cfg *config.Config, st *store.Store) *API {
	return &API{ctx: ctx, cfg: cfg, store: st}
}

func (a *API) SetRunner(r *runner.Runner) {
	a.run.Store(r)
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", a.status)
	mux.HandleFunc("POST /api/refresh", a.refreshAll)
	mux.HandleFunc("POST /api/refresh/proxy", a.refreshOne)
	mux.HandleFunc("GET /metrics", a.metricsMoved)
	mux.Handle("/", spaHandler())
	return mux
}

func (a *API) metricsMoved(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "metrics moved to "+a.cfg.MetricsAddr()+a.cfg.MetricsBasePath, http.StatusNotFound)
}

func (a *API) MetricsHandler() http.Handler {
	mux := http.NewServeMux()
	h := a.metricsHandler()
	mux.Handle("GET "+a.cfg.MetricsBasePath, h)
	mux.Handle("HEAD "+a.cfg.MetricsBasePath, h)
	return mux
}

func (a *API) metricsHandler() http.Handler {
	h := promhttp.Handler()
	if a.cfg.MetricsUsername == "" {
		return h
	}
	user := []byte(a.cfg.MetricsUsername)
	pass := []byte(a.cfg.MetricsPassword)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(u), user) != 1 ||
			subtle.ConstantTimeCompare([]byte(p), pass) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="metrics"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (a *API) status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	snap := redactSnapshot(a.store.Snapshot())
	if err := json.NewEncoder(w).Encode(snap); err != nil {
		slog.Warn("encode status", "err", err)
	}
}

func (a *API) refreshAll(w http.ResponseWriter, r *http.Request) {
	run := a.run.Load()
	if run == nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	go run.RefreshAll(a.ctx)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (a *API) refreshOne(w http.ResponseWriter, r *http.Request) {
	run := a.run.Load()
	if run == nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	found := false
	for _, p := range a.store.Snapshot().Proxies {
		if p.ID == body.ID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	go func(id string) {
		if err := run.RefreshOne(a.ctx, id); err != nil {
			slog.Warn("refresh one", "id", id, "err", err)
		}
	}(body.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func redactSnapshot(snap model.Snapshot) model.Snapshot {
	proxies := make([]model.ProxyStatus, len(snap.Proxies))
	for i, p := range snap.Proxies {
		proxies[i] = p
		proxies[i].Refreshing = p.Refreshing || snap.RefreshingAll
		if p.Ping != nil {
			ping := *p.Ping
			ping.Error = mask.String(ping.Error)
			proxies[i].Ping = &ping
		}
		if p.Geo != nil {
			g := *p.Geo
			g.Error = mask.String(g.Error)
			g.Report = mask.JSON(g.Report)
			proxies[i].Geo = &g
		}
	}
	snap.Proxies = proxies
	return snap
}

func spaHandler() http.Handler {
	sub, err := fs.Sub(webui.Dist, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "status page missing", http.StatusInternalServerError)
		})
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.Contains(strings.TrimPrefix(r.URL.Path, "/"), ".") {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
