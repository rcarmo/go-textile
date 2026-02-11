package parser

import "strings"

type inlineAttrScanConfig struct {
	allowCodeSpan              bool
	stopOnRestrictedParenSpace bool
	allowBracketedPhraseStop   bool
	stopOnSpaceAfterFragment   bool
	failOnEmptyFirst           bool
	appendRestrictedFragment   bool
}

type inlineAttrScanResult struct {
	attrs           map[string]string
	fragments       []string
	rest            string
	consumed        bool
	hardFail        bool
	emptyAfterFirst bool
}

func scanInlineAttrFragments(rest string, options Options, cfg inlineAttrScanConfig) inlineAttrScanResult {
	result := inlineAttrScanResult{attrs: map[string]string{}, rest: rest}
	fragments := []string{}
	for strings.HasPrefix(result.rest, "(") || strings.HasPrefix(result.rest, "{") || strings.HasPrefix(result.rest, "[") {
		if cfg.allowCodeSpan && strings.HasPrefix(result.rest, "[@") {
			break
		}
		fragment, remaining := extractInlineAttrFragment(result.rest)
		if fragment == "" {
			break
		}
		if cfg.stopOnRestrictedParenSpace && options.Restricted && strings.HasPrefix(fragment, "(") && strings.Contains(fragment, " ") {
			if len(fragments) == 0 {
				result.hardFail = true
				result.rest = rest
				return result
			}
			result.rest = strings.TrimLeft(fragment+remaining, " \t")
			break
		}
		fragmentAttrs := parseAttributes(fragment, options)
		if fragmentAttrs == nil {
			if options.Restricted && (strings.HasPrefix(fragment, "(") || strings.HasPrefix(fragment, "{")) {
				if cfg.appendRestrictedFragment {
					fragments = append(fragments, fragment)
				}
				result.rest = strings.TrimLeft(remaining, " \t")
				result.consumed = true
				continue
			}
			if len(fragments) == 0 && !result.consumed {
				result.hardFail = true
				result.rest = rest
				return result
			}
			if cfg.allowBracketedPhraseStop && strings.HasPrefix(fragment, "[") {
				inner := strings.TrimSuffix(strings.TrimPrefix(fragment, "["), "]")
				if isBracketedPhrase(inner) {
					result.rest = strings.TrimLeft(fragment+remaining, " \t")
					break
				}
			}
			if strings.TrimSpace(remaining) != "" {
				result.rest = strings.TrimLeft(remaining, " \t")
				continue
			}
			break
		}
		if strings.TrimSpace(remaining) == "" {
			if len(fragments) == 0 && cfg.failOnEmptyFirst {
				result.emptyAfterFirst = true
				result.rest = strings.TrimLeft(remaining, " \t")
				return result
			}
			if len(fragments) == 0 && !cfg.failOnEmptyFirst {
				fragments = append(fragments, fragment)
				for k, v := range fragmentAttrs {
					result.attrs[k] = v
				}
				result.consumed = true
				result.rest = strings.TrimLeft(remaining, " \t")
				break
			}
			break
		}
		fragments = append(fragments, fragment)
		for k, v := range fragmentAttrs {
			result.attrs[k] = v
		}
		result.consumed = true
		if remaining == result.rest {
			break
		}
		if cfg.stopOnSpaceAfterFragment && len(remaining) > 0 && (remaining[0] == ' ' || remaining[0] == '\t') {
			result.rest = strings.TrimLeft(remaining, " \t")
			break
		}
		result.rest = remaining
	}
	result.fragments = fragments
	result.rest = strings.TrimLeft(result.rest, " \t")
	return result
}
