package parser

import (
	"strings"

	"github.com/rcarmo/go-textile/internal/document"
)

func buildEscapedParagraph(text string) *document.D {
	p := document.New("p")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		escaped, _ := escapeAndGlyphWithPrev(line, 0)
		p.AddChild(document.Text(escaped, true))
		if i < len(lines)-1 {
			p.AddChild(&document.D{Tag: "br"})
		}
	}
	return p
}
