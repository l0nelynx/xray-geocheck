package httpapi

import (
	"strings"
	"testing"
)

func TestInjectBase(t *testing.T) {
	in := []byte(`<!doctype html><html><head><meta charset="utf-8"></head></html>`)
	out := string(injectBase(in, "/geocheck/"))
	if !strings.Contains(out, `<base href="/geocheck/">`) {
		t.Fatalf("missing base tag: %s", out)
	}
	if !strings.Contains(out, `<head>
<base href="/geocheck/">`) {
		t.Fatalf("base should sit in head: %s", out)
	}
}

func TestNormalizeKeepsRootAssetsRelative(t *testing.T) {
	in := []byte(`<head></head>`)
	out := string(injectBase(in, "/"))
	if !strings.Contains(out, `<base href="/">`) {
		t.Fatalf("got %s", out)
	}
}
