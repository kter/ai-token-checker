package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHelpExitsSuccessfully(t *testing.T) {
	for _, argument := range []string{"-h", "--help"} {
		t.Run(argument, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(context.Background(), []string{argument}, &stdout, &stderr); code != 0 {
				t.Fatalf("run() = %d, want 0; stderr = %q", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), "Usage: ai-token-checker") {
				t.Fatalf("help output = %q", stderr.String())
			}
		})
	}
}

func TestInvalidToolIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"--tool", "other"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run() = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--tool must be claude or codex") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() = %d, want 0", code)
	}
	if stdout.String() != "ai-token-checker dev\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
