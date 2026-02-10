package parser

import (
	"strings"

	"github.com/rcarmo/go-textile/internal/document"
)

func parseTableSection(line string, options Options) (*document.D, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || len(trimmed) < 2 {
		return nil, false
	}
	marker := trimmed[1]
	var tag string
	switch marker {
	case '^':
		tag = "thead"
	case '~':
		tag = "tfoot"
	case '-':
		tag = "tbody"
	default:
		return nil, false
	}
	rest := strings.TrimSpace(trimmed[2:])
	rest = strings.TrimSpace(strings.TrimSuffix(rest, "|"))
	attrs, _, ok := parseTagLine(rest, options)
	if !ok {
		return nil, false
	}
	section := document.New(tag)
	section.Attr = attrs
	return section, true
}

func parseColgroupLine(line string, options Options) (*document.D, error) {
	cells := splitTableRow(line)
	if len(cells) == 0 {
		return nil, nil
	}
	first := strings.TrimSpace(cells[0])
	if !strings.HasPrefix(first, ":") {
		return nil, nil
	}
	working := strings.TrimSpace(strings.TrimPrefix(first, ":"))
	colgroup := document.New("colgroup")
	attrs := map[string]string{}
	if strings.HasPrefix(working, "\\") {
		working = strings.TrimPrefix(working, "\\")
		span, rest := parseLeadingNumber(working)
		if span != "" && strings.HasPrefix(rest, ".") {
			attrs["span"] = span
			working = strings.TrimSpace(rest[1:])
		}
	}
	if working != "" {
		attrs["width"] = strings.TrimSpace(working)
	}
	if len(attrs) > 0 {
		colgroup.Attr = attrs
	}
	for _, cell := range cells[1:] {
		colAttrs := map[string]string{}
		cellText := strings.TrimSpace(cell)
		if cellText != "" && (strings.HasPrefix(cellText, "(") || strings.HasPrefix(cellText, "{") || strings.HasPrefix(cellText, "[")) {
			fragment, remaining := extractAttributeFragment(cellText)
			attrsParsed := parseAttributes(fragment, options)
			for k, v := range attrsParsed {
				colAttrs[k] = v
			}
			cellText = strings.TrimSpace(remaining)
		}
		if cellText != "" {
			colAttrs["width"] = cellText
		}
		col := document.New("col")
		if len(colAttrs) > 0 {
			col.Attr = colAttrs
		}
		colgroup.AddChild(col)
	}
	return colgroup, nil
}
