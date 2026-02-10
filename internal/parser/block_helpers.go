package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rcarmo/go-textile/internal/document"
)

func parseInlineLines(lines []string, tag string, attrs map[string]string, options Options) (*document.D, error) {
	node := document.New(tag)
	node.Attr = attrs
	for i, line := range lines {
		parsed, err := parseInline(line, "", nil, options)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, parsed.Children...)
		if i < len(lines)-1 {
			node.AddChild(document.Text("\n", true))
		}
	}
	return node, nil
}

func hasLeadingWhitespace(line string) bool {
	if line == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(line)
	return unicode.IsSpace(r)
}

func containsBlockTag(lines []string) bool {
	for _, line := range lines {
		idx := 0
		for idx < len(line) {
			pos := strings.Index(line[idx:], "<")
			if pos == -1 {
				break
			}
			pos += idx
			if isInsideCodeSpan(line, pos) {
				idx = pos + 1
				continue
			}
			tag := line[pos:]
			if !isHtmlTagStart(tag) {
				idx = pos + 1
				continue
			}
			end := strings.Index(tag, ">")
			if end == -1 {
				idx = pos + 1
				continue
			}
			tagSegment := tag[:end+1]
			if strings.Contains(tagSegment[1:], "<") {
				idx = pos + 1
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(tagSegment), "</") {
				idx = pos + 1
				continue
			}
			name := htmlTagName(tagSegment)
			if name != "" && !isInlineTagName(name) {
				return true
			}
			idx = pos + 1
		}
	}
	return false
}

func isInsideCodeSpan(line string, pos int) bool {
	if pos <= 0 {
		return false
	}
	count := 0
	for i := 0; i < pos; i++ {
		if line[i] == '@' {
			count++
		}
	}
	if count%2 == 0 {
		return false
	}
	return strings.IndexByte(line[pos:], '@') != -1
}

func containsClosingBlockTag(lines []string) bool {
	for _, line := range lines {
		idx := 0
		for idx < len(line) {
			pos := strings.Index(line[idx:], "</")
			if pos == -1 {
				break
			}
			pos += idx
			if pos > 0 {
				prev, _ := utf8.DecodeLastRuneInString(line[:pos])
				if unicode.IsLetter(prev) || unicode.IsDigit(prev) {
					idx = pos + 2
					continue
				}
			}
			end := strings.Index(line[pos:], ">")
			if end == -1 {
				break
			}
			end += pos
			tag := line[pos : end+1]
			name := htmlTagName(tag)
			if name != "" && !isInlineTagName(name) {
				return true
			}
			idx = end + 1
		}
	}
	return false
}

func isInlineTagName(name string) bool {
	inline := map[string]bool{
		"a":      true,
		"span":   true,
		"cite":   true,
		"em":     true,
		"strong": true,
		"b":      true,
		"i":      true,
		"u":      true,
		"sup":    true,
		"sub":    true,
		"del":    true,
		"ins":    true,
		"code":   true,
		"br":     true,
		"img":    true,
	}
	return inline[name]
}

func rawBlockTagName(line string) string {
	if line == "" || hasLeadingWhitespace(line) {
		return ""
	}
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "<") {
		return ""
	}
	if strings.HasPrefix(trimmed, "</") {
		return ""
	}
	name := htmlTagName(trimmed)
	if name == "" {
		return ""
	}
	if !strings.Contains(name, ":") {
		return ""
	}
	return name
}

func blockContainsClosingTag(lines []string, name string) bool {
	if name == "" {
		return false
	}
	target := "</" + strings.ToLower(name)
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), target) {
			return true
		}
	}
	return false
}
