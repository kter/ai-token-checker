package render

import (
	"os"
	"testing"
	"time"
)

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{-time.Minute, "now"},
		{20 * time.Second, "<1m"},
		{90 * time.Minute, "1h 30m"},
		{49*time.Hour + 20*time.Minute, "2d 1h"},
	}
	for _, test := range tests {
		if got := HumanDuration(test.duration); got != test.want {
			t.Errorf("HumanDuration(%v) = %q, want %q", test.duration, got, test.want)
		}
	}
}

func TestColorEnabled(t *testing.T) {
	tests := []struct {
		name       string
		noColor    bool
		terminal   bool
		noColorEnv bool
		want       bool
	}{
		{"tty", false, true, false, true},
		{"not tty", false, false, false, false},
		{"flag", true, true, false, false},
		{"environment, including empty value", false, true, true, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ColorEnabled(test.noColor, test.terminal, test.noColorEnv); got != test.want {
				t.Fatalf("ColorEnabled() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIsTerminalRejectsDevNull(t *testing.T) {
	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if isTerminal(file.Fd()) {
		t.Fatal("isTerminal(os.DevNull) = true, want false")
	}
}
