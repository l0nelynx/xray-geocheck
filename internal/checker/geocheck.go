package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"xray-geocheck/internal/model"
)

const geoTimeout = 5 * time.Minute

// RunGeocheck executes geocheck through a local SOCKS5 proxy and returns the JSON report.
func RunGeocheck(ctx context.Context, bin, socksAddr string) model.GeoResult {
	runCtx, cancel := context.WithTimeout(ctx, geoTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, bin, "-p", socksAddr, "--json", "-4", "-q")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := model.GeoResult{CheckedAt: time.Now().UTC()}
	raw := extractJSON(stdout.Bytes())
	if len(raw) == 0 {
		res.OK = false
		if err != nil {
			res.Error = fmt.Sprintf("%v: %s", err, truncateBytes(stderr.Bytes(), 400))
		} else {
			res.Error = "geocheck produced no JSON"
		}
		return res
	}
	if !json.Valid(raw) {
		res.OK = false
		res.Error = "geocheck stdout was not valid JSON"
		return res
	}
	res.OK = true
	res.Report = json.RawMessage(raw)
	if err != nil {
		res.Error = err.Error()
	}
	return res
}

func extractJSON(b []byte) []byte {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil
	}
	if b[0] == '{' {
		return b
	}
	idx := bytes.LastIndexByte(b, '{')
	if idx < 0 {
		return nil
	}
	return bytes.TrimSpace(b[idx:])
}

func truncateBytes(b []byte, n int) string {
	s := string(bytes.TrimSpace(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
