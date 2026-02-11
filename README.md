# go-textile

![go-textile icon](docs/icon-256.png)

A Go implementation of the Textile markup language with a stdlib-first parser and fixture-driven compatibility against the php-textile test suite. The renderer aims for parity with php-textile behavior without vendoring or invoking the PHP library.

## Features

### Block-level parsing
- Headings (`h1.`–`h6.`), paragraphs, and blockquotes.
- Code blocks (`bc`, `pre`) and extended block syntax (`..`).
- Lists (ordered/unordered), including nested and mixed list types.
- Tables with captions, colgroups, thead/tfoot/tbody sections, and cell alignment.
- Definition lists (classic and dash-style variants).
- Raw block handling for custom tags (optional).
- Block-level HTML wrapper detection and divider blocks (`<br>`, `<hr>`, `<img>`).

### Inline parsing
- Emphasis, strong, bold/italic, insert/delete, sub/sup, cite, code spans.
- Links (inline, reference, quoted, bracketed) and images with attributes.
- Footnotes and notelists.
- Attribute fragments (class/id/style/lang/title) with normalization and ordering.
- Glyph substitutions (quotes, dashes, ellipses, trademarks, fractions, dimension “x”).
- Acronyms and caps wrapping.
- Bracketed phrases and fractions.

### Modes and policies
- Restricted mode with HTML sanitization and scheme allowlisting.
- Lite mode for minimal Textile parsing.
- HTML5 vs. XHTML rendering (void tag style).
- Optional raw HTML block passthrough.
- URL sanitization/encoding helpers with prefix support.

## Implementation status

- ✅ All vendored php-textile fixtures pass (`go test ./...`).
- ✅ Stdlib-first parsing with manual rune scanning (no regex-heavy parsing).
- ✅ Fixture-driven test harness with filtering/limiting support.

If new php-textile fixtures are added, run the full suite to confirm parity.

## Usage

```go
package main

import (
	"fmt"

	gotextile "github.com/rcarmo/go-textile"
)

func main() {
	input := "h1. Hello"
	html, err := gotextile.TextileToHtml(input)
	if err != nil {
		panic(err)
	}
	fmt.Println(html)
}
```

### Options

```go
type Options struct {
	Lite                bool
	Restricted          bool
	Images              bool
	DimensionlessImages bool
	LinkRelationship    string
	LinkPrefix          string
	ImagePrefix         string
	LineWrap            int
	RawBlocks           bool
	BlockTags           bool
	HTML5               bool
	NoGlyphs            bool
}
```

Use `DefaultOptions()` as a baseline and override as needed:

```go
options := gotextile.DefaultOptions()
options.Restricted = true
options.HTML5 = true
html, err := gotextile.TextileToHtmlWithOptions(input, options)
```

## Testing

The test suite is driven by vendored php-textile fixtures in `test/fixtures`.

```sh
# Run all fixtures

go test ./...

# Filter fixtures

TEXTILE_FIXTURE_FILTER="links.yaml" go test ./...

# Limit fixtures

TEXTILE_FIXTURE_LIMIT=50 go test ./...
```

## Fixture provenance

See `test/fixtures/README.md` for the php-textile source/version details and included assets.

## License

MIT — see `LICENSE`.
