package parser

import "strings"

func parseHtmlWrapper(trimmed string, lines []string) (string, string, map[string]string, bool) {
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(trimmed, "<p") && strings.Contains(lower, ">") && strings.HasSuffix(trimmed, "</p>") {
		start := strings.Index(trimmed, ">")
		end := strings.LastIndex(trimmed, "</p>")
		if start != -1 && end != -1 && end > start {
			inner := trimmed[start+1 : end]
			attrs := parseHtmlAttributes(trimmed[:start+1])
			return "p", inner, attrs, true
		}
	}
	if strings.HasPrefix(trimmed, "<pre") && strings.Contains(lower, ">") && strings.HasSuffix(trimmed, "</pre>") {
		start := strings.Index(trimmed, ">")
		end := strings.LastIndex(trimmed, "</pre>")
		if start != -1 && end != -1 && end > start {
			inner := trimmed[start+1 : end]
			attrs := parseHtmlAttributes(trimmed[:start+1])
			return "pre", inner, attrs, true
		}
	}
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") && isHtmlTagStart(trimmed) {
		if isInlineHtmlTag(trimmed) {
			return "", "", nil, false
		}
		if isVoidHtmlTag(trimmed) {
			if strings.Contains(trimmed, "</") {
				return "", "", nil, false
			}
			return "", trimmed, nil, true
		}
	}
	return "", "", nil, false
}

func parseHtmlAttributes(tag string) map[string]string {
	start := strings.Index(tag, " ")
	end := strings.LastIndex(tag, ">")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	attrs := map[string]string{}
	chunks := strings.Fields(tag[start:end])
	for _, chunk := range chunks {
		if !strings.Contains(chunk, "=") {
			continue
		}
		parts := strings.SplitN(chunk, "=", 2)
		key := parts[0]
		val := strings.Trim(parts[1], "\"'")
		attrs[key] = val
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func isHtmlTagStart(tag string) bool {
	if len(tag) < 3 {
		return false
	}
	trimmed := strings.TrimSpace(tag)
	if !strings.HasPrefix(trimmed, "<") || len(trimmed) < 2 {
		return false
	}
	second := trimmed[1]
	if second == '/' || second == '!' {
		return true
	}
	return (second >= 'a' && second <= 'z') || (second >= 'A' && second <= 'Z')
}

func isVoidHtmlTag(tag string) bool {
	name := htmlTagName(tag)
	if name == "" {
		return false
	}
	voids := map[string]bool{
		"br":   true,
		"hr":   true,
		"img":  true,
		"meta": true,
		"link": true,
	}
	return voids[name]
}

func isDividerBlock(text string) bool {
	idx := 0
	hasTag := false
	for idx < len(text) {
		pos := strings.Index(text[idx:], "<")
		if pos == -1 {
			return strings.TrimSpace(text[idx:]) == "" && hasTag
		}
		pos += idx
		if strings.TrimSpace(text[idx:pos]) != "" {
			return false
		}
		end := strings.Index(text[pos:], ">")
		if end == -1 {
			return false
		}
		end += pos
		tag := text[pos : end+1]
		if strings.Contains(tag[1:], "<") {
			return false
		}
		name := htmlTagName(tag)
		if name == "" || (name != "br" && name != "hr" && name != "img") {
			return false
		}
		hasTag = true
		idx = end + 1
	}
	return hasTag
}

func htmlTagName(tag string) string {
	trimmed := strings.TrimSpace(tag)
	if !strings.HasPrefix(trimmed, "<") {
		return ""
	}
	trimmed = strings.TrimPrefix(trimmed, "<")
	trimmed = strings.TrimPrefix(trimmed, "/")
	name := ""
	for i := 0; i < len(trimmed); i++ {
		c := trimmed[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == ':' || c == '-' {
			name += string(c)
			continue
		}
		break
	}
	return strings.ToLower(name)
}
