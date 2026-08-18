package subscription

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"xray-geocheck/internal/config"
	"xray-geocheck/internal/model"
)

var skipProtocols = map[string]bool{
	"freedom":   true,
	"blackhole": true,
	"dns":       true,
	"loopback":  true,
	"block":     true,
	"direct":    true,
}

var proxyProtocols = map[string]bool{
	"vless":       true,
	"vmess":       true,
	"trojan":      true,
	"shadowsocks": true,
	"hysteria":    true,
	"wireguard":   true,
	"socks":       true,
	"http":        true,
}

type jsonConfig struct {
	Remarks   string            `json:"remarks"`
	Outbounds []json.RawMessage `json:"outbounds"`
}

func Fetch(ctx context.Context, cfg *config.Config) ([]model.Host, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.SubscriptionURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-hwid", cfg.HWID)
	req.Header.Set("x-device-os", cfg.DeviceOS)
	req.Header.Set("x-ver-os", cfg.DeviceOSVersion)
	req.Header.Set("x-device-model", config.DeviceModel)
	req.Header.Set("user-agent", cfg.UserAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("subscription request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription status %d: %s", resp.StatusCode, truncate(body, 200))
	}
	hosts, err := Parse(body)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("subscription contained no proxy hosts")
	}
	return hosts, nil
}

func Parse(body []byte) ([]model.Host, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty subscription body")
	}
	if hosts, err := tryJSON(trimmed); err == nil && len(hosts) > 0 {
		return hosts, nil
	}
	if decoded, err := decodeBase64Flexible(trimmed); err == nil {
		decoded = bytes.TrimSpace(decoded)
		if hosts, err := tryJSON(decoded); err == nil && len(hosts) > 0 {
			return hosts, nil
		}
		if hosts, err := ParseURIList(decoded); err == nil && len(hosts) > 0 {
			return hosts, nil
		}
	}
	if hosts, err := ParseURIList(trimmed); err == nil && len(hosts) > 0 {
		return hosts, nil
	}
	return nil, fmt.Errorf("unrecognized subscription format")
}

func tryJSON(b []byte) ([]model.Host, error) {
	if !json.Valid(b) {
		return nil, fmt.Errorf("not json")
	}
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil, fmt.Errorf("empty json")
	}
	if b[0] == '[' {
		var items []json.RawMessage
		if err := json.Unmarshal(b, &items); err != nil {
			return nil, err
		}
		var hosts []model.Host
		for i, item := range items {
			h, err := hostFromConfig(item, i)
			if err != nil {
				continue
			}
			hosts = append(hosts, h)
		}
		if len(hosts) == 0 {
			return nil, fmt.Errorf("json array had no proxy outbounds")
		}
		return hosts, nil
	}
	h, err := hostFromConfig(b, 0)
	if err != nil {
		return nil, err
	}
	return []model.Host{h}, nil
}

func hostFromConfig(raw json.RawMessage, idx int) (model.Host, error) {
	var cfg jsonConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return model.Host{}, err
	}
	ob, err := firstProxyOutbound(cfg.Outbounds)
	if err != nil {
		return model.Host{}, err
	}
	proto := stringField(ob, "protocol")
	addr := outboundAddress(ob)
	remarks := strings.TrimSpace(cfg.Remarks)
	if remarks == "" {
		remarks = stringField(ob, "tag")
	}
	if remarks == "" {
		remarks = addr
	}
	if remarks == "" {
		remarks = fmt.Sprintf("proxy-%d", idx+1)
	}
	return model.Host{
		ID:       fmt.Sprintf("%s@%s#%d", remarks, addr, idx),
		Remarks:  remarks,
		Address:  addr,
		Protocol: proto,
		Outbound: ob,
	}, nil
}

func firstProxyOutbound(raws []json.RawMessage) (map[string]any, error) {
	for _, raw := range raws {
		var ob map[string]any
		if err := json.Unmarshal(raw, &ob); err != nil {
			continue
		}
		proto := strings.ToLower(stringField(ob, "protocol"))
		tag := strings.ToLower(stringField(ob, "tag"))
		if skipProtocols[proto] || skipProtocols[tag] {
			continue
		}
		if proxyProtocols[proto] || proto != "" {
			return ob, nil
		}
	}
	return nil, fmt.Errorf("no proxy outbound found")
}

func stringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func outboundAddress(ob map[string]any) string {
	settings, _ := ob["settings"].(map[string]any)
	if settings == nil {
		return ""
	}
	if vnext, ok := settings["vnext"].([]any); ok && len(vnext) > 0 {
		if node, ok := vnext[0].(map[string]any); ok {
			return fmt.Sprintf("%v:%v", node["address"], node["port"])
		}
	}
	if servers, ok := settings["servers"].([]any); ok && len(servers) > 0 {
		if node, ok := servers[0].(map[string]any); ok {
			return fmt.Sprintf("%v:%v", node["address"], node["port"])
		}
	}
	if addr, ok := settings["address"].(string); ok {
		return fmt.Sprintf("%s:%v", addr, settings["port"])
	}
	return ""
}

func decodeBase64Flexible(b []byte) ([]byte, error) {
	s := string(bytes.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, b))
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var last error
	for _, enc := range encodings {
		out, err := enc.DecodeString(s)
		if err == nil {
			return out, nil
		}
		last = err
	}
	return nil, last
}

func truncate(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
