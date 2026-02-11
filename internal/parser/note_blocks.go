package parser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rcarmo/go-textile/internal/document"
)

func parseNoteDefinition(blk block, options Options) bool {
	line := strings.TrimSpace(blk.Lines[0])
	if !strings.HasPrefix(line, "note#") {
		return false
	}
	rest := strings.TrimPrefix(line, "note#")
	idx := 0
	for idx < len(rest) && rest[idx] != ' ' && rest[idx] != '\t' && rest[idx] != '^' && rest[idx] != '(' && rest[idx] != '{' && rest[idx] != '[' && rest[idx] != '.' {
		idx++
	}
	if idx == 0 {
		return false
	}
	label := rest[:idx]
	hasCaret := false
	if idx < len(rest) && rest[idx] == '^' {
		hasCaret = true
		idx++
	}
	rest = strings.TrimSpace(rest[idx:])
	attrs := map[string]string{}
	if strings.HasPrefix(rest, "(") || strings.HasPrefix(rest, "{") || strings.HasPrefix(rest, "[") {
		fragment, remaining := extractAttributeFragment(rest)
		attrs = parseAttributes(fragment, options)
		rest = strings.TrimSpace(remaining)
	}
	if strings.HasPrefix(rest, ".") {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "."))
	}
	content := strings.TrimSpace(rest)
	content = appendBlockLines(content, blk.Lines[1:])
	def, ok := noteDefs[label]
	if !ok {
		def = &noteDef{label: label}
		noteDefs[label] = def
	}
	if def.order == 0 {
		noteDefOrder = append(noteDefOrder, label)
		noteDefSeqNum++
		def.order = noteDefSeqNum
	}
	if len(attrs) == 0 {
		attrs = nil
	}
	def.attrs = attrs
	def.content = content
	def.hasCaret = hasCaret
	def.defined = true
	return true
}

type noteListSpec struct {
	list                *document.D
	marker              string
	includeUnreferenced bool
	noBacklinks         bool
	singleBackref       bool
}

func parseNoteList(blk block, options Options) (*document.D, bool, error) {
	line := strings.TrimSpace(blk.Lines[0])
	if !strings.HasPrefix(line, "notelist") {
		return nil, false, nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "notelist"))
	attrs := map[string]string{}
	if strings.HasPrefix(rest, "(") || strings.HasPrefix(rest, "{") || strings.HasPrefix(rest, "[") {
		fragment, remaining := extractAttributeFragment(rest)
		attrs = parseAttributes(fragment, options)
		rest = strings.TrimSpace(remaining)
	}
	marker := "a"
	includeUnreferenced := false
	noBacklinks := false
	singleBackref := false
	if strings.HasPrefix(rest, "!") {
		noBacklinks = true
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "!"))
	}
	if strings.HasPrefix(rest, ":") {
		opts := strings.TrimSpace(strings.TrimPrefix(rest, ":"))
		if opts != "" {
			runes := []rune(opts)
			marker = string(runes[0])
			opts = string(runes[1:])
		}
		if strings.Contains(opts, "+") {
			includeUnreferenced = true
		}
		if strings.Contains(opts, "^") {
			singleBackref = true
		}
	}
	if len(noteDefOrder) == 0 {
		noteNoLinks = true
	}
	if len(noteRefOrder) == 0 && !includeUnreferenced {
		return nil, true, nil
	}
	ol := document.New("ol")
	if len(attrs) > 0 {
		ol.Attr = attrs
	}
	noteListSpecs = append(noteListSpecs, noteListSpec{
		list:                ol,
		marker:              marker,
		includeUnreferenced: includeUnreferenced,
		noBacklinks:         noBacklinks,
		singleBackref:       singleBackref,
	})
	return ol, true, nil
}

func buildNoteListItem(def *noteDef, marker string, noBacklinks bool, singleBackref bool, options Options) (*document.D, error) {
	li := document.New("li")
	if def.attrs != nil {
		li.Attr = map[string]string{}
		for k, v := range def.attrs {
			if noBacklinks && k == "id" {
				continue
			}
			li.Attr[k] = v
		}
		if len(li.Attr) == 0 {
			li.Attr = nil
		}
	}
	localNoBacklinks := noBacklinks || def.refCount == 0
	if !localNoBacklinks {
		count := def.refCount
		if def.hasCaret {
			singleBackref = true
		}
		if singleBackref && count > 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			markerText := marker
			if isLetterMarker(marker) {
				markerRune, _ := utf8.DecodeRuneInString(marker)
				markerText = string(markerRune + rune(i))
			}
			href := ""
			if def.defined && !noteNoLinks {
				href = "#noteref"
			}
			sup, _ := buildSupLink(document.Text(markerText, true), href, nil, true)
			li.AddChild(sup)
			if i < count-1 {
				li.AddChild(document.Text(" ", true))
			}
		}
		if def.defined {
			span := document.New("span")
			if !noteNoLinks {
				span.Attr = map[string]string{"id": "note"}
			}
			span.AddChild(document.Text(" ", true))
			li.AddChild(span)
		}
	} else if noBacklinks {
		span := document.New("span")
		span.AddChild(document.Text(" ", true))
		li.AddChild(span)
	}
	content := def.content
	undefined := false
	if content == "" && def.index > 0 {
		content = fmt.Sprintf("Undefined Note [#%d].", def.index)
		undefined = true
	}
	if content != "" {
		if undefined {
			if !localNoBacklinks {
				li.AddChild(document.Text(" ", true))
			}
			escaped, _ := escapeAndGlyphWithPrev(content, 0)
			li.AddChild(document.Text(escaped, true))
			return li, nil
		}
		parsed, err := parseInline(content, "", nil, options)
		if err != nil {
			return nil, err
		}
		li.Children = append(li.Children, parsed.Children...)
	}
	return li, nil
}

func isLetterMarker(marker string) bool {
	if marker == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(marker)
	return unicode.IsLetter(r) && len([]rune(marker)) == 1
}

