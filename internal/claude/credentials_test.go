package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadTokenLinuxFile(t *testing.T) {
	home := t.TempDir()
	directory := filepath.Join(home, ".claude")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"claudeAiOauth":{"accessToken":"read-only-token","expiresAt":1780000000000}}`)
	if err := os.WriteFile(filepath.Join(directory, ".credentials.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	token, err := loadToken(context.Background(), "linux", home, nil)
	if err != nil {
		t.Fatalf("loadToken() error = %v", err)
	}
	if token.AccessToken != "read-only-token" {
		t.Fatalf("AccessToken = %q", token.AccessToken)
	}
	wantExpiry := time.UnixMilli(1780000000000)
	if !token.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("ExpiresAt = %v, want %v", token.ExpiresAt, wantExpiry)
	}
}
