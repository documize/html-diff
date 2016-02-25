package htmldiff

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func copyNode(to, from *html.Node) {
	to.Attr = from.Attr
	to.Data = from.Data
	to.DataAtom = from.DataAtom
	to.Namespace = from.Namespace
	to.Type = from.Type
}

func nodeEqual(base, comp *html.Node) bool {
	if comp.Data != base.Data ||
		comp.DataAtom != base.DataAtom ||
		comp.Namespace != base.Namespace ||
		comp.Type != base.Type ||
		len(comp.Attr) != len(base.Attr) {
		return false
	}
	if !attrEqual(base, comp) {
		return false
	}
	return true
}

// findBody finds the first body HTML node if it exists in the tree. Required to bypass the page title text.
func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == atom.Body {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r := findBody(c)
		if r != nil {
			return r
		}
	}
	return nil
}

// find the first leaf in the tree that is a text node
func firstLeaf(n *html.Node) (*html.Node, bool) {
	if n != nil {
		switch n.Type {
		case html.TextNode:
			return n, true
		}
		// no valid node found
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			r, ok := firstLeaf(c)
			if ok {
				return r, ok
			}
		}
	}
	return nil, false
}
