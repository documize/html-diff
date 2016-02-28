package htmldiff

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func delAttr(attr []html.Attribute, ai int) (ret []html.Attribute) {
	if len(attr) <= 1 || ai >= len(attr) {
		return nil
	}
	return append(attr[:ai], attr[ai+1:]...)
}

// clean() obviously normalises styles/colspan and removes any CleanTags specified, along with newlines;
// but less obviously (as a side-effect of Parse/Render) makes all the character handling (for example "&#160;" as utf-8) the same.
// TODO more cleaning of the input HTML, as required.
func (c *Config) clean(raw string) (io.Reader, error) {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return nil, err
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for ai := 0; ai < len(n.Attr); ai++ {
				a := n.Attr[ai]
				switch {
				case strings.ToLower(a.Key) == "style":
					if strings.TrimSpace(a.Val) == "" { // delete empty styles
						n.Attr = delAttr(n.Attr, ai)
						ai--
					} else { // tidy non-empty styles
						// TODO there could be more here to make sure the style entries are in the same order etc.
						n.Attr[ai].Val = strings.Replace(a.Val, " ", "", -1)
						if !strings.HasSuffix(n.Attr[ai].Val, ";") {
							n.Attr[ai].Val += ";"
						}
					}
				case n.DataAtom == atom.Td &&
					strings.ToLower(a.Key) == "colspan" &&
					strings.TrimSpace(a.Val) == "1":
					n.Attr = delAttr(n.Attr, ai)
					ai--
				}
			}
		}
	searchChildren:
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			switch ch.Type {
			case html.ElementNode:
				for _, rr := range c.CleanTags {
					if rr == ch.Data {
						n.RemoveChild(ch)
						goto searchChildren
					}
				}
			}
			f(ch)
		}
	}
	f(doc)
	var buf bytes.Buffer
	err = html.Render(&buf, doc)
	return &buf, err
}
