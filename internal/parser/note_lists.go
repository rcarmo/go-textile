package parser

import "fmt"

func finalizeNoteLists(options Options) {
	for _, spec := range noteListSpecs {
		if spec.list == nil {
			continue
		}
		spec.list.Children = nil
		for _, label := range noteRefOrder {
			def, ok := noteDefs[label]
			if !ok {
				def = &noteDef{label: label, content: fmt.Sprintf("Undefined Note [#%d].", noteIndex[label]), index: noteIndex[label]}
			}
			li, err := buildNoteListItem(def, spec.marker, spec.noBacklinks, spec.singleBackref, options)
			if err != nil {
				continue
			}
			spec.list.AddChild(li)
		}
		if spec.includeUnreferenced {
			for _, label := range noteDefOrder {
				def := noteDefs[label]
				if def == nil || def.referenced {
					continue
				}
				li, err := buildNoteListItem(def, spec.marker, spec.noBacklinks, spec.singleBackref, options)
				if err != nil {
					continue
				}
				spec.list.AddChild(li)
			}
		}
	}
}
