package mask

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"
)

var ipv4Re = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)(?:/(?:3[0-2]|[12]?\d))?\b`)

// IPv4: 104.28.193.121 → 104.***.***.*21
// IPv6: 2a03:4000:5f::1412 → 2a03:***:*12
func IP(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	prefix, suffix := s, ""
	if i := strings.LastIndex(s, "/"); i > 0 && net.ParseIP(s[:i]) != nil {
		prefix, suffix = s[:i], s[i:]
	}
	ip := net.ParseIP(prefix)
	if ip == nil {
		return s
	}
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.***.***.*%d%s", v4[0], int(v4[3])%100, suffix)
	}
	raw := ip.String()
	hex := strings.ReplaceAll(raw, ":", "")
	if len(hex) < 2 {
		return s
	}
	first := strings.Split(raw, ":")[0]
	return first + ":***:*" + hex[len(hex)-2:] + suffix
}

func String(s string) string {
	if s == "" {
		return s
	}
	if net.ParseIP(s) != nil || cidrIP(s) != "" {
		return IP(s)
	}
	return ipv4Re.ReplaceAllStringFunc(s, IP)
}

func cidrIP(s string) string {
	i := strings.LastIndex(s, "/")
	if i <= 0 {
		return ""
	}
	if net.ParseIP(s[:i]) != nil {
		return s[:i]
	}
	return ""
}

func JSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return json.RawMessage([]byte(strconvQuote(String(string(raw)))))
	}
	out, err := json.Marshal(walk(v))
	if err != nil {
		return raw
	}
	return out
}

func walk(v any) any {
	switch t := v.(type) {
	case string:
		return String(t)
	case map[string]any:
		for k, val := range t {
			t[k] = walk(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = walk(val)
		}
		return t
	default:
		return v
	}
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
