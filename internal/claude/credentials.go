package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

var ErrCredentialsNotFound = errors.New("Claude credentials not found")

type Token struct {
	AccessToken string
	ExpiresAt   time.Time
}

type credentialFile struct {
	OAuth struct {
		AccessToken string `json:"accessToken"`
		ExpiresAt   int64  `json:"expiresAt"`
	} `json:"claudeAiOauth"`
}

// LoadToken reads Claude credentials without changing or refreshing them.
func LoadToken(ctx context.Context) (Token, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Token{}, fmt.Errorf("find home directory: %w", err)
	}
	return loadToken(ctx, runtime.GOOS, home, runSecurity)
}

type securityRunner func(context.Context) ([]byte, error)

func runSecurity(ctx context.Context) ([]byte, error) {
	return exec.CommandContext(ctx, "security", "find-generic-password", "-s", "Claude Code-credentials", "-w").Output()
}

func loadToken(ctx context.Context, goos, home string, security securityRunner) (Token, error) {
	var data []byte
	var err error

	switch goos {
	case "darwin":
		data, err = security(ctx)
	case "linux":
		data, err = os.ReadFile(filepath.Join(home, ".claude", ".credentials.json"))
	default:
		return Token{}, fmt.Errorf("unsupported operating system %q", goos)
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Token{}, ErrCredentialsNotFound
		}
		var exitErr *exec.ExitError
		if goos == "darwin" && errors.As(err, &exitErr) {
			return Token{}, ErrCredentialsNotFound
		}
		return Token{}, fmt.Errorf("read Claude credentials: %w", err)
	}

	var credentials credentialFile
	if err := json.Unmarshal(data, &credentials); err != nil {
		return Token{}, fmt.Errorf("parse Claude credentials: %w", err)
	}
	if credentials.OAuth.AccessToken == "" {
		return Token{}, ErrCredentialsNotFound
	}

	token := Token{AccessToken: credentials.OAuth.AccessToken}
	if credentials.OAuth.ExpiresAt > 0 {
		token.ExpiresAt = time.UnixMilli(credentials.OAuth.ExpiresAt)
	}
	return token, nil
}
