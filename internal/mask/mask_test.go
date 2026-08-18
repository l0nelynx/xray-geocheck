package mask

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIPv4(t *testing.T) {
	got := IP("104.28.193.121")
	want := "104.***.***.*21"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if IP("89.58.10.20") != "89.***.***.*20" {
		t.Fatalf("got %q", IP("89.58.10.20"))
	}
	if IP("89.58.10.0/22") != "89.***.***.*0/22" {
		t.Fatalf("cidr got %q", IP("89.58.10.0/22"))
	}
}

func TestStringEmbedsIPv4(t *testing.T) {
	got := String(`lookup 104.28.193.121 on 1.1.1.1:53`)
	if got != "lookup 104.***.***.*21 on 1.***.***.*1:53" {
		t.Fatalf("got %q", got)
	}
}

func TestIPv6(t *testing.T) {
	got := IP("2a03:4000:5f::1412")
	if got != "2a03:***:*12" {
		t.Fatalf("got %q", got)
	}
}

func TestJSONWalksIdentity(t *testing.T) {
	in := json.RawMessage(`{"identity":{"ipv4":"104.28.193.121"},"reputation":{"range":"89.58.10.0/22"}}`)
	out := string(JSON(in))
	if strings.Contains(out, "104.28.193.121") || strings.Contains(out, "89.58.10.0") {
		t.Fatalf("leaked IP in %s", out)
	}
	if !strings.Contains(out, "104.***.***.*21") || !strings.Contains(out, "89.***.***.*0/22") {
		t.Fatalf("missing mask in %s", out)
	}
}
