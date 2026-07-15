package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kter/ai-token-checker/internal/quota"
)

const usageURL = "https://api.anthropic.com/api/oauth/usage"

var ErrTokenExpired = errors.New("Claude token expired")

type Client struct {
	HTTPClient *http.Client
	URL        string
}

func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		URL:        usageURL,
	}
}

func (c *Client) Usage(ctx context.Context, accessToken string) ([]quota.Window, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("create Claude usage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("request Claude usage: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return nil, time.Time{}, ErrTokenExpired
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, time.Time{}, fmt.Errorf("Claude usage API returned %s", response.Status)
	}

	windows, err := ParseUsage(response.Body)
	if err != nil {
		return nil, time.Time{}, err
	}
	return windows, time.Now(), nil
}

type usageResponse struct {
	FiveHour *usageWindow `json:"five_hour"`
	SevenDay *usageWindow `json:"seven_day"`
}

type usageWindow struct {
	Utilization float64         `json:"utilization"`
	ResetsAt    json.RawMessage `json:"resets_at"`
}

func ParseUsage(reader io.Reader) ([]quota.Window, error) {
	var response usageResponse
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("parse Claude usage response: %w", err)
	}

	var windows []quota.Window
	for _, item := range []struct {
		label string
		raw   *usageWindow
	}{
		{label: quota.LabelFiveHour, raw: response.FiveHour},
		{label: quota.LabelWeekly, raw: response.SevenDay},
	} {
		if item.raw == nil {
			continue
		}
		resetsAt, err := parseResetTime(item.raw.ResetsAt)
		if err != nil {
			return nil, fmt.Errorf("parse Claude %s reset time: %w", item.label, err)
		}
		remaining := 100 - item.raw.Utilization
		if remaining < 0 {
			remaining = 0
		}
		windows = append(windows, quota.Window{
			Label:            item.label,
			UsedPercent:      item.raw.Utilization,
			RemainingPercent: &remaining,
			ResetsAt:         resetsAt,
		})
	}
	if len(windows) == 0 {
		return nil, errors.New("Claude usage response contains no 5-hour or weekly quota")
	}
	return windows, nil
}

func parseResetTime(raw json.RawMessage) (time.Time, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return time.Time{}, errors.New("missing resets_at")
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return parsed, nil
		}
		if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
			return time.Unix(seconds, 0), nil
		}
		return time.Time{}, fmt.Errorf("unsupported value %q", value)
	}

	var seconds json.Number
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&seconds); err != nil {
		return time.Time{}, fmt.Errorf("unsupported value %s", raw)
	}
	epoch, err := seconds.Int64()
	if err != nil {
		return time.Time{}, fmt.Errorf("unsupported value %s", raw)
	}
	return time.Unix(epoch, 0), nil
}
