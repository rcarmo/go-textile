package parser

import "strings"

func decodePercentEncoding(value string) (string, bool) {
	if !strings.Contains(value, "%") {
		return value, true
	}
	var builder strings.Builder
	for i := 0; i < len(value); {
		if value[i] != '%' {
			builder.WriteByte(value[i])
			i++
			continue
		}
		if i+2 >= len(value) {
			return "", false
		}
		high := fromHex(value[i+1])
		low := fromHex(value[i+2])
		if high < 0 || low < 0 {
			return "", false
		}
		builder.WriteByte(byte(high*16 + low))
		i += 3
	}
	return builder.String(), true
}

func hasPercentEncoding(value string) bool {
	for i := 0; i+2 < len(value); i++ {
		if value[i] != '%' {
			continue
		}
		if fromHex(value[i+1]) >= 0 && fromHex(value[i+2]) >= 0 {
			return true
		}
	}
	return false
}

func fromHex(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10
	default:
		return -1
	}
}
