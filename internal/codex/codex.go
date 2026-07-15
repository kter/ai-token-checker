package codex

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kter/ai-token-checker/internal/quota"
)

var ErrNoCachedQuota = errors.New("no cached Codex quota")

type Snapshot struct {
	Windows   []quota.Window
	Freshness time.Time
	File      string
}

func LoadNewest(home string) (Snapshot, error) {
	path, modified, err := newestRollout(filepath.Join(home, ".codex", "sessions"))
	if err != nil {
		return Snapshot{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("open Codex rollout: %w", err)
	}
	defer file.Close()

	snapshot, err := ParseRollout(file, modified)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.File = path
	return snapshot, nil
}

func newestRollout(root string) (string, time.Time, error) {
	var newestPath string
	var newestTime time.Time
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, os.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "rollout-") || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if newestPath == "" || info.ModTime().After(newestTime) {
			newestPath = path
			newestTime = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("find Codex rollout: %w", err)
	}
	if newestPath == "" {
		return "", time.Time{}, ErrNoCachedQuota
	}
	return newestPath, newestTime, nil
}

type rolloutLine struct {
	Timestamp string  `json:"timestamp"`
	Type      string  `json:"type"`
	Payload   payload `json:"payload"`
}

type payload struct {
	Type       string      `json:"type"`
	RateLimits *rateLimits `json:"rate_limits"`
}

type rateLimits struct {
	Primary   *rateWindow `json:"primary"`
	Secondary *rateWindow `json:"secondary"`
}

type rateWindow struct {
	UsedPercent   float64 `json:"used_percent"`
	WindowMinutes int64   `json:"window_minutes"`
	ResetsAt      int64   `json:"resets_at"`
}

func ParseRollout(reader io.Reader, fallback time.Time) (Snapshot, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	var latest *rolloutLine
	for scanner.Scan() {
		var line rolloutLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type == "event_msg" && line.Payload.Type == "token_count" && line.Payload.RateLimits != nil {
			copy := line
			latest = &copy
		}
	}
	if err := scanner.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("scan Codex rollout: %w", err)
	}
	if latest == nil {
		return Snapshot{}, ErrNoCachedQuota
	}

	byLabel := make(map[string]quota.Window)
	for _, raw := range []*rateWindow{latest.Payload.RateLimits.Primary, latest.Payload.RateLimits.Secondary} {
		if raw == nil {
			continue
		}
		label, ok := ClassifyWindow(raw.WindowMinutes)
		if !ok {
			continue
		}
		minutes := raw.WindowMinutes
		byLabel[label] = quota.Window{
			Label:         label,
			UsedPercent:   raw.UsedPercent,
			WindowMinutes: &minutes,
			ResetsAt:      time.Unix(raw.ResetsAt, 0),
		}
	}
	if len(byLabel) == 0 {
		return Snapshot{}, ErrNoCachedQuota
	}

	windows := make([]quota.Window, 0, len(byLabel))
	for _, label := range []string{quota.LabelFiveHour, quota.LabelWeekly} {
		if window, ok := byLabel[label]; ok {
			windows = append(windows, window)
		}
	}

	freshness := fallback
	if parsed, err := time.Parse(time.RFC3339Nano, latest.Timestamp); err == nil {
		freshness = parsed
	}
	return Snapshot{Windows: windows, Freshness: freshness}, nil
}

// ClassifyWindow uses duration rather than primary/secondary position because
// Codex can put either limit in either slot.
func ClassifyWindow(minutes int64) (string, bool) {
	switch {
	case minutes >= 240 && minutes <= 360:
		return quota.LabelFiveHour, true
	case minutes >= 8640 && minutes <= 11520:
		return quota.LabelWeekly, true
	default:
		return "", false
	}
}
