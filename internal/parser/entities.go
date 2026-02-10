package parser

import "strings"

func consumeEntity(text string) (string, int, bool) {
	if !strings.HasPrefix(text, "&") {
		return "", 0, false
	}
	end := strings.Index(text, ";")
	if end == -1 {
		return "", 0, false
	}
	candidate := text[:end+1]
	if len(candidate) < 3 {
		return "", 0, false
	}
	return candidate, end + 1, true
}
