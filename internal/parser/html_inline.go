package parser

import "strings"

func isInlineHtmlTag(tag string) bool {
	trimmed := strings.TrimSpace(tag)
	if !strings.HasPrefix(trimmed, "<") {
		return false
	}
	trimmed = strings.TrimPrefix(trimmed, "<")
	trimmed = strings.TrimPrefix(trimmed, "/")
	name := ""
	for i := 0; i < len(trimmed); i++ {
		c := trimmed[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			name += string(c)
			continue
		}
		break
	}
	name = strings.ToLower(name)
	if name == "" {
		return false
	}
	inline := map[string]bool{
		"a": true,
		"span": true,
		"cite": true,
		"em": true,
		"strong": true,
		"b": true,
		"i": true,
		"u": true,
		"sup": true,
		"sub": true,
		"del": true,
		"ins": true,
		"code": true,
	}
	return inline[name]
}
