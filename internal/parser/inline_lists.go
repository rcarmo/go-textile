package parser

import "github.com/rcarmo/go-textile/internal/document"

type inlineListJoinConfig struct {
	tag                string
	attrs              map[string]string
	joiner             *document.D
	addJoinerWhenEmpty bool
}

func parseInlineWithListSuffix(lines []string, options Options, cfg inlineListJoinConfig) (*document.D, bool, error) {
	idx := firstListLineIndex(lines)
	if idx <= 0 {
		return nil, false, nil
	}
	prefixLines := lines[:idx]
	listLines := lines[idx:]
	list, err := parseListLines(listLines, options)
	if err != nil {
		return nil, true, err
	}
	if list == nil {
		return nil, false, nil
	}
	inlineNode, err := parseInlineLines(prefixLines, "inline", nil, options)
	if err != nil {
		return nil, true, err
	}
	container := document.New(cfg.tag)
	container.Attr = cfg.attrs
	container.Children = append(container.Children, inlineNode.Children...)
	if cfg.joiner != nil && (cfg.addJoinerWhenEmpty || len(inlineNode.Children) > 0) {
		container.AddChild(cfg.joiner)
	}
	container.AddChild(list)
	return container, true, nil
}
