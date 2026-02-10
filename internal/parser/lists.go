package parser

import (
	"strconv"
	"strings"

	"github.com/rcarmo/go-textile/internal/document"
)

type listItem struct {
	lines []string
	node  *document.D
}

func parseMixedList(lines []string, options Options) (*document.D, error) {
	var root *document.D
	listStack := []*document.D{}
	itemStack := []*listItem{}
	finalize := func(level int) error {
		if level < 0 || level >= len(itemStack) {
			return nil
		}
		item := itemStack[level]
		if item == nil || item.node == nil {
			return nil
		}
		content := strings.Join(item.lines, "\n")
		parsed, err := parseInline(content, "li", nil, options)
		if err != nil {
			return err
		}
		children := parsed.Children
		if len(item.node.Children) > 0 {
			children = append(children, item.node.Children...)
		}
		item.node.Children = children
		item.lines = nil
		return nil
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if len(itemStack) > 0 && itemStack[len(itemStack)-1] != nil {
				itemStack[len(itemStack)-1].lines = append(itemStack[len(itemStack)-1].lines, "")
			}
			continue
		}
		markers, start, content, continuation, ok := parseListMarkers(line)
		if !ok {
			if len(itemStack) > 0 && itemStack[len(itemStack)-1] != nil {
				itemStack[len(itemStack)-1].lines = append(itemStack[len(itemStack)-1].lines, strings.TrimSpace(line))
			}
			continue
		}
		level := len(markers)
		for i := len(itemStack) - 1; i >= level-1; i-- {
			if err := finalize(i); err != nil {
				return nil, err
			}
			if i >= 0 && i < len(itemStack) {
				itemStack[i] = nil
			}
		}
		if len(listStack) > level {
			listStack = listStack[:level]
			itemStack = itemStack[:level]
		}
		for len(listStack) < level {
			marker := rune(markers[len(listStack)])
			newList := document.New(listTag(marker))
			startValue := start
			listLevel := len(listStack) + 1
			if marker == '#' && startValue == "" && continuation && len(listStack) == level-1 {
				if lastOrderedIndexByLevel[listLevel] > 0 {
					startValue = strconv.Itoa(lastOrderedIndexByLevel[listLevel] + 1)
				} else {
					startValue = "1"
				}
			}
			if marker == '#' && startValue != "" && len(listStack) == level-1 {
				newList.Attr = map[string]string{"start": startValue}
			}
			if len(listStack) == 0 {
				root = newList
			} else {
				parentItem := itemStack[len(listStack)-1]
				if parentItem != nil {
					parentItem.node.AddChild(newList)
				}
			}
			listStack = append(listStack, newList)
			itemStack = append(itemStack, nil)
		}
		currentMarker := rune(markers[level-1])
		if listStack[level-1].Tag != listTag(currentMarker) {
			for i := len(itemStack) - 1; i >= level-1; i-- {
				if err := finalize(i); err != nil {
					return nil, err
				}
				itemStack[i] = nil
			}
			listStack = listStack[:level-1]
			itemStack = itemStack[:level-1]
			newList := document.New(listTag(currentMarker))
			startValue := start
			listLevel := len(listStack) + 1
			if currentMarker == '#' && startValue == "" && continuation {
				if lastOrderedIndexByLevel[listLevel] > 0 {
					startValue = strconv.Itoa(lastOrderedIndexByLevel[listLevel] + 1)
				} else {
					startValue = "1"
				}
			}
			if currentMarker == '#' && startValue != "" {
				newList.Attr = map[string]string{"start": startValue}
			}
			if len(listStack) == 0 {
				root = newList
			} else {
				parentItem := itemStack[len(listStack)-1]
				if parentItem != nil {
					parentItem.node.AddChild(newList)
				}
			}
			listStack = append(listStack, newList)
			itemStack = append(itemStack, nil)
		}
		if level == 1 {
			attrs, rest := parseInlineAttributes(content, options)
			if attrs != nil && rest != content {
				if listStack[0].Attr == nil {
					listStack[0].Attr = attrs
				} else {
					for k, v := range attrs {
						listStack[0].Attr[k] = v
					}
				}
				content = rest
			} else if listAttrs, ok := parseListAttrsOnly(content, options); ok {
				if listStack[0].Attr == nil {
					listStack[0].Attr = listAttrs
				} else {
					for k, v := range listAttrs {
						listStack[0].Attr[k] = v
					}
				}
				continue
			}
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		item := &listItem{lines: []string{strings.TrimSpace(content)}, node: document.New("li")}
		listStack[level-1].AddChild(item.node)
		itemStack[level-1] = item
	}
	for i := len(itemStack) - 1; i >= 0; i-- {
		if err := finalize(i); err != nil {
			return nil, err
		}
	}
	if root != nil {
		updateOrderedIndices(root, 1)
	}
	return root, nil
}

func parseListMarkers(line string) (string, string, string, bool, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return "", "", "", false, false
	}
	markers := []rune{}
	idx := 0
	for idx < len(trimmed) {
		r := rune(trimmed[idx])
		if r != '*' && r != '#' {
			break
		}
		markers = append(markers, r)
		idx++
	}
	if len(markers) == 0 {
		return "", "", "", false, false
	}
	continuation := false
	if idx < len(trimmed) && trimmed[idx] == '_' && markers[len(markers)-1] == '#' {
		continuation = true
		idx++
	}
	start := ""
	if idx < len(trimmed) && markers[len(markers)-1] == '#' {
		startIdx := idx
		for idx < len(trimmed) && trimmed[idx] >= '0' && trimmed[idx] <= '9' {
			idx++
		}
		if idx > startIdx {
			start = trimmed[startIdx:idx]
		}
	}
	if idx < len(trimmed) && (trimmed[idx] == '(' || trimmed[idx] == '{' || trimmed[idx] == '[') {
		fragment, rest := extractAttributeFragment(trimmed[idx:])
		if rest == "" {
			return string(markers), start, strings.TrimSpace(fragment), continuation, true
		}
		if rest[0] == '.' {
			return string(markers), start, strings.TrimSpace(fragment), continuation, true
		}
		if rest[0] != ' ' {
			return "", "", "", false, false
		}
		content := strings.TrimSpace(fragment + rest)
		return string(markers), start, content, continuation, true
	}
	if idx >= len(trimmed) || trimmed[idx] != ' ' {
		return "", "", "", false, false
	}
	content := strings.TrimSpace(trimmed[idx+1:])
	return string(markers), start, content, continuation, true
}

func listTag(marker rune) string {
	if marker == '#' {
		return "ol"
	}
	return "ul"
}

func parseListAttrsOnly(content string, options Options) (map[string]string, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, false
	}
	if !strings.HasPrefix(trimmed, "(") && !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return nil, false
	}
	fragment, remaining := extractAttributeFragment(trimmed)
	if strings.TrimSpace(remaining) != "" {
		return nil, false
	}
	attrs := parseAttributes(fragment, options)
	if attrs == nil {
		return nil, false
	}
	return attrs, true
}
