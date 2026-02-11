package parser

import "github.com/rcarmo/go-textile/internal/document"

func buildSupLink(child *document.D, href string, attrs map[string]string, forceLink bool) (*document.D, *document.D) {
	sup := document.New("sup")
	sup.Attr = attrs
	if href == "" && !forceLink {
		sup.AddChild(child)
		return sup, nil
	}
	link := document.New("a")
	if href != "" {
		link.Attr = map[string]string{"href": href}
	}
	link.AddChild(child)
	sup.AddChild(link)
	return sup, link
}
