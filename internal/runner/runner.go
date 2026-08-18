package runner

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"xray-geocheck/internal/binaries"
	"xray-geocheck/internal/checker"
	"xray-geocheck/internal/config"
	"xray-geocheck/internal/metrics"
	"xray-geocheck/internal/model"
	"xray-geocheck/internal/store"
	"xray-geocheck/internal/subscription"
	"xray-geocheck/internal/xray"
)

type Runner struct {
	cfg      *config.Config
	bins     binaries.Paths
	store    *store.Store
	met      *metrics.Collector
	xray     *xray.Instance
	hostMu   sync.Map
	refreshN sync.Mutex
	syncMu   sync.Mutex
}

func New(cfg *config.Config, bins binaries.Paths, st *store.Store, met *metrics.Collector) *Runner {
	return &Runner{
		cfg:   cfg,
		bins:  bins,
		store: st,
		met:   met,
		xray:  xray.New(bins.Xray, cfg.BinDir, cfg.SocksPortBase),
	}
}

func (r *Runner) Stop() {
	r.xray.Stop()
}

func (r *Runner) StartLoops(ctx context.Context) {
	if err := r.sync(ctx); err != nil {
		slog.Error("subscription sync failed", "err", err)
		r.recordSyncError(err)
	} else {
		r.pingAll(ctx)
	}
	go r.geoOnce(ctx)
	go r.subLoop(ctx)
	go r.pingLoop(ctx)
	go r.geoLoop(ctx)
}

func (r *Runner) pingLoop(ctx context.Context) {
	t := time.NewTicker(r.cfg.PingInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.pingAll(ctx)
		}
	}
}

func (r *Runner) subLoop(ctx context.Context) {
	t := time.NewTicker(r.cfg.SubInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.sync(ctx); err != nil {
				slog.Error("subscription sync failed", "err", err)
				r.recordSyncError(err)
			}
		}
	}
}

func (r *Runner) geoLoop(ctx context.Context) {
	t := time.NewTicker(r.cfg.GeoInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.geoOnce(ctx)
		}
	}
}

func (r *Runner) recordSyncError(err error) {
	hosts := r.xray.Hosts()
	r.store.SetSubscriptionMeta(model.SubscriptionInfo{
		OK:        false,
		Error:     err.Error(),
		HostCount: len(hosts),
		FetchedAt: time.Now().UTC(),
		UserAgent: r.cfg.UserAgent,
	})
	r.met.Update(r.store.Snapshot())
}

func (r *Runner) sync(ctx context.Context) error {
	r.syncMu.Lock()
	defer r.syncMu.Unlock()
	hosts, err := subscription.Fetch(ctx, r.cfg)
	if err != nil {
		return err
	}
	if err := r.xray.Apply(ctx, hosts); err != nil {
		return err
	}
	ready := r.xray.Hosts()
	r.store.SetSubscription(model.SubscriptionInfo{
		OK:        true,
		HostCount: len(ready),
		FetchedAt: time.Now().UTC(),
		UserAgent: r.cfg.UserAgent,
	}, ready)
	r.met.Update(r.store.Snapshot())
	slog.Info("subscription synced", "hosts", len(ready))
	return nil
}

func (r *Runner) pingAll(ctx context.Context) {
	hosts := r.xray.Hosts()
	if len(hosts) == 0 {
		return
	}
	r.store.SetPingRunning(true)
	defer func() {
		r.store.SetPingRunning(false)
		r.met.Update(r.store.Snapshot())
	}()
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, h := range hosts {
		h := h
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			res := checker.PingGET(ctx, h.SocksAddr(), r.cfg.PingCheckURL)
			r.store.SetPing(h.ID, res)
			if res.Up {
				slog.Info("ping ok", "proxy", h.Remarks, "rtt_ms", res.RTTMs, "status", res.Status)
			} else {
				slog.Warn("ping failed", "proxy", h.Remarks, "err", res.Error, "status", res.Status)
			}
		}()
	}
	wg.Wait()
}

func (r *Runner) lockHost(id string) *sync.Mutex {
	v, _ := r.hostMu.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// RefreshAll re-fetches the subscription immediately, then pings and geochecks every host.
func (r *Runner) RefreshAll(ctx context.Context) {
	if !r.refreshN.TryLock() {
		slog.Info("refresh-all already running")
		return
	}
	defer r.refreshN.Unlock()
	r.store.SetRefreshingAll(true)
	defer r.store.SetRefreshingAll(false)
	if err := r.sync(ctx); err != nil {
		slog.Error("refresh-all subscription sync failed", "err", err)
		r.recordSyncError(err)
	}
	r.pingAll(ctx)
	r.geoOnce(ctx)
}

// RefreshOne runs ping + geocheck for a single host.
func (r *Runner) RefreshOne(ctx context.Context, id string) error {
	var host *model.Host
	for _, h := range r.xray.Hosts() {
		if h.ID == id {
			cp := h
			host = &cp
			break
		}
	}
	if host == nil {
		return errNotFound
	}
	r.store.SetRefreshing(id, true)
	defer r.store.SetRefreshing(id, false)

	mu := r.lockHost(id)
	mu.Lock()
	defer mu.Unlock()

	ping := checker.PingGET(ctx, host.SocksAddr(), r.cfg.PingCheckURL)
	r.store.SetPing(id, ping)
	slog.Info("ping (manual)", "proxy", host.Remarks, "up", ping.Up, "rtt_ms", ping.RTTMs)

	geo := checker.RunGeocheck(ctx, r.bins.Geocheck, host.SocksAddr())
	r.store.SetGeo(id, geo)
	r.met.Update(r.store.Snapshot())
	if geo.OK {
		slog.Info("geocheck (manual) ok", "proxy", host.Remarks)
	} else {
		slog.Warn("geocheck (manual) failed", "proxy", host.Remarks, "err", geo.Error)
	}
	return nil
}

var errNotFound = errors.New("proxy not found")

func IsNotFound(err error) bool {
	return err == errNotFound
}

func (r *Runner) geoOnce(ctx context.Context) {
	hosts := r.xray.Hosts()
	if len(hosts) == 0 {
		return
	}
	r.store.SetGeoRunning(true)
	defer func() {
		r.store.SetGeoRunning(false)
		r.met.Update(r.store.Snapshot())
	}()
	for _, h := range hosts {
		if ctx.Err() != nil {
			return
		}
		mu := r.lockHost(h.ID)
		mu.Lock()
		slog.Info("geocheck start", "proxy", h.Remarks, "socks", h.SocksAddr())
		res := checker.RunGeocheck(ctx, r.bins.Geocheck, h.SocksAddr())
		r.store.SetGeo(h.ID, res)
		r.met.Update(r.store.Snapshot())
		if res.OK {
			slog.Info("geocheck ok", "proxy", h.Remarks)
		} else {
			slog.Warn("geocheck failed", "proxy", h.Remarks, "err", res.Error)
		}
		mu.Unlock()
	}
}
