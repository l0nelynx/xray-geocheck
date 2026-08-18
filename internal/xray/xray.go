package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"xray-geocheck/internal/model"
)

const socksHost = "127.0.0.1"

// Instance wraps a single xray-core process with one SOCKS inbound per host.
type Instance struct {
	mu      sync.Mutex
	bin     string
	workDir string
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	hosts   []model.Host
	port0   int
}

func New(bin, workDir string, portBase int) *Instance {
	return &Instance{bin: bin, workDir: workDir, port0: portBase}
}

func (i *Instance) Hosts() []model.Host {
	i.mu.Lock()
	defer i.mu.Unlock()
	out := make([]model.Host, len(i.hosts))
	copy(out, i.hosts)
	return out
}

func (i *Instance) Apply(ctx context.Context, hosts []model.Host) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	assigned := assignPorts(hosts, i.port0)
	if sameHosts(i.hosts, assigned) && i.cmd != nil && i.cmd.Process != nil {
		return nil
	}
	if err := i.stopLocked(); err != nil {
		slog.Warn("xray stop", "err", err)
	}
	if err := os.MkdirAll(i.workDir, 0o755); err != nil {
		return err
	}
	cfgPath := filepath.Join(i.workDir, "config.json")
	raw, err := buildConfig(assigned)
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(runCtx, i.bin, "run", "-c", cfgPath)
	cmd.Dir = filepath.Dir(i.bin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start xray: %w", err)
	}
	i.cmd = cmd
	i.cancel = cancel
	i.hosts = assigned
	go func() {
		_ = cmd.Wait()
	}()
	if err := waitSOCKS(ctx, assigned, 15*time.Second); err != nil {
		_ = i.stopLocked()
		return err
	}
	slog.Info("xray ready", "hosts", len(assigned))
	return nil
}

func (i *Instance) Stop() {
	i.mu.Lock()
	defer i.mu.Unlock()
	_ = i.stopLocked()
}

func (i *Instance) stopLocked() error {
	if i.cancel != nil {
		i.cancel()
		i.cancel = nil
	}
	if i.cmd != nil && i.cmd.Process != nil {
		_ = i.cmd.Process.Kill()
		i.cmd = nil
	}
	return nil
}

func assignPorts(hosts []model.Host, base int) []model.Host {
	out := make([]model.Host, len(hosts))
	for i, h := range hosts {
		h.SocksHost = socksHost
		h.SocksPort = base + i
		out[i] = h
	}
	return out
}

func sameHosts(a, b []model.Host) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].SocksPort != b[i].SocksPort {
			return false
		}
	}
	return true
}

func buildConfig(hosts []model.Host) ([]byte, error) {
	inbounds := make([]any, 0, len(hosts))
	outbounds := make([]any, 0, len(hosts)+1)
	rules := make([]any, 0, len(hosts))
	for i, h := range hosts {
		inTag := fmt.Sprintf("socks-%d", i)
		outTag := fmt.Sprintf("proxy-%d", i)
		inbounds = append(inbounds, map[string]any{
			"tag":      inTag,
			"listen":   h.SocksHost,
			"port":     h.SocksPort,
			"protocol": "socks",
			"settings": map[string]any{"auth": "noauth", "udp": true},
		})
		ob := cloneMap(h.Outbound)
		ob["tag"] = outTag
		outbounds = append(outbounds, ob)
		rules = append(rules, map[string]any{
			"type":        "field",
			"inboundTag":  []string{inTag},
			"outboundTag": outTag,
		})
	}
	outbounds = append(outbounds, map[string]any{
		"tag":      "block",
		"protocol": "blackhole",
	})
	cfg := map[string]any{
		"log":       map[string]any{"loglevel": "warning"},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"routing": map[string]any{
			"domainStrategy": "AsIs",
			"rules":          rules,
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func cloneMap(in map[string]any) map[string]any {
	b, _ := json.Marshal(in)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	if out == nil {
		out = map[string]any{}
	}
	return out
}

func waitSOCKS(ctx context.Context, hosts []model.Host, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for _, h := range hosts {
		addr := h.SocksAddr()
		for {
			if time.Now().After(deadline) {
				return fmt.Errorf("socks %s did not become ready", addr)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			c, err := net.DialTimeout("tcp", addr, 400*time.Millisecond)
			if err == nil {
				_ = c.Close()
				break
			}
			time.Sleep(150 * time.Millisecond)
		}
	}
	return nil
}
