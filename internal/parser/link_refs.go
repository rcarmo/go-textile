package parser

import "strings"

func isValidLinkRefLabel(label string) bool {
	if label == "" {
		return false
	}
	for _, r := range label {
		if isAlphaNumeric(r) || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func collectLinkRefs(lines []string) ([]string, map[string]string) {
	refs := map[string]string{}
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			end := strings.Index(trimmed, "]")
			if end > 1 {
				label := trimmed[1:end]
				rest := strings.TrimSpace(trimmed[end+1:])
				if rest != "" && !strings.Contains(rest, " ") && !strings.HasPrefix(rest, "[") && isValidLinkRefLabel(label) {
					refs[label] = rest
					continue
				}
			}
		}
		filtered = append(filtered, line)
	}
	return filtered, refs
}
