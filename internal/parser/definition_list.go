package parser

import (
	"strings"

	"github.com/rcarmo/go-textile/internal/document"
)

type defItem struct {
	level   int
	isTerm  bool
	content string
}

func parseClassicDefinitionList(lines []string, options Options) (*document.D, error) {
	root := document.New("dl")
	stack := []*document.D{root}
	lastDD := []*document.D{nil}
	var current *defItem
	hasItems := false
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			if current != nil {
				current.content += "\n"
			}
			continue
		}
		marker, count := defMarker(line)
		if marker == 0 {
			if current != nil {
				current.content += "\n" + strings.TrimSpace(line)
			}
			continue
		}
		if current != nil {
			if err := addDefItem(current, &stack, &lastDD, options); err != nil {
				return root, err
			}
			hasItems = true
		}
		content := strings.TrimSpace(line[count:])
		current = &defItem{level: count, isTerm: marker == ';', content: content}
	}
	if current != nil {
		if err := addDefItem(current, &stack, &lastDD, options); err != nil {
			return root, err
		}
		hasItems = true
	}
	if !hasItems {
		return nil, nil
	}
	return root, nil
}

func defMarker(line string) (rune, int) {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return 0, 0
	}
	marker := rune(trimmed[0])
	if marker != ';' && marker != ':' {
		return 0, 0
	}
	count := 0
	for _, r := range trimmed {
		if r != marker {
			break
		}
		count++
	}
	if count >= len(trimmed) {
		return 0, 0
	}
	rest := trimmed[count:]
	var ok bool
	rest, ok = skipDefListAttrs(rest)
	if !ok {
		return 0, 0
	}
	if rest == "" || rest[0] != ' ' {
		return 0, 0
	}
	return marker, count
}

func skipDefListAttrs(rest string) (string, bool) {
	for {
		switch {
		case strings.HasPrefix(rest, "("):
			end := strings.Index(rest, ")")
			if end == -1 {
				return "", false
			}
			rest = rest[end+1:]
			continue
		case strings.HasPrefix(rest, "["):
			end := strings.Index(rest, "]")
			if end == -1 {
				return "", false
			}
			rest = rest[end+1:]
			continue
		case strings.HasPrefix(rest, "{"):
			end := strings.Index(rest, "}")
			if end == -1 {
				return "", false
			}
			rest = rest[end+1:]
			continue
		}
		break
	}
	return rest, true
}

func addDefItem(item *defItem, stack *[]*document.D, lastDD *[]*document.D, options Options) error {
	level := item.level
	for len(*stack) < level {
		parentDD := (*lastDD)[len(*lastDD)-1]
		newDL := document.New("dl")
		if parentDD != nil {
			parentDD.AddChild(newDL)
		} else {
			(*stack)[len(*stack)-1].AddChild(newDL)
		}
		*stack = append(*stack, newDL)
		*lastDD = append(*lastDD, nil)
	}
	for len(*stack) > level {
		*stack = (*stack)[:len(*stack)-1]
		*lastDD = (*lastDD)[:len(*lastDD)-1]
	}

	dl := (*stack)[level-1]
	content := item.content
	if item.isTerm && level == 1 && dl.Attr == nil {
		attrs, rest := parseInlineAttributes(content, options)
		if attrs != nil && rest != content {
			dl.Attr = attrs
			content = rest
		}
	}
	if item.isTerm {
		dt, err := parseInline(content, "dt", nil, options)
		if err != nil {
			return err
		}
		dl.AddChild(dt)
		return nil
	}
	dd, err := parseInline(content, "dd", nil, options)
	if err != nil {
		return err
	}
	dl.AddChild(dd)
	(*lastDD)[level-1] = dd
	return nil
}
