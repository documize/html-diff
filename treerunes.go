package htmldiff

import (
	"unicode/utf8"

	"github.com/documize/html-diff/diff"

	"golang.org/x/net/html"
)

type treeRune struct {
	leaf   *html.Node
	letter rune
	pos    posT
}

type diffData struct {
	a, b *[]treeRune
}

// Equal exists to fulfill the diff.Data interface.
func (dd diffData) Equal(i, j int) bool {
	if (*dd.a)[i].letter != (*dd.b)[j].letter {
		return false
	}
	if !posEqual((*dd.a)[i].pos, (*dd.b)[j].pos) {
		return false
	}
	return nodeTreeEqual((*dd.a)[i].leaf, (*dd.b)[j].leaf)
}

func nodeTreeEqual(leafA, leafB *html.Node) bool {
	if !nodeEqualExText(leafA, leafB) {
		return false
	}
	if leafA.Parent == nil && leafB.Parent == nil {
		return true // at the top of the tree
	}
	if leafA.Parent != nil && leafB.Parent != nil {
		return nodeEqualExText(leafA.Parent, leafB.Parent) // go up to the next level
	}
	return false // one of the leaves has a parent, the other does not
}

func attrEqual(base, comp *html.Node) bool {
	for a := range comp.Attr {
		if comp.Attr[a].Key != base.Attr[a].Key ||
			comp.Attr[a].Namespace != base.Attr[a].Namespace ||
			comp.Attr[a].Val != base.Attr[a].Val {
			return false
		}
	}
	return true
}

func nodeEqualExText(base, comp *html.Node) bool {
	if base == nil || comp == nil {
		return false
	}
	if comp.DataAtom != base.DataAtom ||
		comp.Namespace != base.Namespace ||
		comp.Type != base.Type ||
		len(comp.Attr) != len(base.Attr) {
		return false
	}
	if !attrEqual(base, comp) {
		return false
	}
	if comp.Data != base.Data && base.Type != html.TextNode {
		return false // only test for the same data if not a text node
	}
	return true
}

func estimateTreeRunes(n *html.Node) int {
	size := 0
	if n.FirstChild == nil { // it is a leaf node
		switch n.Type {
		case html.TextNode:
			size += utf8.RuneCountInString(n.Data) // len(n.Data) would be faster, but use more memory
		default:
			size++
		}
	} else {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			size += estimateTreeRunes(c)
		}
	}
	return size
}

func renderTreeRunes(n *html.Node, tr *[]treeRune) {
	p := getPos(n)
	if n.FirstChild == nil { // it is a leaf node
		switch n.Type {
		case html.TextNode:
			if len(n.Data) == 0 {
				*tr = append(*tr, treeRune{leaf: n, letter: '\u200b' /* zero-width space */, pos: p}) // make sure we catch the node, even if no data
			} else {
				for _, r := range []rune(n.Data) {
					*tr = append(*tr, treeRune{leaf: n, letter: r, pos: p})
				}
			}
		default:
			*tr = append(*tr, treeRune{leaf: n, letter: 0, pos: p})
		}
	} else {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderTreeRunes(c, tr)
		}
	}
}

// wrapper for diff.Granular() -- should only concatanate changes for similar text nodes
func granular(gran int, dd diffData, changes []diff.Change) []diff.Change {
	ret := make([]diff.Change, 0, len(changes))
	startSame := 0
	changeCount := 0
	lastAleaf, lastBleaf := (*dd.a)[0].leaf, (*dd.b)[0].leaf
	for c, cc := range changes {
		if cc.A < len(*dd.a) && cc.B < len(*dd.b) &&
			lastAleaf.Type == html.TextNode && lastBleaf.Type == html.TextNode &&
			(*dd.a)[cc.A].leaf == lastAleaf && (*dd.b)[cc.B].leaf == lastBleaf &&
			nodeEqualExText(lastAleaf, lastBleaf) { // TODO is this last constraint required?
			// do nothing yet, queue it up until there is a difference
			changeCount++
		} else { // no match
			if changeCount > 0 { // flush
				ret = append(ret, diff.Granular(gran, changes[startSame:startSame+changeCount])...)
			}
			ret = append(ret, cc)
			startSame = c + 1 // the one after this
			changeCount = 0
			if cc.A < len(*dd.a) && cc.B < len(*dd.b) {
				lastAleaf, lastBleaf = (*dd.a)[cc.A].leaf, (*dd.b)[cc.B].leaf
			}
		}
	}
	if changeCount > 0 { // flush
		ret = append(ret, diff.Granular(gran, changes[startSame:])...)
	}
	return ret
}
