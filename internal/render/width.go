package render

import (
	"regexp"

	"github.com/mattn/go-runewidth"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func StripANSI(value string) string { return ansiPattern.ReplaceAllString(value, "") }

func VisibleWidth(value string) int { return runewidth.StringWidth(StripANSI(value)) }

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return runewidth.Truncate(value, width, "…")
}

func truncateTail(value string, width int, ascii bool) string {
	tail := "…"
	if ascii {
		tail = "."
	}
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(value) <= width {
		return value
	}
	if width == 1 {
		return tail
	}
	return runewidth.Truncate(value, width, tail)
}
