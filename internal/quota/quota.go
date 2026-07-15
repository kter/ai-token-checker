package quota

import "time"

const (
	LabelFiveHour = "5h"
	LabelWeekly   = "Weekly"
)

// Window is a quota window normalized across providers. RemainingPercent is
// populated when the provider returns it, while WindowMinutes is populated for
// Codex's cached rate-limit records.
type Window struct {
	Label            string
	UsedPercent      float64
	RemainingPercent *float64
	WindowMinutes    *int64
	ResetsAt         time.Time
}

// ToolReport contains either normalized quota windows or a short failure note.
type ToolReport struct {
	Tool      string
	Source    string
	Status    string
	Message   string
	Windows   []Window
	Freshness time.Time
}

func Success(tool, source string, windows []Window, freshness time.Time) ToolReport {
	return ToolReport{
		Tool:      tool,
		Source:    source,
		Status:    "ok",
		Windows:   windows,
		Freshness: freshness,
	}
}

func Failure(tool, source, message string) ToolReport {
	return ToolReport{
		Tool:    tool,
		Source:  source,
		Status:  "unavailable",
		Message: message,
	}
}
