package store

import (
	"sync"

	"xray-geocheck/internal/model"
)

// Store holds the latest check snapshot for the status page and metrics.
type Store struct {
	mu         sync.RWMutex
	snap       model.Snapshot
	byID       map[string]model.ProxyStatus
	refreshing map[string]bool
}

func New() *Store {
	return &Store{
		byID:       map[string]model.ProxyStatus{},
		refreshing: map[string]bool{},
		snap:       model.Snapshot{Proxies: []model.ProxyStatus{}},
	}
}

func (s *Store) Snapshot() model.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.snap
	out.Proxies = append([]model.ProxyStatus(nil), s.snap.Proxies...)
	for i := range out.Proxies {
		out.Proxies[i].Refreshing = s.refreshing[out.Proxies[i].ID]
	}
	return out
}

func (s *Store) SetSubscription(info model.SubscriptionInfo, hosts []model.Host) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.Subscription = info
	next := make([]model.ProxyStatus, 0, len(hosts))
	nextMap := make(map[string]model.ProxyStatus, len(hosts))
	for _, h := range hosts {
		prev, ok := s.byID[h.ID]
		ps := model.ProxyStatus{
			ID:        h.ID,
			Remarks:   h.Remarks,
			Address:   h.Address,
			Protocol:  h.Protocol,
			SocksAddr: h.SocksAddr(),
		}
		if ok {
			ps.Ping = prev.Ping
			ps.Geo = prev.Geo
		}
		next = append(next, ps)
		nextMap[h.ID] = ps
	}
	s.byID = nextMap
	s.snap.Proxies = next
}

func (s *Store) SetSubscriptionMeta(info model.SubscriptionInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.Subscription = info
}

func (s *Store) SetPing(id string, ping model.PingResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.byID[id]
	if !ok {
		return
	}
	cp := ping
	ps.Ping = &cp
	s.byID[id] = ps
	t := ping.CheckedAt
	s.snap.LastPingAt = &t
	s.replaceProxy(ps)
}

func (s *Store) SetGeo(id string, geo model.GeoResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.byID[id]
	if !ok {
		return
	}
	cp := geo
	ps.Geo = &cp
	s.byID[id] = ps
	t := geo.CheckedAt
	s.snap.LastGeoAt = &t
	s.replaceProxy(ps)
}

func (s *Store) SetPingRunning(v bool) {
	s.mu.Lock()
	s.snap.PingRunning = v
	s.mu.Unlock()
}

func (s *Store) SetGeoRunning(v bool) {
	s.mu.Lock()
	s.snap.GeoRunning = v
	s.mu.Unlock()
}

func (s *Store) SetRefreshingAll(v bool) {
	s.mu.Lock()
	s.snap.RefreshingAll = v
	s.mu.Unlock()
}

func (s *Store) SetRefreshing(id string, v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v {
		s.refreshing[id] = true
	} else {
		delete(s.refreshing, id)
	}
}

func (s *Store) replaceProxy(ps model.ProxyStatus) {
	for i := range s.snap.Proxies {
		if s.snap.Proxies[i].ID == ps.ID {
			s.snap.Proxies[i] = ps
			return
		}
	}
	s.snap.Proxies = append(s.snap.Proxies, ps)
}
