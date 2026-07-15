package claude

import (
	"os"
	"strings"
	"testing"

	"github.com/kter/ai-token-checker/internal/quota"
)

func TestParseUsage(t *testing.T) {
	file, err := os.Open("testdata/usage.json")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	windows, err := ParseUsage(file)
	if err != nil {
		t.Fatalf("ParseUsage() error = %v", err)
	}
	if len(windows) != 2 {
		t.Fatalf("len(windows) = %d, want 2", len(windows))
	}
	if windows[0].Label != quota.LabelFiveHour || windows[0].UsedPercent != 14.5 {
		t.Fatalf("5h window = %+v", windows[0])
	}
	if windows[1].Label != quota.LabelWeekly || windows[1].UsedPercent != 23.0 {
		t.Fatalf("weekly window = %+v", windows[1])
	}
	if windows[1].RemainingPercent == nil || *windows[1].RemainingPercent != 77.0 {
		t.Fatalf("weekly remaining = %v", windows[1].RemainingPercent)
	}
}

func TestParseUsageDerivesAndClampsRemainingPercent(t *testing.T) {
	usage := `{
		"five_hour": {
			"utilization": 34.25,
			"resets_at": "2026-07-15T05:00:00Z"
		},
		"seven_day": {
			"utilization": 125,
			"resets_at": "2026-07-20T08:00:00Z"
		}
	}`

	windows, err := ParseUsage(strings.NewReader(usage))
	if err != nil {
		t.Fatalf("ParseUsage() error = %v", err)
	}
	if windows[0].RemainingPercent == nil || *windows[0].RemainingPercent != 65.75 {
		t.Fatalf("5h remaining = %v, want 65.75", windows[0].RemainingPercent)
	}
	if windows[1].RemainingPercent == nil || *windows[1].RemainingPercent != 0 {
		t.Fatalf("weekly remaining = %v, want 0", windows[1].RemainingPercent)
	}
}
