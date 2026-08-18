package checker

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/proxy"

	"xray-geocheck/internal/model"
)

const pingTimeout = 20 * time.Second

// PingGET measures wall-clock RTT of one HTTP GET through a SOCKS5 proxy.
func PingGET(ctx context.Context, socksAddr, targetURL string) model.PingResult {
	start := time.Now()
	status, err := doGET(ctx, socksAddr, targetURL)
	rtt := time.Since(start)
	res := model.PingResult{
		RTTMs:     float64(rtt.Microseconds()) / 1000.0,
		Status:    status,
		CheckedAt: time.Now().UTC(),
	}
	if err != nil {
		res.Error = err.Error()
		res.Up = false
		return res
	}
	res.Up = isSuccess(targetURL, status)
	if !res.Up {
		res.Error = fmt.Sprintf("unexpected status %d", status)
	}
	return res
}

func isSuccess(targetURL string, status int) bool {
	if strings.Contains(targetURL, "generate_204") {
		return status == http.StatusNoContent
	}
	return status >= 200 && status < 400
}

func doGET(ctx context.Context, socksAddr, targetURL string) (int, error) {
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		return 0, err
	}
	transport := &http.Transport{
		DialContext:         asContextDialer(dialer).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   true,
		ForceAttemptHTTP2:   false,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   pingTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, nil
}

type contextDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

func asContextDialer(d proxy.Dialer) contextDialer {
	if cd, ok := d.(proxy.ContextDialer); ok {
		return cd
	}
	return ctxDialer{d: d}
}

type ctxDialer struct {
	d proxy.Dialer
}

func (c ctxDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	type result struct {
		c   net.Conn
		err error
	}
	ch := make(chan result, 1)
	go func() {
		conn, err := c.d.Dial(network, address)
		ch <- result{conn, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.c, r.err
	}
}
