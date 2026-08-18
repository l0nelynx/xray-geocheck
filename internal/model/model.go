package model

import (
	"encoding/json"
	"time"
)

// Host is a single subscription proxy plus its local SOCKS5 endpoint.
type Host struct {
	ID        string         `json:"id"`
	Remarks   string         `json:"remarks"`
	Address   string         `json:"address"`
	Protocol  string         `json:"protocol"`
	Outbound  map[string]any `json:"-"`
	SocksHost string         `json:"socksHost"`
	SocksPort int            `json:"socksPort"`
}

func (h Host) SocksAddr() string {
	return h.SocksHost + ":" + itoa(h.SocksPort)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// PingResult is the wall-clock RTT of one HTTP GET through the proxy.
type PingResult struct {
	Up        bool      `json:"up"`
	RTTMs     float64   `json:"rttMs"`
	Status    int       `json:"status"`
	Error     string    `json:"error,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

// GeoResult wraps a geocheck --json document for one proxy.
type GeoResult struct {
	OK        bool            `json:"ok"`
	Error     string          `json:"error,omitempty"`
	CheckedAt time.Time       `json:"checkedAt"`
	Report    json.RawMessage `json:"report,omitempty"`
}

type SubscriptionInfo struct {
	OK        bool      `json:"ok"`
	Error     string    `json:"error,omitempty"`
	HostCount int       `json:"hostCount"`
	FetchedAt time.Time `json:"fetchedAt"`
	UserAgent string    `json:"userAgent"`
}

type ProxyStatus struct {
	ID         string      `json:"id"`
	Remarks    string      `json:"remarks"`
	Address    string      `json:"address"`
	Protocol   string      `json:"protocol"`
	SocksAddr  string      `json:"socksAddr"`
	Ping       *PingResult `json:"ping,omitempty"`
	Geo        *GeoResult  `json:"geo,omitempty"`
	Refreshing bool        `json:"refreshing"`
}

type Snapshot struct {
	Subscription  SubscriptionInfo `json:"subscription"`
	LastPingAt    *time.Time       `json:"lastPingAt"`
	LastGeoAt     *time.Time       `json:"lastGeoAt"`
	GeoRunning    bool             `json:"geoRunning"`
	PingRunning   bool             `json:"pingRunning"`
	RefreshingAll bool             `json:"refreshingAll"`
	Proxies       []ProxyStatus    `json:"proxies"`
}
