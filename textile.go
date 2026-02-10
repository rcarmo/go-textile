package gotextile

import (
	"github.com/rcarmo/go-textile/internal/parser"
)

// TextileToHtml converts Textile markup to HTML using DefaultOptions.
func TextileToHtml(text string) (string, error) {
	return TextileToHtmlWithOptions(text, DefaultOptions())
}

// TextileToHtmlWithOptions converts Textile markup to HTML using the provided options.
//
// The returned HTML includes a trailing newline to match the fixture suite output.
func TextileToHtmlWithOptions(text string, options Options) (string, error) {
	parserOptions := parser.Options{
		Lite:                options.Lite,
		Restricted:          options.Restricted,
		Images:              options.Images,
		DimensionlessImages: options.DimensionlessImages,
		LinkRelationship:    options.LinkRelationship,
		LinkPrefix:          options.LinkPrefix,
		ImagePrefix:         options.ImagePrefix,
		LineWrap:            options.LineWrap,
		RawBlocks:           options.RawBlocks,
		BlockTags:           options.BlockTags,
		HTML5:               options.HTML5,
		NoGlyphs:            options.NoGlyphs,
	}
	doc, err := parser.ParseDocument(text, parserOptions)
	return doc.ToHtml(), err
}
