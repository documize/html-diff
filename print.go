package htmldiff

import "golang.org/x/net/html"
import "strings"

// vizTree provides a text visualisation of the given html.Node tree, one node per line, stopping at the target.
func vizTree(n, target *html.Node, amended amendedT) string {
	r, _ := vizTree0(n, target, amended, 0, "")
	return r
}

// nodeLevel returns the prefix to show the depth of the node
func nodeLevel(l int, amend rune) (s string) {
	if amend == 0 {
		amend = '='
	}
	for i := 0; i < l; i++ {
		s += string(amend)
	}
	s += ">"
	return s
}

// vizTree0 is the recursive node tree walker.
func vizTree0(n, target *html.Node, amended amendedT, l int, s string) (string, bool) {
	if n == nil {
		return s, true
	}
	s += nodeLevel(l, amended[n])
	switch n.Type {
	case html.ErrorNode:
		s += " Error: "
	case html.TextNode:
		s += " Text: "
	case html.DocumentNode:
		s += "Document: "
	case html.ElementNode:
		s += "Element: "
	case html.CommentNode:
		s += "Comment: "
	case html.DoctypeNode:
		s += "DocType: "
	}
	if len(n.Data) > 10 {
		s += strings.Replace(n.Data[:10], "\n", "", -1)
	} else {
		s += strings.Replace(n.Data, "\n", "", -1)
	}
	if n == target {
		return s + " (Target)\n", true
	}
	s += "\n"
	for c := n.FirstChild; c != nil; c = c.NextSibling {
        var found bool 
		s, found = vizTree0(c, target, amended, l+1, s)
		if found {
			return s, true
		}
	}
	return s, false
}
