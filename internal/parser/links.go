package parser

import (
	"strings"
	"unicode"
)

func parseInlineAttrSequence(text string, options Options) (map[string]string, string) {
	attrs := map[string]string{}
	original := strings.TrimLeft(text, " \t")
	rest := original
	fragments := []string{}
	consumed := false
	for strings.HasPrefix(rest, "(") || strings.HasPrefix(rest, "{") || strings.HasPrefix(rest, "[") {
		fragment, remaining := extractInlineAttrFragment(rest)
		if fragment == "" {
			break
		}
		if options.Restricted && strings.HasPrefix(fragment, "(") && strings.Contains(fragment, " ") {
			if len(fragments) == 0 {
				return nil, original
			}
			rest = strings.TrimLeft(fragment+remaining, " \t")
			break
		}
		fragments = append(fragments, fragment)
		fragmentAttrs := parseAttributes(fragment, options)
		if fragmentAttrs == nil {
			if options.Restricted && (strings.HasPrefix(fragment, "(") || strings.HasPrefix(fragment, "{")) {
				consumed = true
				rest = strings.TrimLeft(remaining, " \t")
				continue
			}
			if len(attrs) == 0 && !consumed {
				return nil, original
			}
			break
		}
		for k, v := range fragmentAttrs {
			attrs[k] = v
		}
		consumed = true
		if remaining == rest {
			break
		}
		if len(remaining) > 0 && (remaining[0] == ' ' || remaining[0] == '\t') {
			rest = strings.TrimLeft(remaining, " \t")
			break
		}
		rest = remaining
	}
	rest = strings.TrimLeft(rest, " \t")
	if len(fragments) > 1 {
		allParen := true
		for _, fragment := range fragments {
			if !strings.HasPrefix(fragment, "(") {
				allParen = false
				break
			}
		}
		if allParen {
			attrs = parseAttributes(fragments[0], options)
			rest = strings.TrimLeft(original[len(fragments[0]):], " \t")
			consumed = true
		}
	}
	if len(attrs) == 0 {
		if !consumed {
			return nil, rest
		}
		if rest == "" {
			return nil, text
		}
		return nil, rest
	}
	if rest == "" {
		return nil, text
	}
	return attrs, rest
}

func extractInlineAttrFragment(rest string) (string, string) {
	if rest == "" {
		return "", rest
	}
	switch rest[0] {
	case '(':
		end := strings.Index(rest, ")")
		if end == -1 {
			return "", rest
		}
		return rest[:end+1], rest[end+1:]
	case '{':
		end := strings.Index(rest, "}")
		if end == -1 {
			return "", rest
		}
		return rest[:end+1], rest[end+1:]
	case '[':
		end := strings.Index(rest, "]")
		if end == -1 {
			return "", rest
		}
		return rest[:end+1], rest[end+1:]
	default:
		return "", rest
	}
}

func extractLinkTitle(text string) (string, string) {
	if strings.HasSuffix(text, ")") {
		open := strings.LastIndex(text, "(")
		if open > 0 {
			prefix := text[:open]
			if !hasTextOutsideParens(prefix) {
				if !strings.Contains(prefix, ")") {
					return text, ""
				}
			}
			title := text[open+1 : len(text)-1]
			if !strings.Contains(title, "(") {
				return strings.TrimSpace(text[:open]), title
			}
		}
	}
	return text, ""
}

func hasTextOutsideParens(text string) bool {
	depth := 0
	for _, r := range text {
		switch r {
		case '(':
			depth++
			continue
		case ')':
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth == 0 && !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func trimLinkPunctuation(url string) (string, int) {
	punct := ".,?!:;|*)\""
	trimCount := 0
	for len(url) > 0 {
		last := url[len(url)-1]
		if strings.ContainsRune(punct, rune(last)) {
			if last == ')' && strings.Contains(url, "(") {
				break
			}
			url = url[:len(url)-1]
			trimCount++
			continue
		}
		break
	}
	return url, trimCount
}

func isAllowedScheme(url string) bool {
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "#") {
		return true
	}
	lower := strings.ToLower(url)
	allowed := []string{
		"http://",
		"https://",
		"ftp://",
		"sftp://",
		"mailto:",
		"mailto://",
		"tel:",
		"callto:",
		"file://",
		"file:/",
	}
	for _, prefix := range allowed {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return !strings.Contains(lower, ":")
}

func isAllowedSchemeRestricted(url string) bool {
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "#") {
		return true
	}
	lower := strings.ToLower(url)
	allowed := []string{
		"http://",
		"https://",
		"ftp://",
		"mailto:",
		"mailto://",
		"tel:",
		"callto:",
	}
	for _, prefix := range allowed {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return !strings.Contains(lower, ":")
}

func sanitizeURL(url string, options Options) string {
	if url == "" {
		return url
	}
	if options.Restricted {
		url = strings.ReplaceAll(url, "<", "&lt;")
		url = strings.ReplaceAll(url, ">", "&gt;")
	}
	scheme, rest, ok := splitScheme(url)
	if !ok {
		return encodeURLPathAndQuery(url, options, true, false)
	}
	lower := strings.ToLower(scheme)
	switch lower {
	case "mailto", "callto":
		return scheme + ":" + encodeURLFragment(rest, options, false, false, true, true)
	case "tel":
		return scheme + ":" + encodeURLFragment(rest, options, true, false, true, false)
	case "file":
		return scheme + ":" + sanitizeFileURL(rest, options)
	default:
		if strings.HasPrefix(rest, "//") {
			host, path := splitHostPath(rest[2:])
			return scheme + "://" + host + encodeURLPathAndQuery(path, options, true, false)
		}
		return scheme + ":" + encodeURLPathAndQuery(rest, options, true, false)
	}
}

func sanitizeFileURL(rest string, options Options) string {
	if strings.HasPrefix(rest, "//") {
		host, path := splitHostPath(rest[2:])
		if host == "" {
			return encodeURLPathAndQuery(path, options, true, false)
		}
		return "//" + host + encodeURLPathAndQuery(path, options, true, false)
	}
	return encodeURLPathAndQuery(rest, options, true, false)
}

func sanitizeImageURL(url string, options Options) string {
	if options.Restricted {
		return sanitizeURL(url, options)
	}
	return url
}

func displayURL(url string) string {
	display := url
	lower := strings.ToLower(display)
	prefixes := []string{
		"http://",
		"https://",
		"ftp://",
		"sftp://",
		"mailto://",
		"mailto:",
		"tel:",
		"callto:",
		"file://",
		"file:/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			display = display[len(prefix):]
			break
		}
	}
	if strings.HasPrefix(lower, "file:/") && strings.HasPrefix(display, "/") {
		display = strings.TrimPrefix(display, "/")
	}
	display = strings.TrimSuffix(display, "/")
	display = strings.ReplaceAll(display, "%3A", ":")
	display = strings.ReplaceAll(display, "%3a", ":")
	return display
}
