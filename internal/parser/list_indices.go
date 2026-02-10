package parser

import (
	"strconv"

	"github.com/rcarmo/go-textile/internal/document"
)

func updateOrderedIndices(node *document.D, level int) {
	if node == nil {
		return
	}
	if node.Tag == "ol" {
		startVal := 1
		if node.Attr != nil {
			if startStr := node.Attr["start"]; startStr != "" {
				if parsed, err := strconv.Atoi(startStr); err == nil {
					startVal = parsed
				}
			}
		}
		lastOrderedIndexByLevel[level] = startVal + len(node.Children) - 1
	}
	for _, child := range node.Children {
		if child.Tag == "li" {
			for _, nested := range child.Children {
				if nested.Tag == "ol" || nested.Tag == "ul" {
					updateOrderedIndices(nested, level+1)
				}
			}
		}
	}
}
