package binaries

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Manifest struct {
	Xray     Tool `json:"xray"`
	Geocheck Tool `json:"geocheck"`
}

type Tool struct {
	Version string            `json:"version"`
	Assets  map[string]string `json:"assets"`
}

type Paths struct {
	Xray     string
	Geocheck string
}

func Ensure(ctx context.Context, depsFile, binDir, xrayURL, geocheckURL string) (Paths, error) {
	raw, err := os.ReadFile(depsFile)
	if err != nil {
		return Paths{}, fmt.Errorf("read deps file %s: %w", depsFile, err)
	}
	var man Manifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return Paths{}, fmt.Errorf("parse deps file: %w", err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return Paths{}, err
	}

	xraySrc := xrayURL
	if xraySrc == "" {
		xraySrc, err = assetURL(man.Xray.Assets, xrayPlatformKey())
		if err != nil {
			return Paths{}, fmt.Errorf("xray: %w", err)
		}
	}
	geoSrc := geocheckURL
	if geoSrc == "" {
		geoSrc, err = assetURL(man.Geocheck.Assets, geocheckPlatformKey())
		if err != nil {
			return Paths{}, fmt.Errorf("geocheck: %w", err)
		}
	}

	slog.Info("downloading xray-core", "version", man.Xray.Version, "url", xraySrc)
	if err := downloadAndExtract(ctx, xraySrc, binDir); err != nil {
		return Paths{}, fmt.Errorf("xray download: %w", err)
	}
	slog.Info("downloading geocheck", "version", man.Geocheck.Version, "url", geoSrc)
	if err := downloadAndExtract(ctx, geoSrc, binDir); err != nil {
		return Paths{}, fmt.Errorf("geocheck download: %w", err)
	}

	xrayPath, err := findBinary(binDir, "xray")
	if err != nil {
		return Paths{}, err
	}
	geoPath, err := findBinary(binDir, "geocheck")
	if err != nil {
		return Paths{}, err
	}
	if err := os.Chmod(xrayPath, 0o755); err != nil {
		return Paths{}, err
	}
	if err := os.Chmod(geoPath, 0o755); err != nil {
		return Paths{}, err
	}
	return Paths{Xray: xrayPath, Geocheck: geoPath}, nil
}

func xrayPlatformKey() string {
	return runtime.GOOS + "_" + normalizeArch(runtime.GOARCH)
}

func geocheckPlatformKey() string {
	if runtime.GOOS == "darwin" {
		return "darwin"
	}
	return runtime.GOOS + "_" + normalizeArch(runtime.GOARCH)
}

func normalizeArch(arch string) string {
	switch arch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return arch
	}
}

func assetURL(assets map[string]string, key string) (string, error) {
	u, ok := assets[key]
	if !ok || u == "" {
		return "", fmt.Errorf("no asset for platform %q (supported: %v)", key, keys(assets))
	}
	return u, nil
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func downloadAndExtract(ctx context.Context, url, destDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "xray-geocheck")
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(destDir, "download-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(tmpName, destDir)
	default:
		return extractZip(tmpName, destDir)
	}
}

func extractZip(path, destDir string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if err := writeZipFile(f, destDir); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(f *zip.File, destDir string) error {
	name := filepath.Base(f.Name)
	if name == "" || name == "." || strings.HasPrefix(name, ".") {
		return nil
	}
	if f.FileInfo().IsDir() {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	dstPath := filepath.Join(destDir, name)
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, rc)
	return err
}

func extractTarGz(path, destDir string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Base(hdr.Name)
		if name == "" || name == "." {
			continue
		}
		dstPath := filepath.Join(destDir, name)
		dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, tr); err != nil {
			dst.Close()
			return err
		}
		dst.Close()
	}
}

func findBinary(dir, name string) (string, error) {
	candidates := []string{name, name + ".exe"}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		for _, c := range candidates {
			if strings.EqualFold(e.Name(), c) {
				return filepath.Join(dir, e.Name()), nil
			}
		}
	}
	return "", fmt.Errorf("binary %q not found in %s", name, dir)
}
