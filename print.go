package htmldiff

import "golang.org/x/net/html"
import "strings"

func VizTree(n *html.Node) string {
	return vizTree(n, 0, "")
}

func vizTree(n *html.Node, l int, s string) string {
	if n == nil {
		return s
	}
	for i := 0; i < l; i++ {
		s += "-"
	}
	s += ">"
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
	s += "\n"
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s = vizTree(c, l+1, s)
	}
	return s
}
