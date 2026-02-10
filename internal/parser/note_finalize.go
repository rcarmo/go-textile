package parser

func finalizeNoteRefs() {
	for label, refs := range noteRefNodes {
		def := noteDefs[label]
		defined := def != nil && def.defined
		for _, ref := range refs {
			if noteNoLinks {
				if ref.link != nil {
					ref.link.Attr = nil
				}
				if ref.span != nil {
					ref.span.Attr = nil
				}
				continue
			}
			if !defined {
				if ref.link != nil {
					ref.link.Attr = nil
				}
				if ref.span != nil {
					ref.span.Attr = nil
				}
			}
		}
	}
}
