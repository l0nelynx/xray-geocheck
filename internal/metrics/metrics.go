package metrics

import (
	"encoding/json"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"xray-geocheck/internal/mask"
	"xray-geocheck/internal/model"
)

type Collector struct {
	pingRTT           *prometheus.GaugeVec
	proxyUp           *prometheus.GaugeVec
	reputationRisk    *prometheus.GaugeVec
	reputationFlag    *prometheus.GaugeVec
	consensusPercent  *prometheus.GaugeVec
	geoResult         *prometheus.GaugeVec
	geoError          *prometheus.GaugeVec
	stashAvailable    *prometheus.GaugeVec
	stashRTT          *prometheus.GaugeVec
	aiRTT             *prometheus.GaugeVec
	connectivityScore *prometheus.GaugeVec
	lastSuccess       *prometheus.GaugeVec
}

func New() *Collector {
	ns := "xray_geocheck"
	return &Collector{
		pingRTT: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_ping_rtt_ms", Help: "HTTP GET RTT through the proxy in milliseconds",
		}, []string{"proxy"}),
		proxyUp: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_proxy_up", Help: "1 if the last ping GET succeeded",
		}, []string{"proxy"}),
		reputationRisk: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_reputation_risk", Help: "Exit IP reputation risk score",
		}, []string{"proxy"}),
		reputationFlag: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_reputation_flag", Help: "1 if the named reputation flag is set",
		}, []string{"proxy", "flag"}),
		consensusPercent: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_consensus_percent", Help: "Geolocation consensus percentage",
		}, []string{"proxy", "ip_version", "code"}),
		geoResult: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_geo_result", Help: "1 for a geo/service observation",
		}, []string{"proxy", "group", "id", "ip_version", "code"}),
		geoError: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_geo_error", Help: "1 if a geo provider returned an error",
		}, []string{"proxy", "group", "id", "ip_version"}),
		stashAvailable: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_stash_available", Help: "1 if stash/access check is available",
		}, []string{"proxy", "id"}),
		stashRTT: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_stash_rtt_ms", Help: "Stash/access check RTT",
		}, []string{"proxy", "id"}),
		aiRTT: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_ai_rtt_ms", Help: "AI endpoint RTT",
		}, []string{"proxy", "id"}),
		connectivityScore: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_connectivity_score", Help: "geocheck path connectivity score",
		}, []string{"proxy"}),
		lastSuccess: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: ns + "_last_success_timestamp", Help: "Unix timestamp of last successful check",
		}, []string{"check"}),
	}
}

func (c *Collector) Update(snap model.Snapshot) {
	c.pingRTT.Reset()
	c.proxyUp.Reset()
	c.reputationRisk.Reset()
	c.reputationFlag.Reset()
	c.consensusPercent.Reset()
	c.geoResult.Reset()
	c.geoError.Reset()
	c.stashAvailable.Reset()
	c.stashRTT.Reset()
	c.aiRTT.Reset()
	c.connectivityScore.Reset()

	if snap.LastPingAt != nil {
		c.lastSuccess.WithLabelValues("ping").Set(float64(snap.LastPingAt.Unix()))
	}
	if snap.LastGeoAt != nil {
		c.lastSuccess.WithLabelValues("geo").Set(float64(snap.LastGeoAt.Unix()))
	}

	for _, p := range snap.Proxies {
		name := p.Remarks
		if p.Ping != nil {
			up := 0.0
			if p.Ping.Up {
				up = 1
			}
			c.proxyUp.WithLabelValues(name).Set(up)
			if p.Ping.Up {
				c.pingRTT.WithLabelValues(name).Set(p.Ping.RTTMs)
			}
		}
		if p.Geo != nil && p.Geo.OK && len(p.Geo.Report) > 0 {
			c.applyReport(name, p.Geo.Report)
		}
	}
}

func (c *Collector) applyReport(proxy string, raw json.RawMessage) {
	var r map[string]any
	if err := json.Unmarshal(raw, &r); err != nil {
		return
	}
	if rep, ok := r["reputation"].(map[string]any); ok {
		if risk, ok := asFloat(rep["risk"]); ok {
			c.reputationRisk.WithLabelValues(proxy).Set(risk)
		}
		for _, flag := range []string{"residential", "proxy", "vpn", "tor", "hosting", "scraper", "compromised", "anonymous"} {
			c.reputationFlag.WithLabelValues(proxy, flag).Set(bool01(rep[flag]))
		}
	}
	if cons, ok := r["consensus"].(map[string]any); ok {
		emitConsensus := func(ver string, v any) {
			arr, _ := v.([]any)
			for _, item := range arr {
				m, _ := item.(map[string]any)
				if m == nil {
					continue
				}
				code, _ := m["code"].(string)
				if pct, ok := asFloat(m["percent"]); ok && code != "" {
					c.consensusPercent.WithLabelValues(proxy, ver, code).Set(pct)
				}
			}
		}
		emitConsensus("ipv4", cons["ipv4"])
		emitConsensus("ipv6", cons["ipv6"])
	}
	if geo, ok := r["geo"].(map[string]any); ok {
		for _, group := range []string{"cdn", "geoip", "services"} {
			arr, _ := geo[group].([]any)
			for _, item := range arr {
				m, _ := item.(map[string]any)
				if m == nil {
					continue
				}
				id, _ := m["id"].(string)
				emitEndpoint(c, proxy, group, id, "ipv4", m["ipv4"])
				emitEndpoint(c, proxy, group, id, "ipv6", m["ipv6"])
			}
		}
	}
	if stash, ok := r["stash_checks"].([]any); ok {
		for _, item := range stash {
			m, _ := item.(map[string]any)
			if m == nil {
				continue
			}
			id, _ := m["id"].(string)
			state, _ := m["state"].(string)
			avail := 0.0
			if state == "available" {
				avail = 1
			}
			c.stashAvailable.WithLabelValues(proxy, id).Set(avail)
			if rtt, ok := asFloat(m["rtt_ms"]); ok {
				c.stashRTT.WithLabelValues(proxy, id).Set(rtt)
			}
		}
	}
	if ai, ok := r["ai_endpoints"].([]any); ok {
		for _, item := range ai {
			m, _ := item.(map[string]any)
			if m == nil {
				continue
			}
			id, _ := m["id"].(string)
			if rtt, ok := asFloat(m["rtt_ms"]); ok {
				c.aiRTT.WithLabelValues(proxy, id).Set(rtt)
			}
		}
	}
	if conn, ok := r["connectivity"].(map[string]any); ok {
		if score, ok := asFloat(conn["score"]); ok {
			c.connectivityScore.WithLabelValues(proxy).Set(score)
		}
	}
}

func emitEndpoint(c *Collector, proxy, group, id, ipVer string, v any) {
	m, _ := v.(map[string]any)
	if m == nil {
		return
	}
	if err, _ := m["error"].(string); err != "" {
		c.geoError.WithLabelValues(proxy, group, id, ipVer).Set(1)
		return
	}
	code, _ := m["value"].(string)
	if code == "" {
		return
	}
	c.geoResult.WithLabelValues(proxy, group, id, ipVer, mask.String(code)).Set(1)
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func bool01(v any) float64 {
	b, ok := v.(bool)
	if ok && b {
		return 1
	}
	return 0
}
