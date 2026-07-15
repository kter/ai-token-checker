package render

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/kter/ai-token-checker/internal/quota"
)

const barWidth = 20

func StdoutIsTerminal() bool {
	return isTerminal(os.Stdout.Fd())
}

func ColorEnabled(noColor, terminal, noColorEnvironmentSet bool) bool {
	return !noColor && terminal && !noColorEnvironmentSet
}

func HumanDuration(duration time.Duration) string {
	if duration <= 0 {
		return "now"
	}
	duration = duration.Round(time.Minute)
	if duration < time.Minute {
		return "<1m"
	}

	minutes := int64(duration / time.Minute)
	days := minutes / (24 * 60)
	minutes %= 24 * 60
	hours := minutes / 60
	minutes %= 60

	parts := make([]string, 0, 3)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 && days == 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	return strings.Join(parts, " ")
}

func Summary(writer io.Writer, reports []quota.ToolReport, now time.Time, color bool) error {
	for index, report := range reports {
		if index > 0 {
			if _, err := fmt.Fprintln(writer); err != nil {
				return err
			}
		}
		if report.Status != "ok" {
			if _, err := fmt.Fprintln(writer, report.Message); err != nil {
				return err
			}
			continue
		}

		heading := title(report.Tool)
		if report.Tool == "codex" {
			heading += " (as of " + report.Freshness.Local().Format(time.RFC3339) + ")"
		}
		if color {
			heading = "\x1b[1;36m" + heading + "\x1b[0m"
		}
		if _, err := fmt.Fprintln(writer, heading); err != nil {
			return err
		}

		byLabel := make(map[string]quota.Window, len(report.Windows))
		for _, window := range report.Windows {
			byLabel[window.Label] = window
		}
		for _, label := range []string{quota.LabelFiveHour, quota.LabelWeekly} {
			window, ok := byLabel[label]
			if !ok {
				if _, err := fmt.Fprintf(writer, "  %-7s %5s  [%-20s]  %s\n", label, "--", "", "no cached window"); err != nil {
					return err
				}
				continue
			}
			bar := progressBar(window.UsedPercent)
			if color {
				bar = barColor(window.UsedPercent) + bar + "\x1b[0m"
			}
			if _, err := fmt.Fprintf(writer, "  %-7s %5.1f%%  [%s]  resets in %s\n",
				label, window.UsedPercent, bar, HumanDuration(window.ResetsAt.Sub(now))); err != nil {
				return err
			}
		}
	}
	return nil
}

func progressBar(percent float64) string {
	percent = math.Max(0, math.Min(100, percent))
	filled := int(math.Round(percent / 100 * barWidth))
	return strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
}

func barColor(percent float64) string {
	switch {
	case percent >= 90:
		return "\x1b[31m"
	case percent >= 70:
		return "\x1b[33m"
	default:
		return "\x1b[32m"
	}
}

func title(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

type jsonOutput struct {
	GeneratedAt time.Time    `json:"generated_at"`
	Tools       []jsonReport `json:"tools"`
}

type jsonReport struct {
	Tool      string       `json:"tool"`
	Source    string       `json:"source"`
	Status    string       `json:"status"`
	Message   string       `json:"message,omitempty"`
	Freshness *time.Time   `json:"freshness,omitempty"`
	Windows   []jsonWindow `json:"windows"`
}

type jsonWindow struct {
	Label                string    `json:"label"`
	UsedPercent          float64   `json:"used_percent"`
	RemainingPercent     *float64  `json:"remaining_percent,omitempty"`
	WindowMinutes        *int64    `json:"window_minutes,omitempty"`
	ResetsAt             time.Time `json:"resets_at"`
	ResetsAtEpochSeconds int64     `json:"resets_at_epoch_seconds"`
	ResetsInSeconds      int64     `json:"resets_in_seconds"`
	ResetsIn             string    `json:"resets_in"`
}

func JSON(writer io.Writer, reports []quota.ToolReport, now time.Time) error {
	output := jsonOutput{GeneratedAt: now, Tools: make([]jsonReport, 0, len(reports))}
	for _, report := range reports {
		item := jsonReport{
			Tool:    report.Tool,
			Source:  report.Source,
			Status:  report.Status,
			Message: report.Message,
			Windows: make([]jsonWindow, 0, len(report.Windows)),
		}
		if !report.Freshness.IsZero() {
			freshness := report.Freshness
			item.Freshness = &freshness
		}
		for _, window := range report.Windows {
			seconds := int64(math.Max(0, window.ResetsAt.Sub(now).Seconds()))
			item.Windows = append(item.Windows, jsonWindow{
				Label:                window.Label,
				UsedPercent:          window.UsedPercent,
				RemainingPercent:     window.RemainingPercent,
				WindowMinutes:        window.WindowMinutes,
				ResetsAt:             window.ResetsAt,
				ResetsAtEpochSeconds: window.ResetsAt.Unix(),
				ResetsInSeconds:      seconds,
				ResetsIn:             HumanDuration(window.ResetsAt.Sub(now)),
			})
		}
		output.Tools = append(output.Tools, item)
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
