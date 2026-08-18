package subscription

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"xray-geocheck/internal/model"
)

// ParseURIList parses vless/vmess/trojan/ss share links, one per line.
func ParseURIList(body []byte) ([]model.Host, error) {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	var hosts []model.Host
	idx := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		h, err := parseURI(line, idx)
		if err != nil {
			continue
		}
		hosts = append(hosts, h)
		idx++
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no share links parsed")
	}
	return hosts, nil
}

func parseURI(raw string, idx int) (model.Host, error) {
	scheme, rest, ok := strings.Cut(raw, "://")
	if !ok {
		return model.Host{}, fmt.Errorf("missing scheme")
	}
	scheme = strings.ToLower(scheme)
	var (
		ob      map[string]any
		remarks string
		err     error
	)
	switch scheme {
	case "vless":
		ob, remarks, err = parseVless("vless://" + rest)
	case "trojan":
		ob, remarks, err = parseTrojan("trojan://" + rest)
	case "ss":
		ob, remarks, err = parseShadowsocks("ss://" + rest)
	case "vmess":
		ob, remarks, err = parseVmess(rest)
	default:
		return model.Host{}, fmt.Errorf("unsupported scheme %s", scheme)
	}
	if err != nil {
		return model.Host{}, err
	}
	addr := outboundAddress(ob)
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
		Protocol: stringField(ob, "protocol"),
		Outbound: ob,
	}, nil
}

func parseVless(raw string) (map[string]any, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, "", err
	}
	uuid := u.User.Username()
	host := u.Hostname()
	port := atoiDefault(u.Port(), 443)
	q := u.Query()
	remarks, _ := url.PathUnescape(u.Fragment)
	user := map[string]any{"id": uuid, "encryption": first(q.Get("encryption"), "none")}
	if flow := q.Get("flow"); flow != "" {
		user["flow"] = flow
	}
	ob := map[string]any{
		"protocol": "vless",
		"settings": map[string]any{
			"vnext": []any{
				map[string]any{
					"address": host,
					"port":    port,
					"users":   []any{user},
				},
			},
		},
		"streamSettings": streamSettings(q),
	}
	return ob, remarks, nil
}

func parseTrojan(raw string) (map[string]any, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, "", err
	}
	password, _ := u.User.Password()
	if password == "" {
		password = u.User.Username()
	}
	host := u.Hostname()
	port := atoiDefault(u.Port(), 443)
	q := u.Query()
	remarks, _ := url.PathUnescape(u.Fragment)
	ob := map[string]any{
		"protocol": "trojan",
		"settings": map[string]any{
			"servers": []any{
				map[string]any{
					"address":  host,
					"port":     port,
					"password": password,
				},
			},
		},
		"streamSettings": streamSettings(q),
	}
	return ob, remarks, nil
}

func parseShadowsocks(raw string) (map[string]any, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, "", err
	}
	q := u.Query()
	remarks, _ := url.PathUnescape(u.Fragment)
	method, password := "", ""
	if u.User != nil {
		method = u.User.Username()
		password, _ = u.User.Password()
	}
	if password == "" && method != "" && !strings.Contains(method, ":") {
		if decoded, err := decodeBase64Flexible([]byte(method)); err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				method, password = parts[0], parts[1]
			}
		}
	}
	host := u.Hostname()
	port := atoiDefault(u.Port(), 443)
	ob := map[string]any{
		"protocol": "shadowsocks",
		"settings": map[string]any{
			"servers": []any{
				map[string]any{
					"address":  host,
					"port":     port,
					"method":   method,
					"password": password,
				},
			},
		},
		"streamSettings": streamSettings(q),
	}
	return ob, remarks, nil
}

func parseVmess(rest string) (map[string]any, string, error) {
	payload := rest
	if i := strings.Index(payload, "#"); i >= 0 {
		payload = payload[:i]
	}
	decoded, err := decodeBase64Flexible([]byte(payload))
	if err != nil {
		return nil, "", err
	}
	var m map[string]any
	if err := json.Unmarshal(decoded, &m); err != nil {
		return nil, "", err
	}
	remarks := fmt.Sprint(m["ps"])
	host := fmt.Sprint(m["add"])
	port := atoiDefault(fmt.Sprint(m["port"]), 443)
	id := fmt.Sprint(m["id"])
	net := first(fmt.Sprint(m["net"]), "tcp")
	q := url.Values{}
	q.Set("type", net)
	if v := fmt.Sprint(m["path"]); v != "" && v != "<nil>" {
		q.Set("path", v)
	}
	if v := fmt.Sprint(m["host"]); v != "" && v != "<nil>" {
		q.Set("host", v)
	}
	if v := fmt.Sprint(m["tls"]); v == "tls" {
		q.Set("security", "tls")
	}
	if v := fmt.Sprint(m["sni"]); v != "" && v != "<nil>" {
		q.Set("sni", v)
	}
	if v := fmt.Sprint(m["fp"]); v != "" && v != "<nil>" {
		q.Set("fp", v)
	}
	if v := fmt.Sprint(m["alpn"]); v != "" && v != "<nil>" {
		q.Set("alpn", v)
	}
	scy := first(fmt.Sprint(m["scy"]), "auto")
	ob := map[string]any{
		"protocol": "vmess",
		"settings": map[string]any{
			"vnext": []any{
				map[string]any{
					"address": host,
					"port":    port,
					"users": []any{
						map[string]any{
							"id":       id,
							"alterId":  atoiDefault(fmt.Sprint(m["aid"]), 0),
							"security": scy,
						},
					},
				},
			},
		},
		"streamSettings": streamSettings(q),
	}
	return ob, remarks, nil
}

func streamSettings(q url.Values) map[string]any {
	network := first(q.Get("type"), q.Get("network"), "tcp")
	security := first(q.Get("security"), "none")
	ss := map[string]any{
		"network":  network,
		"security": security,
	}
	switch network {
	case "ws":
		ws := map[string]any{"path": first(q.Get("path"), "/")}
		if host := first(q.Get("host"), q.Get("sni")); host != "" {
			ws["headers"] = map[string]any{"Host": host}
		}
		ss["wsSettings"] = ws
	case "httpupgrade":
		ss["httpupgradeSettings"] = map[string]any{
			"path": first(q.Get("path"), "/"),
			"host": q.Get("host"),
		}
	case "grpc":
		ss["grpcSettings"] = map[string]any{
			"serviceName": q.Get("serviceName"),
			"authority":   q.Get("authority"),
		}
	case "xhttp":
		x := map[string]any{"path": q.Get("path"), "host": q.Get("host")}
		if mode := q.Get("mode"); mode != "" {
			x["mode"] = mode
		}
		ss["xhttpSettings"] = x
	case "tcp":
		if ht := q.Get("headerType"); ht != "" && ht != "none" {
			ss["tcpSettings"] = map[string]any{
				"header": map[string]any{"type": ht},
			}
		}
	}
	switch security {
	case "tls":
		tls := map[string]any{}
		if sni := first(q.Get("sni"), q.Get("host")); sni != "" {
			tls["serverName"] = sni
		}
		if fp := q.Get("fp"); fp != "" {
			tls["fingerprint"] = fp
		}
		if alpn := q.Get("alpn"); alpn != "" {
			tls["alpn"] = strings.Split(alpn, ",")
		}
		ss["tlsSettings"] = tls
	case "reality":
		reality := map[string]any{
			"serverName": first(q.Get("sni"), q.Get("serverName")),
			"publicKey":  first(q.Get("pbk"), q.Get("publicKey")),
			"shortId":    first(q.Get("sid"), q.Get("shortId")),
		}
		if fp := q.Get("fp"); fp != "" {
			reality["fingerprint"] = fp
		}
		if spx := first(q.Get("spx"), q.Get("spiderX")); spx != "" {
			reality["spiderX"] = spx
		}
		ss["realitySettings"] = reality
	}
	return ss
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" && v != "<nil>" {
			return v
		}
	}
	return ""
}

func atoiDefault(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}
