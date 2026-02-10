package gotextile

// Options represents configuration switches for the Textile renderer.
// These map to the php-textile configuration used in the fixture suite.
type Options struct {
	// Lite disables most block-level parsing for a lighter-weight subset of Textile.
	Lite bool
	// Restricted sanitizes HTML and enforces scheme allowlists for links/images.
	Restricted bool
	// Images enables inline image handling (when false, images are left as text).
	Images bool
	// DimensionlessImages drops width/height attributes when true.
	DimensionlessImages bool
	// LinkRelationship sets the rel attribute on generated links (e.g., "nofollow").
	LinkRelationship string
	// LinkPrefix prepends a prefix to relative link URLs.
	LinkPrefix string
	// ImagePrefix prepends a prefix to relative image URLs.
	ImagePrefix string
	// LineWrap controls newline handling; 0 replaces newlines with spaces, -1 preserves them.
	LineWrap int
	// RawBlocks enables passthrough for raw block tags with custom namespaces.
	RawBlocks bool
	// BlockTags enables block-level HTML tag parsing/wrapping.
	BlockTags bool
	// HTML5 switches output to HTML5-style void tags and alignment classes.
	HTML5 bool
	// NoGlyphs disables glyph substitutions (e.g., smart quotes, ellipses).
	NoGlyphs bool
}

// DefaultOptions returns the baseline php-textile-compatible defaults.
func DefaultOptions() Options {
	return Options{
		Images:    true,
		LineWrap:  -1,
		BlockTags: true,
	}
}
