package parser

import "strings"

func blockHasDefinition(lines []string) bool {
	for _, line := range lines {
		if strings.Contains(line, ":=") {
			return true
		}
	}
	return false
}
