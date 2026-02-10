package document

import "strings"

func renderTableChildren(d *D, depth int, blockContext bool) string {
	if len(d.Children) == 0 {
		return ""
	}
	var builder strings.Builder
	for i, child := range d.Children {
		if i > 0 {
			builder.WriteString("\n")
		}
		childDepth := depth + 1
		if child.Tag == "tr" {
			childDepth = depth + 2
		}
		builder.WriteString(renderNode(child, childDepth, true))
	}
	return builder.String()
}
