package parser

import "strings"

func extractImageDimensions(url string) (string, string) {
	base := url
	if idx := strings.LastIndex(base, "/"); idx != -1 {
		base = base[idx+1:]
	}
	if dot := strings.LastIndex(base, "."); dot != -1 {
		base = base[:dot]
	}
	sep := strings.LastIndexAny(base, "xX")
	if sep == -1 {
		return "", ""
	}
	width := base[:sep]
	height := base[sep+1:]
	if width == "" || height == "" {
		return "", ""
	}
	if !isDigits(width) || !isDigits(height) {
		return "", ""
	}
	return width, height
}
