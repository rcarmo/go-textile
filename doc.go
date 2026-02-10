// Package gotextile provides a stdlib-first Textile parser with fixture-driven
// compatibility against php-textile.
//
// The renderer supports block and inline Textile constructs (lists, tables,
// emphasis, links, images, glyph substitutions, footnotes, and more), along with
// restricted and lite modes. Use DefaultOptions as a baseline and override as
// needed.
//
// Example:
//
//	input := "h1. Hello"
//	html, err := gotextile.TextileToHtml(input)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(html)
package gotextile
