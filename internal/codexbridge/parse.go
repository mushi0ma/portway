package codexbridge

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// usageStatus derives ok|warning|exhausted from windows; reset_at is when an exhausted
// account can be used again (latest reset among exhausted windows, null if unknown).
func usageStatus(windows []Window, blocked bool, now time.Time) *Usage {
	u := &Usage{Status: "ok", CheckedAt: now.UTC(), Windows: windows}
	exhausted := blocked
	for _, w := range windows {
		if w.UsedPercent >= 100 {
			exhausted = true
			if w.ResetsAt != nil && (u.ResetAt == nil || w.ResetsAt.After(*u.ResetAt)) {
				u.ResetAt = w.ResetsAt
			}
		} else if w.UsedPercent >= 80 && u.Status == "ok" {
			u.Status = "warning"
		}
	}
	if exhausted {
		u.Status = "exhausted"
	} else {
		u.ResetAt = nil
	}
	return u
}
func windowID(minutes int64, fallback string) string {
	switch minutes {
	case 300:
		return "five_hour"
	case 10080:
		return "weekly"
	}
	return fallback
}

// CodexUsage maps `account/rateLimits/read` into the contract usage shape.
func CodexUsage(raw []byte, now time.Time) *Usage {
	type window struct {
		UsedPercent float64 `json:"usedPercent"`
		Minutes     *int64  `json:"windowDurationMins"`
		ResetsAt    *int64  `json:"resetsAt"`
	}
	var in struct {
		Allowed    *bool `json:"ordinaryUsageAllowed"`
		RateLimits struct {
			Primary   *window `json:"primary"`
			Secondary *window `json:"secondary"`
			Reached   *string `json:"rateLimitReachedType"`
		} `json:"rateLimits"`
	}
	if json.Unmarshal(raw, &in) != nil {
		return &Usage{Status: "unknown", CheckedAt: now.UTC(), Windows: []Window{}}
	}
	windows := []Window{}
	for i, w := range []*window{in.RateLimits.Primary, in.RateLimits.Secondary} {
		if w == nil {
			continue
		}
		out := Window{ID: []string{"primary", "secondary"}[i], UsedPercent: clampPercent(w.UsedPercent), WindowMinutes: w.Minutes}
		if w.Minutes != nil {
			out.ID = windowID(*w.Minutes, out.ID)
		}
		if w.ResetsAt != nil && *w.ResetsAt > 0 {
			t := time.Unix(*w.ResetsAt, 0).UTC()
			out.ResetsAt = &t
		}
		windows = append(windows, out)
	}
	return usageStatus(windows, (in.Allowed != nil && !*in.Allowed) || in.RateLimits.Reached != nil, now)
}

// ClaudeUsage maps the Claude OAuth usage endpoint into the contract usage shape.
func ClaudeUsage(raw []byte, now time.Time) *Usage {
	var in map[string]json.RawMessage
	if json.Unmarshal(raw, &in) != nil {
		return &Usage{Status: "unknown", CheckedAt: now.UTC(), Windows: []Window{}}
	}
	windows := []Window{}
	for _, k := range []struct {
		key, id string
		minutes int64
	}{{"five_hour", "five_hour", 300}, {"seven_day", "weekly", 10080}, {"seven_day_opus", "weekly_opus", 10080}, {"seven_day_sonnet", "weekly_sonnet", 10080}} {
		var w *struct {
			Utilization *float64   `json:"utilization"`
			ResetsAt    *time.Time `json:"resets_at"`
		}
		if json.Unmarshal(in[k.key], &w) != nil || w == nil || w.Utilization == nil {
			continue
		}
		minutes := k.minutes
		out := Window{ID: k.id, UsedPercent: clampPercent(*w.Utilization), WindowMinutes: &minutes}
		if w.ResetsAt != nil {
			t := w.ResetsAt.UTC()
			out.ResetsAt = &t
		}
		windows = append(windows, out)
	}
	var limits []struct {
		Percent  *float64   `json:"percent"`
		ResetsAt *time.Time `json:"resets_at"`
	}
	_ = json.Unmarshal(in["limits"], &limits)
	blocked := false
	for _, l := range limits {
		if l.Percent != nil && *l.Percent >= 100 {
			blocked = true
		}
	}
	if len(windows) == 0 && !blocked {
		return &Usage{Status: "unknown", CheckedAt: now.UTC(), Windows: windows}
	}
	return usageStatus(windows, blocked, now)
}

// ClaudePlan maps the OAuth profile organization type (e.g. claude_pro) to a short plan name.
func ClaudePlan(raw []byte) string {
	var in struct {
		Account struct {
			Max bool `json:"has_claude_max"`
			Pro bool `json:"has_claude_pro"`
		} `json:"account"`
		Organization struct {
			Type string `json:"organization_type"`
		} `json:"organization"`
	}
	if json.Unmarshal(raw, &in) != nil {
		return ""
	}
	if plan := strings.TrimPrefix(in.Organization.Type, "claude_"); plan != "" && plain(plan, 40) {
		return plan
	}
	if in.Account.Max {
		return "max"
	}
	if in.Account.Pro {
		return "pro"
	}
	return ""
}

var urlRE = regexp.MustCompile(`https://[^\s\x07\x1b"'<>\\]+`)

// LoginURL extracts the first HTTPS URL from CLI output, including OSC-8 hyperlinks.
func LoginURL(output string) string {
	return urlRE.FindString(output)
}

var versionTokenRE = regexp.MustCompile(`[0-9]+\.[0-9]+(\.[0-9]+)?([-.+][0-9A-Za-z.]+)?`)

func cliVersion(output string) *string {
	if v := versionTokenRE.FindString(output); v != "" && len(v) <= 40 {
		return &v
	}
	return nil
}

// Classify maps CLI failure text to a public code; the text itself is never published.
func Classify(text string) string {
	t := strings.ToLower(text)
	for _, s := range []string{"usage limit", "rate limit", "rate_limit", "usage_limit", "hit your limit", "limit reached", "limit exceeded", "out of codex messages", "quota", "too many requests", "429", "session limit", "weekly limit"} {
		if strings.Contains(t, s) {
			return "AI_LIMIT_EXHAUSTED"
		}
	}
	for _, s := range []string{"unauthorized", "401", "not logged in", "/login", "login required", "please log in", "log in again", "authentication", "invalid api key", "oauth token", "token expired", "token has expired", "refresh token", "sign in again"} {
		if strings.Contains(t, s) {
			return "AI_AUTH_REQUIRED"
		}
	}
	return ""
}
