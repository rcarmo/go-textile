go-textile
==========

A Go implementation of the Textile markup language, built with stdlib-first parsing and fixture-driven compatibility against the php-textile test suite.

## Usage

```go
html := textile.TextileToHtml(input, textile.Options{})
```

### Options

```go
type Options struct {
	Restricted bool
	HTML5      bool
	LineWrap   int
	RawBlocks  bool
	NoImages   bool
	NoLinks    bool
	NoTables   bool
	NoGlyphs   bool
}
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

## License

MIT - See LICENSE
