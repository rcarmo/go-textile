package parser

func parseCellAlignment(text string) (string, string, bool) {
	alignments := []struct {
		prefix string
		style  string
	}{
		{"<>.", "text-align:justify;"},
		{"<.", "text-align:left;"},
		{">.", "text-align:right;"},
		{"=.", "text-align:center;"},
		{"^.", "vertical-align:top;"},
		{"~.", "vertical-align:bottom;"},
	}
	for _, item := range alignments {
		if len(text) >= len(item.prefix) && text[:len(item.prefix)] == item.prefix {
			return item.style, text[len(item.prefix):], true
		}
	}
	return "", text, false
}

func applyCellStyle(attrs map[string]string, style string) {
	if style == "" {
		return
	}
	if attrs["style"] != "" {
		attrs["style"] = attrs["style"] + style
		return
	}
	attrs["style"] = style
}
