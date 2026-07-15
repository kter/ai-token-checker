package codex

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kter/ai-token-checker/internal/quota"
)

func TestParseRolloutRealWeeklyLine(t *testing.T) {
	line := `{"timestamp":"2026-07-14T06:26:35.956Z","type":"event_msg","payload":{"type":"token_count","info":{},"rate_limits":{"limit_id":"codex","primary":{"used_percent":23.0,"window_minutes":10080,"resets_at":1784537760},"secondary":null,"credits":null,"plan_type":"team"}}}`
	snapshot, err := ParseRollout(strings.NewReader(line), time.Time{})
	if err != nil {
		t.Fatalf("ParseRollout() error = %v", err)
	}
	if len(snapshot.Windows) != 1 {
		t.Fatalf("len(windows) = %d, want 1", len(snapshot.Windows))
	}
	window := snapshot.Windows[0]
	if window.Label != quota.LabelWeekly || window.UsedPercent != 23.0 {
		t.Fatalf("weekly window = %+v", window)
	}
	if window.WindowMinutes == nil || *window.WindowMinutes != 10080 {
		t.Fatalf("window_minutes = %v", window.WindowMinutes)
	}
}

func TestParseRolloutUsesLastRateLimitAndClassifiesFiveHour(t *testing.T) {
	file, err := os.Open("testdata/rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	snapshot, err := ParseRollout(file, time.Time{})
	if err != nil {
		t.Fatalf("ParseRollout() error = %v", err)
	}
	if len(snapshot.Windows) != 2 {
		t.Fatalf("len(windows) = %d, want 2", len(snapshot.Windows))
	}
	if snapshot.Windows[0].Label != quota.LabelFiveHour || snapshot.Windows[0].UsedPercent != 7.5 {
		t.Fatalf("5h window = %+v", snapshot.Windows[0])
	}
	if snapshot.Windows[1].Label != quota.LabelWeekly || snapshot.Windows[1].UsedPercent != 24.0 {
		t.Fatalf("weekly window = %+v", snapshot.Windows[1])
	}
}

func TestClassifyWindow(t *testing.T) {
	tests := []struct {
		minutes int64
		label   string
		ok      bool
	}{
		{300, quota.LabelFiveHour, true},
		{10080, quota.LabelWeekly, true},
		{60, "", false},
	}
	for _, test := range tests {
		label, ok := ClassifyWindow(test.minutes)
		if label != test.label || ok != test.ok {
			t.Errorf("ClassifyWindow(%d) = %q, %v; want %q, %v", test.minutes, label, ok, test.label, test.ok)
		}
	}
}

func TestParseRolloutFallsBackToFileMtime(t *testing.T) {
	fallback := time.Date(2026, 7, 15, 1, 2, 3, 0, time.UTC)
	line := `{"type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":1,"window_minutes":300,"resets_at":1784537760}}}}`
	snapshot, err := ParseRollout(strings.NewReader(line), fallback)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Freshness.Equal(fallback) {
		t.Fatalf("Freshness = %v, want %v", snapshot.Freshness, fallback)
	}
}
