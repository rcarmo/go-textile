package parser

import "strings"

func renderPreContent(content string) string {
	var builder strings.Builder
	idx := 0
	for idx < len(content) {
		start := strings.Index(content[idx:], "@")
		if start == -1 {
			builder.WriteString(escapeHTML(content[idx:]))
			break
		}
		start += idx
		end := strings.Index(content[start+1:], "@")
		if end == -1 {
			builder.WriteString(escapeHTML(content[idx:]))
			break
		}
		end = start + 1 + end
		builder.WriteString(escapeHTML(content[idx:start]))
		builder.WriteString("<code>")
		builder.WriteString(escapeHTML(content[start+1 : end]))
		builder.WriteString("</code>")
		idx = end + 1
	}
	return builder.String()
}
