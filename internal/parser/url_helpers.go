package parser

import (
	"strings"
	"unicode"
)

func splitScheme(url string) (string, string, bool) {
	idx := strings.Index(url, ":")
	if idx <= 0 {
		return "", url, false
	}
	scheme := url[:idx]
	for i, r := range scheme {
		if i == 0 {
			if !unicode.IsLetter(r) {
				return "", url, false
			}
			continue
		}
		if !isAlphaNumeric(r) && r != '+' && r != '-' && r != '.' {
			return "", url, false
		}
	}
	return scheme, url[idx+1:], true
}

func splitHostPath(rest string) (string, string) {
	idx := len(rest)
	for i, r := range rest {
		if r == '/' || r == '?' || r == '#' {
			idx = i
			break
		}
	}
	return rest[:idx], rest[idx:]
}

func shouldApplyPrefix(url string) bool {
	if url == "" {
		return false
	}
	if strings.HasPrefix(url, "//") || strings.HasPrefix(url, "/") || strings.HasPrefix(url, "./") || strings.HasPrefix(url, "../") || strings.HasPrefix(url, "#") {
		return false
	}
	if _, _, ok := splitScheme(url); ok {
		return false
	}
	return true
}

func encodeURLFragment(text string, options Options, allowPlus bool, allowColon bool, allowAmp bool, allowEquals bool) string {
	safe := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	if allowPlus {
		safe += "+"
	}
	if allowColon {
		safe += ":"
	}
	safe += "/?#[]@%"
	if allowEquals {
		safe += "="
	}
	if allowAmp {
		safe += "&"
	}
	if !options.Restricted {
		safe += "!$'*;,"
	}
	var builder strings.Builder
	for _, b := range []byte(text) {
		if b < 0x80 && strings.ContainsRune(safe, rune(b)) {
			builder.WriteByte(b)
			continue
		}
		builder.WriteString("%")
		builder.WriteString(strings.ToUpper(hexByte(b)))
	}
	return builder.String()
}

func hexByte(b byte) string {
	const hexdigits = "0123456789ABCDEF"
	return string([]byte{hexdigits[b>>4], hexdigits[b&0x0F]})
}
