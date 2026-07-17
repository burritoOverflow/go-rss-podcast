package main

import (
	"fmt"
	"strconv"
	"strings"
)

// stripHTML removes HTML tags from s and collapses whitespace, producing a
// plain-text approximation suitable for terminal rendering.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// formatDuration converts an iTunes duration value (seconds, or already
// formatted like "H:MM:SS") into a "H:MM:SS" / "MM:SS" display string.
func formatDuration(d string) string {
	if d == "" {
		return "-"
	}
	if sec, err := strconv.Atoi(d); err == nil && sec >= 0 {
		h := sec / 3600
		m := (sec % 3600) / 60
		s := sec % 60
		if h > 0 {
			return fmt.Sprintf("%d:%02d:%02d", h, m, s)
		}
		return fmt.Sprintf("%02d:%02d", m, s)
	}
	return d
}

// truncate shortens s to at most max runes, appending "..." when truncation
// occurs.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// padRight pads s with trailing spaces to width n, truncating (with "...")
// if s is longer than n.
func padRight(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > n {
		if n <= 3 {
			return string(runes[:n])
		}
		return string(runes[:n-3]) + "..."
	}
	return s + strings.Repeat(" ", n-len(runes))
}

// padLeft pads s with leading zeros to width n.
func padLeft(s string, n int) string {
	runes := []rune(s)
	if len(runes) >= n {
		return string(runes)
	}
	return strings.Repeat("0", n-len(runes)) + s
}
