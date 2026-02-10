package parser

import (
	"strings"

	"github.com/rcarmo/go-textile/internal/document"
)

func parseTableBlock(lines []string, options Options) (*document.D, error) {
	idx := 0
	table := document.New("table")
	if len(lines) == 0 {
		return table, nil
	}
	trimmed := strings.TrimSpace(lines[0])
	if strings.HasPrefix(trimmed, "table") {
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "table"))
		attrs, content, ok := parseTagLine(rest, options)
		if ok {
			table.Attr = attrs
			if content != "" {
				if table.Attr == nil {
					table.Attr = map[string]string{}
				}
				table.Attr["summary"] = content
			}
		}
		idx++
	}
	var currentRow string
	var currentAttrs map[string]string
	hasRows := false
	currentParent := table
	flushRow := func() error {
		if strings.TrimSpace(currentRow) == "" {
			return nil
		}
		cells := splitTableRow(currentRow)
		empty := true
		for _, cell := range cells {
			if strings.TrimSpace(cell) != "" {
				empty = false
				break
			}
		}
		trimmedRow := strings.TrimSpace(currentRow)
		if empty && trimmedRow == "|" {
			currentRow = ""
			currentAttrs = nil
			return nil
		}
		row := document.New("tr")
		row.Attr = currentAttrs
		for _, cell := range cells {
			node, err := parseTableCell(cell, options)
			if err != nil {
				return err
			}
			row.AddChild(node)
		}
		currentParent.AddChild(row)
		hasRows = true
		currentRow = ""
		currentAttrs = nil
		return nil
	}
	for ; idx < len(lines); idx++ {
		line := strings.TrimRight(lines[idx], "\r")
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		pipeCount := strings.Count(trimmedLine, "|")
		if strings.HasPrefix(trimmedLine, "|=") && !hasRows && currentRow == "" {
			if err := flushRow(); err != nil {
				return table, err
			}
			captionLine := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "|="))
			captionLine = strings.TrimSpace(strings.TrimSuffix(captionLine, "|"))
			attrs, content, ok := parseTagLine(captionLine, options)
			if ok {
				caption, err := parseInline(content, "caption", attrs, options)
				if err != nil {
					return table, err
				}
				table.AddChild(caption)
			}
			continue
		}
		if strings.HasPrefix(trimmedLine, "|:") && !hasRows && currentRow == "" {
			if err := flushRow(); err != nil {
				return table, err
			}
			colgroup, err := parseColgroupLine(trimmedLine, options)
			if err != nil {
				return table, err
			}
			if colgroup != nil {
				table.AddChild(colgroup)
			}
			continue
		}
		if pipeCount == 1 && (strings.HasPrefix(trimmedLine, "|^") || strings.HasPrefix(trimmedLine, "|~") || strings.HasPrefix(trimmedLine, "|-") ) {
			if err := flushRow(); err != nil {
				return table, err
			}
			section, ok := parseTableSection(trimmedLine, options)
			if ok {
				table.AddChild(section)
				currentParent = section
				continue
			}
		}
		rowAttrs, rowLine := parseTableRowPrefix(line, options)
		if strings.HasPrefix(strings.TrimSpace(rowLine), "|") {
			if err := flushRow(); err != nil {
				return table, err
			}
			currentRow = rowLine
			currentAttrs = rowAttrs
			continue
		}
		if currentRow != "" {
			currentRow += "\n" + line
		}
	}
	if err := flushRow(); err != nil {
		return table, err
	}
	return table, nil
}

func parseTableRowPrefix(line string, options Options) (map[string]string, string) {
	trimmed := strings.TrimSpace(line)
	pipe := strings.Index(trimmed, "|")
	if pipe == -1 {
		return nil, line
	}
	prefix := strings.TrimSpace(trimmed[:pipe])
	if prefix == "" {
		return nil, line
	}
	attrs, _, ok := parseTagLine(prefix, options)
	if !ok {
		return nil, line
	}
	return attrs, strings.TrimSpace(trimmed[pipe:])
}

func parseTableCell(cell string, options Options) (*document.D, error) {
	raw := cell
	working := strings.TrimLeft(raw, " ")
	leading := raw[:len(raw)-len(working)]
	tag := "td"
	attrs := map[string]string{}
	hasMarker := false
	if strings.HasPrefix(working, "_") {
		headerCandidate := working[1:]
		headerOK := false
		switch {
		case strings.HasPrefix(headerCandidate, "."):
			headerOK = true
		case strings.HasPrefix(headerCandidate, "<") || strings.HasPrefix(headerCandidate, ">") || strings.HasPrefix(headerCandidate, "="):
			headerOK = true
		case strings.HasPrefix(headerCandidate, "^") || strings.HasPrefix(headerCandidate, "~"):
			headerOK = true
		case strings.HasPrefix(headerCandidate, "(") || strings.HasPrefix(headerCandidate, "{") || strings.HasPrefix(headerCandidate, "["):
			headerOK = true
		case strings.HasPrefix(headerCandidate, "\\") || strings.HasPrefix(headerCandidate, "/"):
			headerOK = true
		}
		if headerOK {
			hasMarker = true
			tag = "th"
			working = headerCandidate
			if style, rest, ok := parseCellAlignment(working); ok {
				applyCellStyle(attrs, style)
				working = rest
			}
			if strings.HasPrefix(working, ".") {
				working = working[1:]
			}
		}
	}
	if tag == "td" {
		if style, rest, ok := parseCellAlignment(working); ok {
			hasMarker = true
			applyCellStyle(attrs, style)
			working = rest
		}
	}
	if strings.HasPrefix(working, "\\") {
		hasMarker = true
		working = strings.TrimPrefix(working, "\\")
		span, rest := parseLeadingNumber(working)
		if span != "" {
			attrs["colspan"] = span
			if strings.HasPrefix(rest, ".") {
				working = rest[1:]
			} else if style, remaining, ok := parseCellAlignment(rest); ok {
				applyCellStyle(attrs, style)
				working = remaining
				if strings.HasPrefix(working, ".") {
					working = working[1:]
				}
			} else {
				working = rest
			}
		}
	}
	if strings.HasPrefix(working, "/") {
		hasMarker = true
		working = strings.TrimPrefix(working, "/")
		span, rest := parseLeadingNumber(working)
		if span != "" && strings.HasPrefix(rest, ".") {
			attrs["rowspan"] = span
			working = rest[1:]
		}
	}
	if strings.HasPrefix(working, "(") || strings.HasPrefix(working, "{") || strings.HasPrefix(working, "[") {
		cellAttrs, content, ok := parseTagLine(working, options)
		if ok {
			hasMarker = true
			for k, v := range cellAttrs {
				attrs[k] = v
			}
			working = content
		}
	}
	if hasMarker {
		working = strings.TrimLeft(working, " ")
	} else {
		working = leading + working
	}
	if len(attrs) == 0 {
		attrs = nil
	}
	if strings.Contains(working, "\n") {
		trimmedContent := strings.TrimSpace(working)
		if isListStart(trimmedContent, '*') || isListStart(trimmedContent, '#') {
			list, err := parseMixedList(strings.Split(working, "\n"), options)
			if err != nil {
				return nil, err
			}
			cell := document.New(tag)
			cell.Attr = attrs
			if list != nil {
				cell.AddChild(list)
			}
			return cell, nil
		}
		if (strings.HasPrefix(trimmedContent, ";") || strings.HasPrefix(trimmedContent, ":")) {
			dl, err := parseClassicDefinitionList(strings.Split(working, "\n"), options)
			if err != nil {
				return nil, err
			}
			if dl != nil {
				cell := document.New(tag)
				cell.Attr = attrs
				cell.AddChild(dl)
				return cell, nil
			}
		}
		if strings.HasPrefix(trimmedContent, "-") && strings.Contains(trimmedContent, ":=") {
			dl, err := parseDefinitionList(strings.Split(working, "\n"), options)
			if err != nil {
				return nil, err
			}
			cell := document.New(tag)
			cell.Attr = attrs
			cell.AddChild(dl)
			return cell, nil
		}
	}
	node, err := parseInline(working, tag, attrs, options)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func parseLeadingNumber(text string) (string, string) {
	idx := 0
	for idx < len(text) && text[idx] >= '0' && text[idx] <= '9' {
		idx++
	}
	if idx == 0 {
		return "", text
	}
	return text[:idx], text[idx:]
}
