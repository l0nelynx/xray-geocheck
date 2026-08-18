package hwid

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var valid = regexp.MustCompile(`^[a-zA-Z0-9=-]{10,64}$`)

// LoadOrCreate returns a stable 16-character hardware id stored under dataDir.
func LoadOrCreate(dataDir string) (string, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dataDir, "hwid")
	if b, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(b))
		if valid.MatchString(id) {
			return id, nil
		}
	}
	id, err := generate(16)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write hwid: %w", err)
	}
	return id, nil
}

func generate(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := range buf {
		out[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(out), nil
}
