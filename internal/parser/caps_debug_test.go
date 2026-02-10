package parser

import "testing"

func TestCapsDebug(t *testing.T) {
	text := "<!-- Here is a <span>HTML</span> comment -->"
	escaped, _ := escapeAndGlyphWithCaps(text, 0)
	t.Log(escaped)
}
