package gotextile

// Options represents configuration switches for the Textile renderer.
// These map to the php-textile configuration used in the fixture suite.
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
}

// DefaultOptions returns the baseline php-textile-compatible defaults.
func DefaultOptions() Options {
	return Options{
		Images:   true,
		LineWrap: -1,
		BlockTags: true,
	}
}
