package parser

import "strings"

func addClass(attrs map[string]string, class string) {
	if attrs == nil {
		return
	}
	if attrs["class"] == "" {
		attrs["class"] = class
		return
	}
	parts := strings.Fields(attrs["class"])
	for _, part := range parts {
		if part == class {
			return
		}
	}
	attrs["class"] = attrs["class"] + " " + class
}
