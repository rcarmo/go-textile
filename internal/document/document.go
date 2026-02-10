package document

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

type D struct {
	Tag      string
	Text     string
	Attr     map[string]string
	Children []*D
	Raw      bool
}

var UseHTML5 bool

func New(tag string) *D {
	return &D{Tag: tag, Children: make([]*D, 0)}
}

func Text(text string, raw bool) *D {
	return &D{Text: text, Raw: raw}
}

func (d *D) AddChild(c *D) {
	d.Children = append(d.Children, c)
}

func (d *D) ToHtml() string {
	output := renderNode(d, 0, true)
	if output != "" && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	return output
}

func renderNode(d *D, depth int, blockContext bool) string {
	if d.Tag == "" {
		if len(d.Children) > 0 {
			var builder strings.Builder
			for i, c := range d.Children {
				if i > 0 {
					builder.WriteString("\n\n")
				}
				builder.WriteString(renderNode(c, depth, true))
			}
			return builder.String()
		}
		if d.Raw {
			return d.Text
		}
		return html.EscapeString(d.Text)
	}

	if d.Tag == "inline" {
		return renderChildren(d, depth, false)
	}

	attr := renderAttributes(d.Attr)
	voidTags := map[string]bool{
		"br":   true,
		"img":  true,
		"hr":   true,
		"meta": true,
		"link": true,
		"col":  true,
	}
	blockContainers := map[string]bool{
		"blockquote": true,
		"ul":         true,
		"ol":         true,
		"dl":         true,
		"table":      true,
		"tr":         true,
		"thead":      true,
		"tbody":      true,
		"tfoot":      true,
		"colgroup":   true,
	}
	blockTags := map[string]bool{
		"p":       true,
		"pre":     true,
		"tr":      true,
		"li":      true,
		"dt":      true,
		"dd":      true,
		"td":      true,
		"th":      true,
		"caption": true,
	}
	for i := 1; i <= 6; i++ {
		blockTags[fmt.Sprintf("h%d", i)] = true
	}

	if strings.Contains(d.Tag, "+") {
		content := renderChildren(d, depth, false)
		format := "%s"
		parts := strings.Split(d.Tag, "+")
		for _, tag := range parts {
			format = fmt.Sprintf(format, fmt.Sprintf("<%s%s>%s</%s>", tag, attr, "%s", tag))
			attr = ""
		}
		return applyIndent(fmt.Sprintf(format, content), depth, blockContext)
	}

	if voidTags[d.Tag] {
		format := "<%s%s />"
		if UseHTML5 {
			format = "<%s%s>"
		}
		content := fmt.Sprintf(format, d.Tag, attr)
		if d.Tag == "br" {
			content += "\n"
		}
		return applyIndent(content, depth, blockContext)
	}

	if blockContainers[d.Tag] {
		childDepth := depth + 1
		if !blockContext {
			childDepth = 1
		}
		if d.Tag == "colgroup" {
			childDepth = depth
		}
		inner := renderChildren(d, childDepth, true)
		if d.Tag == "table" {
			inner = renderTableChildren(d, depth, blockContext)
		}
		open := fmt.Sprintf("<%s%s>", d.Tag, attr)
		close := fmt.Sprintf("</%s>", d.Tag)
		closeDepth := depth
		closeBlock := blockContext
		if listHasMixedChildren(d) {
			if !blockContext || depth == 0 {
				closeDepth = 1
				closeBlock = true
			}
		}
		if inner == "" {
			return applyIndent(open, depth, blockContext) + "\n" + applyIndent(close, closeDepth, closeBlock)
		}
		return applyIndent(open, depth, blockContext) + "\n" + inner + "\n" + applyIndent(close, closeDepth, closeBlock)
	}

	if d.Tag == "li" || d.Tag == "dd" {
		inlineContent := ""
		var blockParts []string
		for _, child := range d.Children {
			if blockContainers[child.Tag] {
				blockParts = append(blockParts, renderNode(child, depth, true))
				continue
			}
			inlineContent += renderNode(child, depth, false)
		}
		if len(blockParts) == 0 {
			return applyIndent(fmt.Sprintf("<%s%s>%s</%s>", d.Tag, attr, inlineContent, d.Tag), depth, blockContext)
		}
		inner := inlineContent
		if inner != "" {
			inner += "\n"
		}
		inner += strings.Join(blockParts, "\n")
		return applyIndent(fmt.Sprintf("<%s%s>%s</%s>", d.Tag, attr, inner, d.Tag), depth, blockContext)
	}

	content := renderChildren(d, depth, false)
	if blockTags[d.Tag] {
		content = renderChildren(d, depth, false)
		return applyIndent(fmt.Sprintf("<%s%s>%s</%s>", d.Tag, attr, content, d.Tag), depth, blockContext)
	}

	return fmt.Sprintf("<%s%s>%s</%s>", d.Tag, attr, content, d.Tag)
}

func renderChildren(d *D, depth int, blockContext bool) string {
	if len(d.Children) == 0 {
		return ""
	}
	var builder strings.Builder
	if blockContext {
		for i, c := range d.Children {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(renderNode(c, depth, true))
		}
		return builder.String()
	}
	for _, c := range d.Children {
		builder.WriteString(renderNode(c, depth, false))
	}
	return builder.String()
}

func applyIndent(text string, depth int, blockContext bool) string {
	if !blockContext || depth == 0 {
		return text
	}
	return strings.Repeat("\t", depth) + text
}

func listHasMixedChildren(list *D) bool {
	if list.Tag != "ul" && list.Tag != "ol" {
		return false
	}
	for _, li := range list.Children {
		for _, child := range li.Children {
			if child.Tag == "ul" || child.Tag == "ol" {
				if child.Tag != list.Tag {
					return true
				}
			}
		}
	}
	return false
}

func renderAttributes(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		if attrs[k] == "" && k != "alt" && k != "title" {
			continue
		}
		b.WriteString(" ")
		b.WriteString(k)
		b.WriteString("=\"")
		b.WriteString(escapeAttrValue(attrs[k]))
		b.WriteString("\"")
	}
	return b.String()
}

func escapeAttrValue(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		case '’':
			b.WriteString("&#8217;")
		case '‘':
			b.WriteString("&#8216;")
		case '“':
			b.WriteString("&#8220;")
		case '”':
			b.WriteString("&#8221;")
		case '©':
			b.WriteString("&#169;")
		case '®':
			b.WriteString("&#174;")
		case '™':
			b.WriteString("&#8482;")
		case '–':
			b.WriteString("&#8211;")
		case '—':
			b.WriteString("&#8212;")
		case '…':
			b.WriteString("&#8230;")
		case '×':
			b.WriteString("&#215;")
		case '°':
			b.WriteString("&#176;")
		case '±':
			b.WriteString("&#177;")
		case '¼':
			b.WriteString("&#188;")
		case '½':
			b.WriteString("&#189;")
		case '¾':
			b.WriteString("&#190;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (d *D) ToText() string {
	return ""
}
