package investigate

import (
	"regexp"
	"strings"
)

var (
	secretAssign = regexp.MustCompile(`(?i)\b(api[_-]?key|token|password|secret|authorization|cookie)\b\s*[:=]\s*\S+`)
	bearer       = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._\-+/=]+`)
	skKey        = regexp.MustCompile(`\bsk-[a-zA-Z0-9_\-]{8,}\b`)
	connString   = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://[^\s:]+:[^\s@]+@[^\s]+`)
)

// Sanitize redacts credential-shaped text. Evidence is still untrusted after this.
func Sanitize(value string) string {
	value = clip(value, 1500)
	value = skKey.ReplaceAllString(value, "[redacted]")
	value = bearer.ReplaceAllString(value, "Bearer [redacted]")
	value = connString.ReplaceAllString(value, "[redacted-connection]")
	value = secretAssign.ReplaceAllString(value, "$1=[redacted]")
	return value
}

func SanitizeItems(items []Item) []Item {
	out := make([]Item, len(items))
	for i, item := range items {
		item.Title = Sanitize(item.Title)
		item.Body = Sanitize(item.Body)
		item.Source = Sanitize(item.Source)
		out[i] = item
	}
	return out
}

func clip(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "…"
}

func containsInstruction(value string) bool {
	return strings.Contains(strings.ToLower(value), "ignore previous instructions")
}
