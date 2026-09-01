package tui

import (
	"regexp"
	"strings"
)

var geosyncPattern = regexp.MustCompile(`^(?:0|[+-]?\d+|[+-]?\d{1,2}:\d{2}:\d{2})$`)

func validateGeosync(s string) bool {
	clean := strings.TrimSpace(s)
	if clean == "" || clean == "0" {
		return true
	}
	return geosyncPattern.MatchString(clean)
}

func parseExts(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';' || r == '\t'
	})
	var res []string
	seen := make(map[string]bool)
	for _, p := range parts {
		c := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(p)), ".")
		if c != "" && !seen[c] {
			seen[c] = true
			res = append(res, c)
		}
	}
	return res
}

func calculateWindow(total, current, windowSize int) (int, int) {
	if total <= windowSize {
		return 0, total
	}
	start := current - windowSize/2
	if start < 0 {
		start = 0
	}
	end := start + windowSize
	if end > total {
		end = total
		start = end - windowSize
	}
	return start, end
}
