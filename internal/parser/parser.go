package parser

import (
	"bufio"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rcarmo/go-textile/internal/document"
)

type Options struct {
	Lite                bool
	Restricted          bool
	Images              bool
	DimensionlessImages bool
	LinkRelationship    string
	LinkPrefix          string
	ImagePrefix         string
	LineWrap            int
	RawBlocks           bool
	BlockTags           bool
	HTML5               bool
}

type block struct {
	Lines []string
}

type noteDef struct {
	label      string
	content    string
	attrs      map[string]string
	hasCaret   bool
	referenced bool
	defined    bool
	index      int
	order      int
	refCount   int
}

type noteRefNode struct {
	sup    *document.D
	link   *document.D
	span   *document.D
	noLink bool
}

var (
	footnoteRefs  map[string]int
	linkRefs      map[string]string
	noteDefs      map[string]*noteDef
	noteRefOrder  []string
	noteIndex     map[string]int
	noteDefOrder  []string
	noteDefSeqNum   int
	noteRefNodes             map[string][]noteRefNode
	noteListSpecs            []noteListSpec
	noteNoLinks              bool
	restrictedParsing        bool
	lastOrderedIndexByLevel  map[int]int
)

func ParseDocument(text string, options Options) (*document.D, error) {
	document.UseHTML5 = options.HTML5
	footnoteRefs = map[string]int{}
	noteDefs = map[string]*noteDef{}
	noteRefOrder = []string{}
	noteIndex = map[string]int{}
	noteDefOrder = []string{}
	noteDefSeqNum = 0
	noteRefNodes = map[string][]noteRefNode{}
	noteListSpecs = nil
	noteNoLinks = false
	restrictedParsing = options.Restricted
	lastOrderedIndexByLevel = map[int]int{}
	lines := readLines(text)
	lines, linkRefs = collectLinkRefs(lines)
	blocks := splitBlocks(lines)
	d := document.New("")
	for _, blk := range blocks {
		if len(blk.Lines) == 0 {
			continue
		}
		parsed, err := parseBlock(blk, options)
		if err != nil {
			return nil, err
		}
		if parsed != nil {
			d.AddChild(parsed)
		}
	}
	finalizeNoteRefs()
	finalizeNoteLists(options)
	return d, nil
}

func readLines(text string) []string {
	lines := make([]string, 0, 256)
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(text) > 0 && strings.HasSuffix(text, "\n") {
		lines = append(lines, "")
	}
	return lines
}

func splitBlocks(lines []string) []block {
	blocks := make([]block, 0, 128)
	current := make([]string, 0, 16)
	inExtended := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !inExtended && isExtendedStart(line) {
			if len(current) > 0 {
				blocks = append(blocks, block{Lines: append([]string{}, current...)})
				current = current[:0]
			}
			inExtended = true
			current = append(current, line)
			continue
		}
		if inExtended {
			current = append(current, line)
			if line == "" && i+1 < len(lines) && isExtendedTerminator(lines[i+1]) {
				blocks = append(blocks, block{Lines: append([]string{}, current...)})
				current = current[:0]
				inExtended = false
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			if len(current) > 0 {
				blocks = append(blocks, block{Lines: append([]string{}, current...)})
				current = current[:0]
			}
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		blocks = append(blocks, block{Lines: append([]string{}, current...)})
	}
	return blocks
}

func isExtendedStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "bc..") || strings.HasPrefix(trimmed, "pre..") || strings.HasPrefix(trimmed, "notextile..")
}

func isBlockStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if isListStart(trimmed, '*') || isListStart(trimmed, '#') || strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "- ") {
		return true
	}
	if strings.HasPrefix(trimmed, "###.") {
		return true
	}
	if strings.HasPrefix(trimmed, "bq") || strings.HasPrefix(trimmed, "bc") || strings.HasPrefix(trimmed, "pre") || strings.HasPrefix(trimmed, "p") || strings.HasPrefix(trimmed, "h") {
		return true
	}
	if strings.HasPrefix(trimmed, "fn") || strings.HasPrefix(trimmed, "note") || strings.HasPrefix(trimmed, "notelist") {
		return true
	}
	return false
}

func isExtendedTerminator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if isListStart(trimmed, '*') || isListStart(trimmed, '#') {
		return false
	}
	return isBlockStart(line)
}

func parseBlock(blk block, options Options) (*document.D, error) {
	trimmed := strings.TrimSpace(blk.Lines[0])
	if trimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "###.") {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "<!--") {
		comment := strings.Join(blk.Lines, "\n")
		if strings.HasSuffix(strings.TrimSpace(blk.Lines[len(blk.Lines)-1]), "-->") {
			if options.Restricted {
				escaped, _ := escapeAndGlyphWithPrev(comment, 0)
				escaped = wrapCaps(escaped)
				escaped = strings.ReplaceAll(escaped, "\n", "<br />\n")
				p := document.New("p")
				p.AddChild(document.Text(escaped, true))
				return p, nil
			}
			return document.Text(comment, true), nil
		}
	}
	if htmlTag, inner, attrs, ok := parseHtmlWrapper(trimmed, blk.Lines); ok {
		if options.Restricted {
			return parseInline(strings.Join(blk.Lines, "\n"), "p", nil, options)
		}
		if htmlTag != "" {
			return parseInline(inner, htmlTag, attrs, options)
		}
		if len(blk.Lines) > 1 {
			return document.Text(strings.Join(blk.Lines, "\n"), true), nil
		}
		return document.Text(trimmed, true), nil
	}
	if options.RawBlocks && !options.Restricted {
		if name := rawBlockTagName(blk.Lines[0]); name != "" {
			if len(blk.Lines) == 1 || blockContainsClosingTag(blk.Lines, name) {
				return document.Text(strings.Join(blk.Lines, "\n"), true), nil
			}
		}
	}
	if !options.Restricted {
		if _, content, ok := parseBlockWithTag(blk, "notextile", options); ok {
			return document.Text(content, true), nil
		}
	}
	if options.Lite {
		if strings.HasPrefix(trimmed, "p") {
			if attrs, content, ok := parseBlockWithTag(blk, "p", options); ok {
				return parseInline(content, "p", attrs, options)
			}
		}
		content := strings.Join(blk.Lines, "\n")
		if options.Restricted && strings.Contains(content, "<!--") {
			escaped, _ := escapeAndGlyphWithPrev(content, 0)
			escaped = wrapCaps(escaped)
			escaped = strings.ReplaceAll(escaped, "\n", "<br />\n")
			p := document.New("p")
			p.AddChild(document.Text(escaped, true))
			return p, nil
		}
		return parseInline(content, "p", nil, options)
	}
	if parseNoteDefinition(blk, options) {
		return nil, nil
	}
	if noteList, ok, err := parseNoteList(blk, options); ok {
		return noteList, err
	}
	if strings.HasPrefix(trimmed, "h") {
		if heading, attrs, content := parseHeading(blk, options); heading != "" {
			return parseInline(content, heading, attrs, options)
		}
	}
	if strings.HasPrefix(trimmed, "bq") {
		if attrs, content, ok := parseBlockWithTag(blk, "bq", options); ok {
			cite, content := extractBlockquoteCite(content)
			if cite != "" {
				if attrs == nil {
					attrs = map[string]string{}
				}
				attrs["cite"] = cite
			}
			blockquote := document.New("blockquote")
			blockquote.Attr = attrs
			p, err := parseInline(content, "p", nil, options)
			if err != nil {
				return nil, err
			}
			blockquote.AddChild(p)
			return blockquote, nil
		}
	}
	if strings.HasPrefix(trimmed, "fn") {
		if footnote, ok, err := parseFootnoteBlock(blk, options); ok {
			return footnote, err
		}
	}
	if strings.HasPrefix(trimmed, "bc") {
		if attrs, content, ok := parseBlockWithTag(blk, "bc", options); ok {
			content = strings.TrimSuffix(content, "\n")
			pre := document.New("pre+code")
			pre.Attr = attrs
			pre.AddChild(document.Text(escapeHTML(content), true))
			return pre, nil
		}
	}
	if strings.HasPrefix(trimmed, "pre") {
		if attrs, content, ok := parseBlockWithTag(blk, "pre", options); ok {
			content = strings.TrimSuffix(content, "\n")
			pre := document.New("pre")
			pre.Attr = attrs
			pre.AddChild(document.Text(renderPreContent(content), true))
			return pre, nil
		}
	}
	if isListStart(trimmed, '*') || isListStart(trimmed, '#') {
		return parseMixedList(blk.Lines, options)
	}
	if strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "table") {
		return parseTableBlock(blk.Lines, options)
	}
	if strings.HasPrefix(trimmed, "-") && blockHasDefinition(blk.Lines) {
		return parseDefinitionList(blk.Lines, options)
	}
	if strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, ":") {
		dl, err := parseClassicDefinitionList(blk.Lines, options)
		if err != nil {
			return nil, err
		}
		if dl != nil {
			return dl, nil
		}
	}
	if strings.HasPrefix(trimmed, "p") {
		if attrs, content, ok := parseBlockWithTag(blk, "p", options); ok {
			return parseInline(content, "p", attrs, options)
		}
	}
	if hasLeadingWhitespace(blk.Lines[0]) || (!options.Restricted && options.BlockTags && containsBlockTag(blk.Lines)) {
		return parseInlineLines(blk.Lines, "inline", nil, options)
	}
	return parseInline(strings.Join(blk.Lines, "\n"), "p", nil, options)
}

func parseHeading(blk block, options Options) (string, map[string]string, string) {
	line := strings.TrimSpace(blk.Lines[0])
	if len(line) < 3 || !strings.HasPrefix(line, "h") {
		return "", nil, ""
	}
	level := line[1]
	if level < '1' || level > '6' {
		return "", nil, ""
	}
	rest := line[2:]
	attrs, content, ok := parseTagLine(rest, options)
	if !ok {
		return "", nil, ""
	}
	content = appendBlockLines(content, blk.Lines[1:])
	return fmt.Sprintf("h%c", level), attrs, content
}

func parseBlockWithTag(blk block, tag string, options Options) (map[string]string, string, bool) {
	line := strings.TrimSpace(blk.Lines[0])
	if !strings.HasPrefix(line, tag) {
		return nil, "", false
	}
	rest := strings.TrimPrefix(line, tag)
	attrs, content, ok := parseTagLine(rest, options)
	if !ok {
		return nil, "", false
	}
	content = appendBlockLines(content, blk.Lines[1:])
	return attrs, content, true
}

func parseTagLine(rest string, options Options) (map[string]string, string, bool) {
	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
	leftPad, rightPad, rest := parsePadding(rest)
	align, rest := parseAlignment(rest)
	attrsPart := ""
	if strings.HasPrefix(rest, "(") || strings.HasPrefix(rest, "{") || strings.HasPrefix(rest, "[") {
		attrsPart, rest = extractAttributeFragment(rest)
	}
	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
	if !strings.HasPrefix(rest, ".") {
		return nil, "", false
	}
	rest = strings.TrimPrefix(rest, ".")
	if strings.HasPrefix(rest, ".") {
		rest = strings.TrimPrefix(rest, ".")
	}
	content := strings.TrimLeftFunc(rest, unicode.IsSpace)
	attrs := parseAttributes(attrsPart, options)
	if align != "" {
		if attrs == nil {
			attrs = map[string]string{}
		}
		if attrs["style"] != "" {
			attrs["style"] = attrs["style"] + "text-align:" + align + ";"
		} else {
			attrs["style"] = "text-align:" + align + ";"
		}
	}
	if leftPad > 0 || rightPad > 0 {
		if attrs == nil {
			attrs = map[string]string{}
		}
		paddingStyle := ""
		if leftPad > 0 {
			paddingStyle += "padding-left:" + strconv.Itoa(leftPad) + "em;"
		}
		if rightPad > 0 {
			paddingStyle += "padding-right:" + strconv.Itoa(rightPad) + "em;"
		}
		attrs["style"] = attrs["style"] + paddingStyle
	}
	return attrs, content, true
}

func extractAttributeFragment(rest string) (string, string) {
	start := 0
	end := 0
	for start < len(rest) {
		if rest[start] == '(' {
			end = strings.Index(rest[start:], ")")
			if end == -1 {
				return rest[:start], rest[start:]
			}
			end += start + 1
			start = end
			continue
		}
		if rest[start] == '{' {
			end = strings.Index(rest[start:], "}")
			if end == -1 {
				return rest[:start], rest[start:]
			}
			end += start + 1
			start = end
			continue
		}
		if rest[start] == '[' {
			end = strings.Index(rest[start:], "]")
			if end == -1 {
				return rest[:start], rest[start:]
			}
			end += start + 1
			start = end
			continue
		}
		break
	}
	return rest[:start], rest[start:]
}

func parseAlignment(rest string) (string, string) {
	if strings.HasPrefix(rest, "<>") {
		return "justify", strings.TrimPrefix(rest, "<>")
	}
	if strings.HasPrefix(rest, "<") {
		return "left", strings.TrimPrefix(rest, "<")
	}
	if strings.HasPrefix(rest, ">") {
		return "right", strings.TrimPrefix(rest, ">")
	}
	if strings.HasPrefix(rest, "=") {
		return "center", strings.TrimPrefix(rest, "=")
	}
	if strings.HasPrefix(rest, "&lt;&gt;") {
		return "justify", strings.TrimPrefix(rest, "&lt;&gt;")
	}
	if strings.HasPrefix(rest, "&lt;") {
		return "left", strings.TrimPrefix(rest, "&lt;")
	}
	if strings.HasPrefix(rest, "&gt;") {
		return "right", strings.TrimPrefix(rest, "&gt;")
	}
	return "", rest
}

func parsePadding(rest string) (int, int, string) {
	if len(rest) < 2 {
		return 0, 0, rest
	}
	if rest[0] != '(' && rest[0] != ')' {
		return 0, 0, rest
	}
	if rest[1] != '(' && rest[1] != ')' {
		return 0, 0, rest
	}
	idx := 0
	left := 0
	for idx < len(rest) && rest[idx] == '(' {
		left++
		idx++
	}
	right := 0
	for idx < len(rest) && rest[idx] == ')' {
		right++
		idx++
	}
	return left, right, rest[idx:]
}

func appendBlockLines(first string, lines []string) string {
	if len(lines) == 0 {
		return first
	}
	return strings.Join(append([]string{first}, lines...), "\n")
}

func extractBlockquoteCite(content string) (string, string) {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, ":") {
		parts := strings.SplitN(trimmed[1:], " ", 2)
		if len(parts) == 2 {
			return parts[0], strings.TrimSpace(parts[1])
		}
		return strings.TrimSpace(trimmed[1:]), ""
	}
	return "", content
}

func parseDefinitionList(lines []string, options Options) (*document.D, error) {
	dl := document.New("dl")
	type termEntry struct {
		text  string
		attrs map[string]string
	}
	var pendingTerms []termEntry
	var currentDef []string
	inDef := false
	flush := func() error {
		if len(pendingTerms) == 0 {
			currentDef = nil
			inDef = false
			return nil
		}
		defText := strings.TrimSpace(strings.Join(currentDef, "\n"))
		hasBlank := false
		for _, line := range currentDef {
			if strings.TrimSpace(line) == "" {
				hasBlank = true
				break
			}
		}
		for _, term := range pendingTerms {
			dt, err := parseInline(term.text, "dt", term.attrs, options)
			if err != nil {
				return err
			}
			dl.AddChild(dt)
		}
		var dd *document.D
		if hasBlank {
			p, err := parseInline(defText, "p", nil, options)
			if err != nil {
				return err
			}
			dd = document.New("dd")
			dd.AddChild(p)
		} else {
			ddParsed, err := parseInline(defText, "dd", nil, options)
			if err != nil {
				return err
			}
			dd = ddParsed
		}
		dl.AddChild(dd)
		pendingTerms = nil
		currentDef = nil
		inDef = false
		return nil
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inDef {
				currentDef = append(currentDef, "")
			}
			continue
		}
		startsDash := strings.HasPrefix(trimmed, "-")
		hasDef := strings.Contains(trimmed, ":=")
		if startsDash && hasDef {
			if inDef {
				if err := flush(); err != nil {
					return nil, err
				}
			}
			parts := strings.SplitN(trimmed, ":=", 2)
			termText := strings.TrimSpace(strings.TrimPrefix(parts[0], "-"))
			attrs, termText := parseInlineAttributes(termText, options)
			pendingTerms = append(pendingTerms, termEntry{text: termText, attrs: attrs})
			defText := strings.TrimSpace(parts[1])
			currentDef = []string{defText}
			inDef = true
			if strings.HasSuffix(defText, "=:") {
				currentDef = []string{strings.TrimSpace(strings.TrimSuffix(defText, "=:"))}
				if err := flush(); err != nil {
					return nil, err
				}
			}
			continue
		}
		if startsDash && !hasDef {
			if inDef {
				if err := flush(); err != nil {
					return nil, err
				}
			}
			termText := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			attrs, termText := parseInlineAttributes(termText, options)
			pendingTerms = append(pendingTerms, termEntry{text: termText, attrs: attrs})
			continue
		}
		if !startsDash && hasDef && len(pendingTerms) > 0 {
			if inDef {
				if err := flush(); err != nil {
					return nil, err
				}
			}
			parts := strings.SplitN(trimmed, ":=", 2)
			termText := strings.TrimSpace(parts[0])
			if termText != "" {
				last := &pendingTerms[len(pendingTerms)-1]
				if last.text != "" {
					last.text = strings.TrimSpace(last.text) + "\n" + termText
				} else {
					last.text = termText
				}
			}
			defText := strings.TrimSpace(parts[1])
			currentDef = []string{defText}
			inDef = true
			if strings.HasSuffix(defText, "=:") {
				currentDef = []string{strings.TrimSpace(strings.TrimSuffix(defText, "=:"))}
				if err := flush(); err != nil {
					return nil, err
				}
			}
			continue
		}
		if inDef {
			if strings.HasSuffix(trimmed, "=:") {
				currentDef = append(currentDef, strings.TrimSpace(strings.TrimSuffix(trimmed, "=:")))
				if err := flush(); err != nil {
					return nil, err
				}
				continue
			}
			currentDef = append(currentDef, trimmed)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return dl, nil
}


func isListStart(line string, marker rune) bool {
	markers, _, _, _, ok := parseListMarkers(line)
	if !ok || markers == "" {
		return false
	}
	return rune(markers[0]) == marker
}

func parseTable(lines []string, options Options) (*document.D, error) {
	table := document.New("table")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		row := document.New("tr")
		cells := splitTableRow(line)
		for _, cell := range cells {
			td, err := parseInline(cell, "td", nil, options)
			if err != nil {
				return table, err
			}
			row.AddChild(td)
		}
		table.AddChild(row)
	}
	return table, nil
}

func splitTableRow(line string) []string {
	trimmed := strings.TrimPrefix(line, "|")
	trimmed = strings.TrimRight(trimmed, " \t\r")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	return parts
}

func parseInline(text string, tag string, attrs map[string]string, options Options) (*document.D, error) {
	node := document.New(tag)
	node.Attr = attrs
	idx := 0
	prevRune := rune(0)
	for idx < len(text) {
		if strings.HasPrefix(text[idx:], "\n") {
			if options.LineWrap == 0 {
				if prevRune == 0 || !unicode.IsSpace(prevRune) {
					node.AddChild(document.Text(" ", true))
					prevRune = ' '
				}
				idx++
				continue
			}
			node.AddChild(&document.D{Tag: "br"})
			idx++
			continue
		}
		r, _ := utf8.DecodeRuneInString(text[idx:])
		if r == '<' {
			if comment, advance, ok := parseInlineComment(text, idx, options); ok {
				node.AddChild(comment)
				idx += advance
				continue
			}
			if advance, ok := parseRawBreak(text, idx); ok {
				node.AddChild(&document.D{Tag: "br"})
				if idx+advance < len(text) && text[idx+advance] == '\n' {
					advance++
				}
				idx += advance
				continue
			}
			if htmlNode, last, advance, ok := parseInlineHTMLTag(text, idx, options); ok {
				node.AddChild(htmlNode)
				prevRune = last
				idx += advance
				continue
			}
		}
		if r == '"' {
			if link, last, advance, ok := parseQuotedLink(text, idx, options); ok {
				node.AddChild(link)
				prevRune = last
				idx += advance
				continue
			}
		}
		if r == '!' {
			if img, advance, ok := parseBang(text, idx, options); ok {
				node.AddChild(img)
				prevRune = lastRune(text[idx:idx+advance], prevRune)
				idx += advance
				continue
			}
		}
		if r == 'x' || r == 'X' {
			prevNon := prevNonSpaceRune(text, idx, prevRune)
			nextNon := nextNonSpaceRune(text, idx+1)
			if isDimNumberChar(prevNon) && isDimNumberChar(nextNon) {
				node.AddChild(document.Text("&#215;", true))
				prevRune = '×'
				idx++
				continue
			}
		}
		if r == '[' {
			if frac, last, advance, ok := parseBracketFraction(text, idx); ok {
				node.AddChild(document.Text(frac, true))
				prevRune = last
				idx += advance
				continue
			}
			if children, advance, ok := parseBracketedPhrase(text, idx, options); ok {
				for _, child := range children {
					node.AddChild(child)
				}
				prevRune = lastRune(text[idx:idx+advance], prevRune)
				idx += advance
				continue
			}
			if note, advance, ok := parseNoteRef(text, idx); ok {
				node.AddChild(note)
				prevRune = lastRune(text[idx:idx+advance], prevRune)
				idx += advance
				continue
			}
			if foot, advance, ok := parseFootnoteRef(text, idx); ok {
				node.AddChild(foot)
				prevRune = lastRune(text[idx:idx+advance], prevRune)
				idx += advance
				continue
			}
		}
		if unicode.IsUpper(r) {
			if acro, advance, ok := parseAcronym(text, idx); ok {
				node.AddChild(acro)
				prevRune = lastRune(text[idx:idx+advance], prevRune)
				idx += advance
				continue
			}
		}
		if r == '@' {
			if !options.Lite {
				if code, advance, ok := parseDelimited(text, idx, "@", true); ok {
					node.AddChild(&document.D{Tag: "code", Children: []*document.D{document.Text(escapeHTML(code), true)}})
					idx += advance
					continue
				}
			}
		}
		if r == '*' {
			if child, last, advance, ok := parseDelimitedInline(text, idx, "**", "b", options); ok {
				node.AddChild(child)
				prevRune = last
				idx += advance
				continue
			}
			prev := rune(0)
			next := rune(0)
			if idx > 0 {
				prev, _ = utf8.DecodeLastRuneInString(text[:idx])
			}
			if idx+1 < len(text) {
				next, _ = utf8.DecodeRuneInString(text[idx+1:])
			}
			if !isAlphaNumeric(prev) || !isAlphaNumeric(next) {
				if child, last, advance, ok := parseDelimitedInline(text, idx, "*", "strong", options); ok {
					node.AddChild(child)
					prevRune = last
					idx += advance
					continue
				}
			}
		}
		if r == '_' {
			prev := rune(0)
			next := rune(0)
			nextDouble := rune(0)
			if idx > 0 {
				prev, _ = utf8.DecodeLastRuneInString(text[:idx])
			}
			if idx+1 < len(text) {
				next, _ = utf8.DecodeRuneInString(text[idx+1:])
			}
			if idx+2 < len(text) {
				nextDouble, _ = utf8.DecodeRuneInString(text[idx+2:])
			}
			if !isAlphaNumeric(prev) || !isAlphaNumeric(nextDouble) {
				if child, last, advance, ok := parseDelimitedInline(text, idx, "__", "i", options); ok {
					node.AddChild(child)
					prevRune = last
					idx += advance
					continue
				}
			}
			if next != '_' && prev != '_' {
				if !isAlphaNumeric(prev) || !isAlphaNumeric(next) {
					if child, last, advance, ok := parseDelimitedInline(text, idx, "_", "em", options); ok {
						node.AddChild(child)
						prevRune = last
						idx += advance
						continue
					}
				}
			}
		}
		if r == '^' {
			if child, last, advance, ok := parseDelimitedInline(text, idx, "^", "sup", options); ok {
				node.AddChild(child)
				prevRune = last
				idx += advance
				continue
			}
		}
		if r == '~' {
			if child, last, advance, ok := parseDelimitedInline(text, idx, "~", "sub", options); ok {
				node.AddChild(child)
				prevRune = last
				idx += advance
				continue
			}
		}
		if r == '+' {
			if child, last, advance, ok := parseDelimitedInline(text, idx, "+", "ins", options); ok {
				node.AddChild(child)
				prevRune = last
				idx += advance
				continue
			}
		}
		if r == '-' {
			if strings.HasPrefix(text[idx:], "--") {
				prevRune = addTextNodes(node, "--", prevRune)
				idx += 2
				continue
			}
			prev := rune(0)
			next := rune(0)
			if idx > 0 {
				prev, _ = utf8.DecodeLastRuneInString(text[:idx])
			}
			if idx+1 < len(text) {
				next, _ = utf8.DecodeRuneInString(text[idx+1:])
			}
			if prev == ' ' && next == ' ' {
				node.AddChild(document.Text("&#8211;", true))
				prevRune = '–'
				idx++
				continue
			}
			if !isAlphaNumeric(prev) && (isAlphaNumeric(next) || next == '(' || next == '[' || next == '{') {
				if child, last, advance, ok := parseDelimitedInline(text, idx, "-", "del", options); ok {
					node.AddChild(child)
					prevRune = last
					idx += advance
					continue
				}
			}
		}
		if r == '?' {
			if child, last, advance, ok := parseDelimitedInline(text, idx, "??", "cite", options); ok {
				node.AddChild(child)
				prevRune = last
				idx += advance
				continue
			}
		}
		if r == '%' {
			prev := rune(0)
			if idx > 0 {
				prev, _ = utf8.DecodeLastRuneInString(text[:idx])
			}
			if !unicode.IsDigit(prev) {
				if span, last, advance, ok := parseSpan(text, idx, options); ok {
					node.AddChild(span)
					prevRune = last
					idx += advance
					continue
				}
			}
		}
		if r == '=' {
			if strings.HasPrefix(text[idx:], "==") {
				if content, advance, ok := parseDelimited(text, idx, "==", false); ok {
					node.AddChild(document.Text(content, true))
					idx += advance
					continue
				}
			}
		}
		next := nextSpecialIndex(text, idx)
		if next == idx {
			r, size := utf8.DecodeRuneInString(text[idx:])
			if r == '"' || r == '\'' {
				nextRune := rune(0)
				if idx+size < len(text) {
					nextRune, _ = utf8.DecodeRuneInString(text[idx+size:])
				}
				handledEmpty := false
				if r == '"' && unicode.IsSpace(nextRune) {
					scan := idx + size
					for scan < len(text) {
						r2, size2 := utf8.DecodeRuneInString(text[scan:])
						if unicode.IsSpace(r2) {
							scan += size2
							continue
						}
						if r2 == '"' {
							node.AddChild(document.Text("&#8220;", true))
							node.AddChild(document.Text("&#8221;", true))
							prevRune = r
							idx = scan + size2
							handledEmpty = true
						}
						break
					}
				}
				if handledEmpty {
					continue
				}
				entity := ""
				if r == '\'' && (prevRune == 0 || unicode.IsSpace(prevRune)) && unicode.IsDigit(nextRune) {
					if hasClosingSingleQuote(text, idx+size) {
						entity = "&#8216;"
					} else {
						entity = "&#8217;"
					}
				} else {
					entity = quoteEntity(r, prevRune, nextRune)
				}
				node.AddChild(document.Text(entity, true))
				prevRune = r
				idx += size
				continue
			}
			prevRune = addTextNodes(node, string(r), prevRune)
			idx += size
			continue
		}
		segment := text[idx:next]
		prevRune = addTextNodes(node, segment, prevRune)
		idx = next
	}
	return node, nil
}

func parseDelimited(text string, idx int, delim string, literal bool) (string, int, bool) {
	if !strings.HasPrefix(text[idx:], delim) {
		return "", 0, false
	}
	start := idx + len(delim)
	scan := start
	braceDepth := 0
	for scan <= len(text)-len(delim) {
		if text[scan] == '{' {
			braceDepth++
			scan++
			continue
		}
		if text[scan] == '}' && braceDepth > 0 {
			braceDepth--
			scan++
			continue
		}
		if braceDepth == 0 && strings.HasPrefix(text[scan:], delim) {
			content := text[start:scan]
			return content, len(delim)*2 + scan - start, true
		}
		scan++
	}
	return "", 0, false
}

func parseDelimitedInline(text string, idx int, delim string, tag string, options Options) (*document.D, rune, int, bool) {
	content, advance, ok := parseDelimited(text, idx, delim, false)
	if !ok {
		return nil, 0, 0, false
	}
	if strings.TrimSpace(content) == "" {
		return nil, 0, 0, false
	}
	if tag == "del" && strings.Contains(content, "\n") {
		return nil, 0, 0, false
	}
	attrs, inner := parseInlineAttributes(content, options)
	if attrs != nil {
		if tag == "del" || tag == "ins" {
			delete(attrs, "lang")
			if attrs["id"] != "" && strings.Contains(attrs["class"], " ") {
				attrs["id"] = ""
				parts := strings.Fields(attrs["class"])
				if len(parts) > 0 {
					attrs["class"] = parts[0]
				}
			}
			if attrs["id"] == "" {
				delete(attrs, "id")
			}
		}
	}
	child, err := parseInline(inner, tag, attrs, options)
	if err != nil {
		return nil, 0, 0, false
	}
	return child, lastRune(inner, 0), advance, true
}

func parseSpan(text string, idx int, options Options) (*document.D, rune, int, bool) {
	content, advance, ok := parseDelimited(text, idx, "%", false)
	if !ok {
		return nil, 0, 0, false
	}
	if strings.TrimSpace(content) == "" {
		return nil, 0, 0, false
	}
	attrs, inner := parseInlineAttributes(content, options)
	child, err := parseInline(inner, "span", attrs, options)
	if err != nil {
		return nil, 0, 0, false
	}
	return child, lastRune(inner, 0), advance, true
}

func parseInlineAttributes(text string, options Options) (map[string]string, string) {
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "[") {
		if end := strings.Index(trimmed, "]"); end != -1 {
			inner := trimmed[1:end]
			if isBracketedPhrase(inner) {
				return nil, trimmed
			}
		}
	}
	if strings.HasPrefix(trimmed, "(") || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		attrsPart, rest := extractAttributeFragment(trimmed)
		attrs := parseAttributes(attrsPart, options)
		return attrs, strings.TrimSpace(rest)
	}
	return nil, trimmed
}

func parseBracketFraction(text string, idx int) (string, rune, int, bool) {
	if strings.HasPrefix(text[idx:], "[1/2]") {
		return "&#189;", '½', 5, true
	}
	if strings.HasPrefix(text[idx:], "[1/4]") {
		return "&#188;", '¼', 5, true
	}
	if strings.HasPrefix(text[idx:], "[3/4]") {
		return "&#190;", '¾', 5, true
	}
	return "", 0, 0, false
}

func parseBracketedPhrase(text string, idx int, options Options) ([]*document.D, int, bool) {
	depth := 1
	scan := idx + 1
	end := -1
	for scan < len(text) {
		r, size := utf8.DecodeRuneInString(text[scan:])
		if r == '[' {
			depth++
		} else if r == ']' {
			depth--
			if depth == 0 {
				end = scan
				break
			}
		}
		scan += size
	}
	if end == -1 {
		return nil, 0, false
	}
	inner := text[idx+1 : end]
	if !isBracketedPhrase(inner) {
		return nil, 0, false
	}
	parsed, err := parseInline(inner, "", nil, options)
	if err != nil {
		return nil, 0, false
	}
	return parsed.Children, end - idx + 1, true
}

func isBracketedPhrase(inner string) bool {
	pairs := []struct {
		start string
		end   string
	}{
		{"**", "**"},
		{"__", "__"},
		{"??", "??"},
		{"==", "=="},
		{"*", "*"},
		{"_", "_"},
		{"^", "^"},
		{"~", "~"},
		{"+", "+"},
		{"-", "-"},
		{"@", "@"},
		{"%", "%"},
		{"!", "!"},
	}
	for _, pair := range pairs {
		if strings.HasPrefix(inner, pair.start) && strings.HasSuffix(inner, pair.end) {
			return true
		}
	}
	if strings.HasPrefix(inner, "\"") && strings.Contains(inner, "\":") {
		return true
	}
	return false
}

func parseFootnoteRef(text string, idx int) (*document.D, int, bool) {
	if !strings.HasPrefix(text[idx:], "[") {
		return nil, 0, false
	}
	end := strings.Index(text[idx+1:], "]")
	if end == -1 {
		return nil, 0, false
	}
	end += idx + 1
	payload := text[idx+1 : end]
	if payload == "" {
		return nil, 0, false
	}
	noLink := strings.HasSuffix(payload, "!")
	if noLink {
		payload = strings.TrimSuffix(payload, "!")
	}
	if !isUnicodeDigits(payload) {
		return nil, 0, false
	}
	sup := document.New("sup")
	attrs := map[string]string{"class": "footnote"}
	if footnoteRefs != nil {
		footnoteRefs[payload]++
		if footnoteRefs[payload] == 1 {
			attrs["id"] = "fnrev"
		}
	}
	sup.Attr = attrs
	escaped, _ := escapeAndGlyphWithPrev(payload, 0)
	if noLink {
		sup.AddChild(document.Text(escaped, true))
		return sup, end - idx + 1, true
	}
	link := document.New("a")
	link.Attr = map[string]string{"href": "#fn"}
	link.AddChild(document.Text(escaped, true))
	sup.AddChild(link)
	return sup, end - idx + 1, true
}

func parseNoteRef(text string, idx int) (*document.D, int, bool) {
	if !strings.HasPrefix(text[idx:], "[#") {
		return nil, 0, false
	}
	end := strings.Index(text[idx+2:], "]")
	if end == -1 {
		return nil, 0, false
	}
	end += idx + 2
	payload := strings.TrimSpace(text[idx+2 : end])
	if payload == "" {
		return nil, 0, false
	}
	noLink := strings.HasSuffix(payload, "!")
	if noLink {
		payload = strings.TrimSuffix(payload, "!")
	}
	label := payload
	if label == "" {
		return nil, 0, false
	}
	index, ok := noteIndex[label]
	if !ok {
		index = len(noteRefOrder) + 1
		noteIndex[label] = index
		noteRefOrder = append(noteRefOrder, label)
	}
	def, ok := noteDefs[label]
	if !ok {
		def = &noteDef{label: label}
		noteDefs[label] = def
	}
	def.referenced = true
	def.index = index
	def.refCount++
	sup := document.New("sup")
	span := document.New("span")
	span.Attr = map[string]string{"id": "noteref"}
	span.AddChild(document.Text(strconv.Itoa(index), true))
	var link *document.D
	if noLink {
		sup.AddChild(span)
	} else {
		link = document.New("a")
		link.Attr = map[string]string{"href": "#note"}
		link.AddChild(span)
		sup.AddChild(link)
	}
	noteRefNodes[label] = append(noteRefNodes[label], noteRefNode{sup: sup, link: link, span: span, noLink: noLink})
	return sup, end - idx + 1, true
}

func parseFootnoteBlock(blk block, options Options) (*document.D, bool, error) {
	line := strings.TrimSpace(blk.Lines[0])
	if !strings.HasPrefix(line, "fn") {
		return nil, false, nil
	}
	idx := 2
	numStart := idx
	for idx < len(line) {
		r, size := utf8.DecodeRuneInString(line[idx:])
		if !unicode.IsDigit(r) {
			break
		}
		idx += size
	}
	if idx == numStart {
		return nil, false, nil
	}
	number := line[numStart:idx]
	modifier := byte(0)
	if idx < len(line) && (line[idx] == '^' || line[idx] == '!') {
		modifier = line[idx]
		idx++
	}
	rest := line[idx:]
	attrs, content, ok := parseTagLine(rest, options)
	if !ok {
		return nil, false, nil
	}
	customAttrs := attrs != nil && (attrs["class"] != "" || attrs["id"] != "")
	if attrs == nil {
		attrs = map[string]string{}
	}
	if attrs["class"] == "" {
		attrs["class"] = "footnote"
	}
	if attrs["id"] == "" {
		attrs["id"] = "fn"
	}
	content = appendBlockLines(content, blk.Lines[1:])
	p, err := parseInline(content, "p", attrs, options)
	if err != nil {
		return nil, true, err
	}
	sup := document.New("sup")
	if customAttrs {
		sup.Attr = map[string]string{"id": "fn"}
	}
	escaped, _ := escapeAndGlyphWithPrev(number, 0)
	if modifier == '^' {
		link := document.New("a")
		link.Attr = map[string]string{"href": "#fnrev"}
		link.AddChild(document.Text(escaped, true))
		sup.AddChild(link)
	} else {
		sup.AddChild(document.Text(escaped, true))
	}
	space := document.Text(" ", true)
	p.Children = append([]*document.D{sup, space}, p.Children...)
	return p, true, nil
}

func parseRawBreak(text string, idx int) (int, bool) {
	if idx+3 > len(text) {
		return 0, false
	}
	lower := strings.ToLower(text[idx:])
	if !strings.HasPrefix(lower, "<br") {
		return 0, false
	}
	end := strings.Index(lower, ">")
	if end == -1 {
		return 0, false
	}
	return end + 1, true
}

func parseInlineComment(text string, idx int, options Options) (*document.D, int, bool) {
	if !strings.HasPrefix(text[idx:], "<!--") {
		return nil, 0, false
	}
	end := strings.Index(text[idx+4:], "-->")
	if end == -1 {
		return nil, 0, false
	}
	end += idx + 4
	comment := text[idx : end+3]
	if options.Restricted {
		escaped, _ := escapeAndGlyphWithPrev(comment, 0)
		escaped = wrapCaps(escaped)
		escaped = strings.ReplaceAll(escaped, "\n", "<br />\n")
		return document.Text(escaped, true), len(comment), true
	}
	return document.Text(comment, true), len(comment), true
}

func parseInlineHTMLTag(text string, idx int, options Options) (*document.D, rune, int, bool) {
	if idx+1 >= len(text) {
		return nil, 0, 0, false
	}
	next := rune(text[idx+1])
	if next != '/' && !unicode.IsLetter(next) {
		return nil, 0, 0, false
	}
	end := strings.Index(text[idx:], ">")
	if end == -1 {
		return nil, 0, 0, false
	}
	end += idx
	tag := text[idx : end+1]
	if options.Restricted {
		escaped, _ := escapeAndGlyphWithPrev(tag, 0)
		return document.Text(escaped, true), '>', end - idx + 1, true
	}
	return document.Text(tag, true), '>', end - idx + 1, true
}

func parseAcronym(text string, idx int) (*document.D, int, bool) {
	start := idx
	for idx < len(text) {
		r := rune(text[idx])
		if r < 'A' || r > 'Z' {
			break
		}
		idx++
	}
	if idx-start < 2 {
		return nil, 0, false
	}
	if idx >= len(text) || text[idx] != '(' {
		return nil, 0, false
	}
	end := strings.Index(text[idx+1:], ")")
	if end == -1 {
		return nil, 0, false
	}
	end += idx + 1
	acronymText := text[start:idx]
	titleText := text[idx+1 : end]
	acro := document.New("acronym")
	acro.Attr = map[string]string{"title": glyphPlain(titleText)}
	span := document.New("span")
	span.Attr = map[string]string{"class": "caps"}
	escapedAcronym, _ := escapeAndGlyphWithPrev(acronymText, 0)
	span.AddChild(document.Text(escapedAcronym, true))
	acro.AddChild(span)
	return acro, end - start + 1, true
}

func parseQuotedLink(text string, idx int, options Options) (*document.D, rune, int, bool) {
	if idx > 0 {
		prev, _ := utf8.DecodeLastRuneInString(text[:idx])
		if isAlphaNumeric(prev) {
			return nil, 0, 0, false
		}
	}
	endQuote := -1
	for scan := idx + 1; scan < len(text); {
		if text[scan] != '"' {
			scan++
			continue
		}
		if scan+1 < len(text) && text[scan+1] == '"' {
			if scan+2 < len(text) && text[scan+2] == ':' {
				rawText := text[idx+1 : scan+1]
				trimmed := strings.TrimSpace(rawText)
				if trimmed != "" && strings.HasPrefix(trimmed, "\"") {
					endQuote = scan + 1
					break
				}
			}
			scan += 2
			continue
		}
		if scan+1 < len(text) && text[scan+1] == ':' {
			rawText := text[idx+1 : scan]
			if strings.TrimSpace(rawText) != "" {
				endQuote = scan
				break
			}
		}
		scan++
	}
	if endQuote == -1 {
		return nil, 0, 0, false
	}
	if endQuote+1 >= len(text) || text[endQuote+1] != ':' {
		return nil, 0, 0, false
	}
	urlStart := endQuote + 2
	urlEnd := urlStart
	for urlEnd < len(text) {
		r, size := utf8.DecodeRuneInString(text[urlEnd:])
		if unicode.IsSpace(r) || r == '<' {
			break
		}
		urlEnd += size
	}
	rawURL := text[urlStart:urlEnd]
	trimmedURL, trimCount := trimLinkPunctuation(rawURL)
	if strings.TrimSpace(trimmedURL) == "" {
		return nil, 0, 0, false
	}
	urlEnd = urlStart + len(rawURL) - trimCount
	rawLinkText := text[idx+1 : endQuote]
	trimmedLeft := strings.TrimLeft(rawLinkText, " \t")
	if trimmedLeft == "" {
		return nil, 0, 0, false
	}
	if rawLinkText != trimmedLeft && !strings.HasPrefix(trimmedLeft, "(") {
		return nil, 0, 0, false
	}
	linkText := strings.TrimSpace(rawLinkText)
	if linkText == "" {
		return nil, 0, 0, false
	}
	if idx := strings.Index(rawLinkText, "\"("); idx != -1 {
		if end := strings.Index(rawLinkText[idx+2:], ")"); end != -1 {
			fragment := rawLinkText[idx+2 : idx+2+end]
			if strings.ContainsAny(fragment, "#.") {
				return nil, 0, 0, false
			}
		}
	}
	linkText = normalizeLinkQuotes(linkText)
	if strings.HasSuffix(linkText, "\"") && !strings.HasPrefix(linkText, "\"") {
		return nil, 0, 0, false
	}
	linkAttrs, linkText := parseInlineAttrSequence(linkText, options)
	linkText, title := extractLinkTitle(linkText)
	linkText = strings.TrimSpace(linkText)
	if linkText == "" {
		return nil, 0, 0, false
	}
	if strings.HasPrefix(linkText, "$(") && strings.HasSuffix(linkText, ")") {
		title = linkText[2 : len(linkText)-1]
		linkText = "$"
	}
	url := trimmedURL
	refExpanded := false
	if linkRefs != nil {
		if ref, ok := linkRefs[url]; ok {
			url = ref
			refExpanded = true
		}
	}
	if options.Restricted {
		if !isAllowedSchemeRestricted(url) {
			return nil, 0, 0, false
		}
	} else if !isAllowedScheme(url) {
		return nil, 0, 0, false
	}
	displaySource := url
	url = sanitizeURL(url, options)
	link := document.New("a")
	attrs := map[string]string{"href": url}
	for k, v := range linkAttrs {
		attrs[k] = v
	}
	if title != "" {
		attrs["title"] = title
	}
	if options.LinkRelationship != "" {
		attrs["rel"] = options.LinkRelationship
	}
	link.Attr = attrs
	if linkText == "$" {
		if refExpanded {
			linkText = displaySource
		} else {
			linkText = displayURL(displaySource)
		}
	}
	child, err := parseInline(linkText, "", nil, options)
	if err == nil {
		link.Children = child.Children
	}
	return link, lastRune(linkText, 0), urlEnd - idx, true
}

func normalizeLinkQuotes(text string) string {
	if text == "" {
		return text
	}
	text = strings.ReplaceAll(text, "\"\"\"", "&#8220;&quot;&#8221;")
	if strings.HasPrefix(text, "\"\"") {
		text = "\"" + strings.TrimPrefix(text, "\"\"")
	}
	if strings.HasSuffix(text, "\"\"") {
		text = strings.TrimSuffix(text, "\"\"") + "\""
	}
	text = strings.ReplaceAll(text, "\"\"", "&#8220;&#8221;")
	return text
}

func parseBang(text string, idx int, options Options) (*document.D, int, bool) {
	if idx > 0 {
		prev, _ := utf8.DecodeLastRuneInString(text[:idx])
		if isAlphaNumeric(prev) {
			return nil, 0, false
		}
	}
	end := strings.Index(text[idx+1:], "!")
	if end == -1 {
		return nil, 0, false
	}
	end += idx + 1
	payload := text[idx+1 : end]
	if strings.TrimSpace(payload) == "" {
		return nil, 0, false
	}
	linkTarget := ""
	advance := end - idx + 1
	if end+1 < len(text) && text[end+1] == ':' {
		linkStart := end + 2
		linkEnd := linkStart
		for linkEnd < len(text) {
			r, size := utf8.DecodeRuneInString(text[linkEnd:])
			if unicode.IsSpace(r) {
				break
			}
			linkEnd += size
		}
		linkTarget = sanitizeURL(text[linkStart:linkEnd], options)
		advance = linkEnd - idx
	}
	if !options.Images {
		return nil, 0, false
	}
	align := ""
	if strings.HasPrefix(payload, "<") {
		align = "left"
		payload = strings.TrimPrefix(payload, "<")
	} else if strings.HasPrefix(payload, ">") {
		align = "right"
		payload = strings.TrimPrefix(payload, ">")
	}
	imgAttrs, payload := parseInlineAttrSequence(payload, options)
	url := payload
	alt := ""
	if open := strings.LastIndex(payload, "("); open != -1 && strings.HasSuffix(payload, ")") {
		alt = payload[open+1 : len(payload)-1]
		url = payload[:open]
	}
	url = sanitizeImageURL(url, options)
	img := document.New("img")
	attrs := map[string]string{"src": url, "alt": ""}
	if align != "" {
		if options.HTML5 {
			addClass(attrs, "align-"+align)
		} else {
			attrs["align"] = align
		}
	}
	for k, v := range imgAttrs {
		attrs[k] = v
	}
	if alt != "" {
		attrs["alt"] = alt
		attrs["title"] = alt
	}
	if !options.DimensionlessImages && attrs["width"] == "" && attrs["height"] == "" {
		if width, height := extractImageDimensions(url); width != "" {
			attrs["width"] = width
			attrs["height"] = height
		}
	}
	img.Attr = attrs
	if linkTarget == "" {
		return img, advance, true
	}
	link := document.New("a")
	link.Attr = map[string]string{"href": linkTarget}
	link.AddChild(img)
	return link, advance, true
}

func parseAttributes(fragment string, options Options) map[string]string {
	trimmed := strings.TrimSpace(fragment)
	if trimmed == "" {
		return nil
	}
	attrs := map[string]string{}
	rest := strings.TrimLeft(trimmed, " \t")
	for len(rest) > 0 {
		switch rest[0] {
		case '(':
			end := strings.Index(rest, ")")
			if end == -1 {
				rest = rest[1:]
				continue
			}
			classID := rest[1:end]
			setClassID(attrs, classID, options)
			rest = strings.TrimLeft(rest[end+1:], " \t")
			continue
		case '{':
			end := strings.Index(rest, "}")
			if end == -1 {
				rest = rest[1:]
				continue
			}
			style := strings.TrimSpace(rest[1:end])
			if style != "" && !options.Restricted {
				decoded, ok := decodePercentEncoding(style)
				if ok {
					if hasPercentEncoding(decoded) {
						style = ""
					} else {
						style = decoded
					}
				} else if strings.Contains(style, "%") {
					style = ""
				}
				if style != "" {
					attrs["style"] = normalizeStyle(style)
				}
			}
			rest = strings.TrimLeft(rest[end+1:], " \t")
			continue
		case '[':
			end := strings.Index(rest, "]")
			if end == -1 {
				rest = rest[1:]
				continue
			}
			lang := strings.TrimSpace(rest[1:end])
			if lang != "" && isValidLang(lang) {
				attrs["lang"] = lang
			}
			rest = strings.TrimLeft(rest[end+1:], " \t")
			continue
		default:
			rest = rest[1:]
		}
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func normalizeStyle(style string) string {
	parts := strings.Split(style, ";")
	type entry struct {
		key   string
		value string
	}
	entries := make([]entry, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pieces := strings.SplitN(part, ":", 2)
		key := strings.TrimSpace(pieces[0])
		value := ""
		if len(pieces) > 1 {
			value = strings.TrimRight(pieces[1], " ")
		}
		entries = append(entries, entry{key: key, value: value})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})
	var builder strings.Builder
	for _, item := range entries {
		if item.key == "" {
			continue
		}
		builder.WriteString(item.key)
		if item.value != "" {
			builder.WriteString(":")
			builder.WriteString(item.value)
		}
		builder.WriteString(";")
	}
	return builder.String()
}

func setClassID(attrs map[string]string, classID string, options Options) {
	if options.Restricted {
		return
	}
	classParts := []string{}
	var id string
	for _, part := range strings.Fields(classID) {
		if strings.Contains(part, "#") {
			pieces := strings.SplitN(part, "#", 2)
			if pieces[0] != "" {
				classParts = append(classParts, pieces[0])
			}
			if pieces[1] != "" {
				id = pieces[1]
			}
		} else if strings.HasPrefix(part, "#") {
			id = strings.TrimPrefix(part, "#")
		} else {
			classParts = append(classParts, part)
		}
	}
	if len(classParts) > 0 {
		attrs["class"] = strings.Join(classParts, " ")
	}
	if id != "" {
		attrs["id"] = id
	}
}

func nextSpecialIndex(text string, idx int) int {
	specials := "\n\"!@*_~^?%[<xX-+="
	parenDepth := 0
	for idx < len(text) {
		r, size := utf8.DecodeRuneInString(text[idx:])
		if r == '(' {
			parenDepth++
		} else if r == ')' && parenDepth > 0 {
			parenDepth--
		}
		if strings.ContainsRune(specials, r) {
			if r == '!' && parenDepth > 0 {
				idx += size
				continue
			}
			if r == '-' {
				if idx >= 3 && idx+1 < len(text) && text[idx-3:idx+2] == "(+/-)" {
					idx += size
					continue
				}
			}
			if r == '+' {
				if idx >= 1 && idx+3 < len(text) && text[idx-1:idx+4] == "(+/-)" {
					idx += size
					continue
				}
			}
			if r == 'x' || r == 'X' {
				prevNon := prevNonSpaceRune(text, idx, 0)
				nextNon := nextNonSpaceRune(text, idx+size)
				if !isDimNumberChar(prevNon) || !isDimNumberChar(nextNon) {
					idx += size
					continue
				}
			}
			return idx
		}
		idx += size
	}
	return len(text)
}

type capsSegment struct {
	text  string
	caps  bool
}

func findAcronymIndex(text string) int {
	for i := 0; i < len(text); i++ {
		if text[i] < 'A' || text[i] > 'Z' {
			continue
		}
		j := i
		for j < len(text) && text[j] >= 'A' && text[j] <= 'Z' {
			j++
		}
		if j-i >= 2 && j < len(text) && text[j] == '(' {
			return i
		}
		i = j
	}
	return -1
}

func addTextNodes(node *document.D, text string, prev rune) rune {
	if text == "" {
		return prev
	}
	for len(text) > 0 {
		idx := findAcronymIndex(text)
		if idx == -1 {
			return addPlainTextNodes(node, text, prev)
		}
		prefix := text[:idx]
		prev = addPlainTextNodes(node, prefix, prev)
		acro, advance, ok := parseAcronym(text, idx)
		if !ok {
			r, size := utf8.DecodeRuneInString(text[idx:])
			prev = addPlainTextNodes(node, string(r), prev)
			text = text[idx+size:]
			continue
		}
		node.AddChild(acro)
		prev = lastRune(text[idx:idx+advance], prev)
		text = text[idx+advance:]
	}
	return prev
}

func addPlainTextNodes(node *document.D, text string, prev rune) rune {
	if text == "" {
		return prev
	}
	segments := splitCapsSegments(text)
	for _, segment := range segments {
		if segment.text == "" {
			continue
		}
		if segment.caps {
			span := document.New("span")
			span.Attr = map[string]string{"class": "caps"}
			escaped, last := escapeAndGlyphWithPrev(segment.text, prev)
			span.AddChild(document.Text(escaped, true))
			node.AddChild(span)
			prev = last
			continue
		}
		escaped, last := escapeAndGlyphWithPrev(segment.text, prev)
		node.AddChild(document.Text(escaped, true))
		prev = last
	}
	return prev
}

func escapeAndGlyphWithCaps(text string, prev rune) (string, rune) {
	if text == "" {
		return "", prev
	}
	segments := splitCapsSegments(text)
	var builder strings.Builder
	last := prev
	for _, segment := range segments {
		if segment.text == "" {
			continue
		}
		escaped, next := escapeAndGlyphWithPrev(segment.text, last)
		if segment.caps {
			builder.WriteString("<span class=\"caps\">")
			builder.WriteString(escaped)
			builder.WriteString("</span>")
		} else {
			builder.WriteString(escaped)
		}
		last = next
	}
	return builder.String(), last
}

func wrapCaps(text string) string {
	if text == "" {
		return ""
	}
	segments := splitCapsSegments(text)
	var builder strings.Builder
	for _, segment := range segments {
		if segment.text == "" {
			continue
		}
		if segment.caps {
			builder.WriteString("<span class=\"caps\">")
			builder.WriteString(segment.text)
			builder.WriteString("</span>")
			continue
		}
		builder.WriteString(segment.text)
	}
	return builder.String()
}

func splitCapsSegments(text string) []capsSegment {
	segments := make([]capsSegment, 0, 8)
	var buf strings.Builder
	idx := 0
	for idx < len(text) {
		r, size := utf8.DecodeRuneInString(text[idx:])
		if unicode.IsUpper(r) {
			start := idx
			count := 0
			for idx < len(text) {
				r2, size2 := utf8.DecodeRuneInString(text[idx:])
				if !unicode.IsUpper(r2) {
					break
				}
				count++
				idx += size2
			}
			if count >= 3 {
				prev := rune(0)
				if start > 0 {
					prev, _ = utf8.DecodeLastRuneInString(text[:start])
				}
				next := rune(0)
				if idx < len(text) {
					next, _ = utf8.DecodeRuneInString(text[idx:])
				}
				if (unicode.IsLetter(prev) && unicode.IsLower(prev)) || (unicode.IsLetter(next) && unicode.IsLower(next)) {
					buf.WriteString(text[start:idx])
					continue
				}
				if buf.Len() > 0 {
					segments = append(segments, capsSegment{text: buf.String()})
					buf.Reset()
				}
				segments = append(segments, capsSegment{text: text[start:idx], caps: true})
				continue
			}
			idx = start
		}
		buf.WriteRune(r)
		idx += size
	}
	if buf.Len() > 0 {
		segments = append(segments, capsSegment{text: buf.String()})
	}
	return segments
}

func escapeAndGlyphWithPrev(text string, prev rune) (string, rune) {
	var builder strings.Builder
	idx := 0
	lastRune := prev
	lastNonSpace := prev
	for idx < len(text) {
		slice := text[idx:]
		switch {
		case strings.HasPrefix(slice, "(c)"):
			builder.WriteString("&#169;")
			idx += 3
			lastRune = '©'
			continue
		case strings.HasPrefix(slice, "(r)"):
			builder.WriteString("&#174;")
			idx += 3
			lastRune = '®'
			continue
		case strings.HasPrefix(slice, "(tm)"):
			builder.WriteString("&#8482;")
			idx += 4
			lastRune = '™'
			continue
		case strings.HasPrefix(slice, "(o)"):
			builder.WriteString("&#176;")
			idx += 3
			lastRune = '°'
			continue
		case strings.HasPrefix(slice, "(1/4)"):
			builder.WriteString("&#188;")
			idx += 5
			lastRune = '¼'
			continue
		case strings.HasPrefix(slice, "(1/2)"):
			builder.WriteString("&#189;")
			idx += 5
			lastRune = '½'
			continue
		case strings.HasPrefix(slice, "(3/4)"):
			builder.WriteString("&#190;")
			idx += 5
			lastRune = '¾'
			continue
		case strings.HasPrefix(slice, "[1/4]"):
			builder.WriteString("&#188;")
			idx += 5
			lastRune = '¼'
			continue
		case strings.HasPrefix(slice, "[1/2]"):
			builder.WriteString("&#189;")
			idx += 5
			lastRune = '½'
			continue
		case strings.HasPrefix(slice, "[3/4]"):
			builder.WriteString("&#190;")
			idx += 5
			lastRune = '¾'
			continue
		case strings.HasPrefix(slice, "(+/-)"):
			builder.WriteString("&#177;")
			idx += 5
			lastRune = '±'
			continue
		case strings.HasPrefix(slice, "..."):
			builder.WriteString("&#8230;")
			idx += 3
			lastRune = '…'
			continue
		case strings.HasPrefix(slice, "--"):
			builder.WriteString("&#8212;")
			idx += 2
			lastRune = '—'
			continue
		}
		r, size := utf8.DecodeRuneInString(slice)
		if r == '-' {
			if idx > 0 && idx+1 < len(text) && text[idx-1] == ' ' && text[idx+1] == ' ' {
				builder.WriteString("&#8211;")
				idx += size
				lastRune = '–'
				continue
			}
		}
		if r == 'x' || r == 'X' {
			prevNon := prevNonSpaceRune(text, idx, lastNonSpace)
			nextNon := nextNonSpaceRune(text, idx+size)
			if isDimNumberChar(prevNon) && isDimNumberChar(nextNon) {
				builder.WriteString("&#215;")
				idx += size
				lastRune = '×'
				lastNonSpace = '×'
				continue
			}
		}
		if r == '"' {
			next := rune(0)
			if idx+size < len(text) {
				next, _ = utf8.DecodeRuneInString(text[idx+size:])
			}
			if lastRune == '\'' && next == '\'' {
				builder.WriteString("&quot;")
				idx += size
				lastRune = r
				continue
			}
			if isLiteralQuoteContext(lastRune, next) {
				builder.WriteString("&quot;")
				idx += size
				lastRune = r
				continue
			}
			if isOpeningQuote(lastRune, next) {
				builder.WriteString("&#8220;")
			} else {
				builder.WriteString("&#8221;")
			}
			idx += size
			lastRune = r
			continue
		}
		if r == '\'' {
			next := rune(0)
			if idx+size < len(text) {
				next, _ = utf8.DecodeRuneInString(text[idx+size:])
			}
			if isAlphaNumeric(lastRune) && isAlphaNumeric(next) {
				builder.WriteString("&#8217;")
				idx += size
				lastRune = '\''
				continue
			}
			if (lastRune == 0 || unicode.IsSpace(lastRune)) && unicode.IsDigit(next) {
				builder.WriteString("&#8217;")
				idx += size
				lastRune = '\''
				continue
			}
			if isOpeningQuote(lastRune, next) {
				builder.WriteString("&#8216;")
			} else {
				builder.WriteString("&#8217;")
			}
			idx += size
			lastRune = '\''
			continue
		}
		switch r {
		case '&':
			if !restrictedParsing {
				if entity, advance, ok := consumeEntity(slice); ok {
					builder.WriteString(entity)
					idx += advance
					lastRune = 0
					continue
				}
			}
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		default:
			builder.WriteRune(r)
		}
		idx += size
		lastRune = r
	}
	return builder.String(), lastRune
}

func escapeHTML(text string) string {
	var builder strings.Builder
	for _, r := range text {
		switch r {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		case '"':
			builder.WriteString("&quot;")
		case '\'':
			builder.WriteString("&#39;")
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func glyphPlain(text string) string {
	var builder strings.Builder
	doubleOpen := true
	singleOpen := true
	idx := 0
	for idx < len(text) {
		slice := text[idx:]
		switch {
		case strings.HasPrefix(slice, "(c)"):
			builder.WriteRune('©')
			idx += 3
			continue
		case strings.HasPrefix(slice, "(r)"):
			builder.WriteRune('®')
			idx += 3
			continue
		case strings.HasPrefix(slice, "(tm)"):
			builder.WriteRune('™')
			idx += 4
			continue
		case strings.HasPrefix(slice, "..."):
			builder.WriteRune('…')
			idx += 3
			continue
		case strings.HasPrefix(slice, "--"):
			builder.WriteRune('—')
			idx += 2
			continue
		}
		r, size := utf8.DecodeRuneInString(slice)
		if r == '"' {
			if doubleOpen {
				builder.WriteRune('“')
			} else {
				builder.WriteRune('”')
			}
			doubleOpen = !doubleOpen
			idx += size
			continue
		}
		if r == '\'' {
			prev := rune(0)
			next := rune(0)
			if idx > 0 {
				prev, _ = utf8.DecodeLastRuneInString(text[:idx])
			}
			if idx+size < len(text) {
				next, _ = utf8.DecodeRuneInString(text[idx+size:])
			}
			if isAlphaNumeric(prev) && isAlphaNumeric(next) {
				builder.WriteRune('’')
				idx += size
				continue
			}
			if singleOpen {
				builder.WriteRune('‘')
			} else {
				builder.WriteRune('’')
			}
			singleOpen = !singleOpen
			idx += size
			continue
		}
		builder.WriteRune(r)
		idx += size
	}
	return builder.String()
}

func lastRune(text string, fallback rune) rune {
	if text == "" {
		return fallback
	}
	r, _ := utf8.DecodeLastRuneInString(text)
	return r
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for i := 0; i < len(text); i++ {
		if !isDigit(text[i]) {
			return false
		}
	}
	return true
}

func isUnicodeDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isDimNumberChar(r rune) bool {
	switch r {
	case '.', '\'', '’', '"', '”', ')', ']', '[', '$', '£', '¥', '¤', '฿', '€', '(', '½', '¼', '¾', '-':
		return true
	default:
		return unicode.IsDigit(r)
	}
}

func prevNonSpaceRune(text string, idx int, fallback rune) rune {
	for idx > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:idx])
		if !unicode.IsSpace(r) {
			return r
		}
		idx -= size
	}
	return fallback
}

func nextNonSpaceRune(text string, idx int) rune {
	for idx < len(text) {
		r, size := utf8.DecodeRuneInString(text[idx:])
		if !unicode.IsSpace(r) {
			return r
		}
		idx += size
	}
	return 0
}

func isValidLang(lang string) bool {
	if lang == "" {
		return false
	}
	parts := strings.Split(lang, "-")
	for _, part := range parts {
		if len(part) < 2 {
			return false
		}
		for _, r := range part {
			if !unicode.IsLetter(r) {
				return false
			}
		}
	}
	return true
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isOpeningQuote(prev, next rune) bool {
	if prev == 0 {
		return true
	}
	if next == 0 {
		return unicode.IsSpace(prev)
	}
	if prev == ':' {
		return unicode.IsLetter(next) || unicode.IsDigit(next)
	}
	if unicode.IsSpace(prev) || prev == '(' || prev == '[' || prev == '{' || prev == '=' {
		return true
	}
	return false
}

func isLiteralQuoteContext(prev, next rune) bool {
	openers := map[rune]bool{
		'{': true,
		'[': true,
		'(': true,
		'«': true,
		'»': true,
		'‹': true,
		'›': true,
		'„': true,
		'‚': true,
		'‘': true,
		'’': true,
		'“': true,
		'”': true,
	}
	closers := map[rune]bool{
		'}': true,
		']': true,
		')': true,
		'»': true,
		'«': true,
		'›': true,
		'‹': true,
		'“': true,
		'”': true,
		'‘': true,
		'’': true,
	}
	return openers[prev] && closers[next]
}

func quoteEntity(r rune, prev, next rune) string {
	if r == '"' {
		if prev == '\'' && next == '\'' {
			return "&quot;"
		}
		if isLiteralQuoteContext(prev, next) {
			return "&quot;"
		}
		if prev == '\'' || prev == '‘' || prev == '’' {
			if unicode.IsLetter(next) || unicode.IsDigit(next) {
				return "&#8220;"
			}
		}
		if isOpeningQuote(prev, next) {
			return "&#8220;"
		}
		return "&#8221;"
	}
	if prev == '"' || prev == '“' || prev == '”' {
		if unicode.IsLetter(next) {
			return "&#8216;"
		}
	}
	if isAlphaNumeric(prev) && isAlphaNumeric(next) {
		return "&#8217;"
	}
	if (prev == 0 || unicode.IsSpace(prev)) && unicode.IsDigit(next) {
		return "&#8217;"
	}
	if isOpeningQuote(prev, next) {
		return "&#8216;"
	}
	return "&#8217;"
}

func hasClosingSingleQuote(text string, idx int) bool {
	for idx < len(text) {
		r, size := utf8.DecodeRuneInString(text[idx:])
		if r == '\'' {
			return true
		}
		if unicode.IsSpace(r) {
			return false
		}
		idx += size
	}
	return false
}
