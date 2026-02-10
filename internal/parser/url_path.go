package parser

import "strings"

func encodeURLPathAndQuery(value string, options Options, allowPlus bool, allowColon bool) string {
	fragment := ""
	if idx := strings.Index(value, "#"); idx != -1 {
		fragment = value[idx+1:]
		value = value[:idx]
	}
	query := ""
	if idx := strings.Index(value, "?"); idx != -1 {
		query = value[idx+1:]
		value = value[:idx]
	}
	encoded := encodeURLFragment(value, options, allowPlus, allowColon, false, false)
	if query != "" {
		encoded += "?" + encodeURLFragment(query, options, allowPlus, allowColon, true, true)
	}
	if fragment != "" {
		encoded += "#" + encodeURLFragment(fragment, options, allowPlus, allowColon, true, true)
	}
	return encoded
}
