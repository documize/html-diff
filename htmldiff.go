package htmldiff

import (
	"bytes"
	"fmt"
	"strings"
	"errors"

	"github.com/documize/html-diff/diff"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Config describes the way that Find() works.
type Config struct {
	Granuality                               int // marked changes should be at least this many runes if possible
	InsertedSpan, DeletedSpan, FormattedSpan []html.Attribute
}

type treeRune struct {
	leaf   *html.Node
	letter rune
}

func (tr treeRune) String() string {
	if tr.leaf.Type == html.TextNode {
		return fmt.Sprintf("%s", string(tr.letter))
	}
	return fmt.Sprintf("<%s>", tr.leaf.Data)
}

type diffData struct {
	a, b *[]treeRune
}

// Equal exists to fulfill the diff.Data interface.
func (dd diffData) Equal(i, j int) bool {
	if (*dd.a)[i].letter != (*dd.b)[j].letter {
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

func nodeEqualExText(base, comp *html.Node) bool {
	if comp.DataAtom != base.DataAtom ||
		comp.Namespace != base.Namespace ||
		comp.Type != base.Type ||
		len(comp.Attr) != len(base.Attr) {
		return false
	}
	for a := range comp.Attr {
		if comp.Attr[a].Key != base.Attr[a].Key ||
			comp.Attr[a].Namespace != base.Attr[a].Namespace ||
			comp.Attr[a].Val != base.Attr[a].Val {
			return false
		}
	}
	if comp.Data != base.Data && base.Type != html.TextNode {
		return false // only test for the same data if not a text node
	}
	return true
}

func renderTreeRunes(n *html.Node, tr *[]treeRune) {
	if n.FirstChild == nil { // it is a leaf node
		if n.Type == html.TextNode {
			for _, r := range []rune(n.Data) {
				*tr = append(*tr, treeRune{leaf: n, letter: r})
			}
		} else {
			*tr = append(*tr, treeRune{leaf: n})
		}
	} else {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderTreeRunes(c, tr)
		}
	}
}

// Find all the differences in the versions of the HTML snippits, versions[0] is the original, all other versions are the edits to be compared.
// The resulting merged HTML snippits are as many as there are edits to compare. 
func (c *Config) Find(versions []string) ([]string, error) {
	if len(versions) < 2 {
		return nil, errors.New("there must be at least two versions to diff, the 0th element is the base")
	}
	sourceTrees := make([]*html.Node, len(versions))
	sourceTreeRunes := make([]*[]treeRune, len(versions))
	parallelErrors := make(chan error)
	for v, vv := range versions {
		go func(v int, vv string) {
			var err error
			sourceTrees[v], err = html.Parse(strings.NewReader(vv))
			tr := make([]treeRune, 0, 1024)
			sourceTreeRunes[v] = &tr
			renderTreeRunes(sourceTrees[v], &tr)
			//for x, y := range tr {
			//	fmt.Printf("Tree Runes: %d %s %#v\n", x, string(y.letter), y.leaf.Type)
			//}
			parallelErrors <- err
		}(v, vv)
	}
	for _ = range versions {
		if err := <-parallelErrors; err != nil {
			return nil, err
		}
	}
	// now all the input trees are buit, we can do the merge
	mergedHTMLs := make([]string, len(versions)-1)

	for m := range mergedHTMLs {
		go func(m int) {
			changes := diff.Diff(len(*sourceTreeRunes[0]), len(*sourceTreeRunes[m+1]),
				diffData{a: sourceTreeRunes[0], b: sourceTreeRunes[m+1]})
			//fmt.Printf("Changes: %d %#v\n", m, changes)
			if len(changes) == 0 { // no changes, so just return the original version
				mergedHTMLs[m] = versions[0]
				parallelErrors <- nil
				return
			}
			mergedTree, err := c.walkChanges(changes, sourceTreeRunes[0], sourceTreeRunes[m+1])
			if err != nil {
				parallelErrors <- err
				return
			}

			var mergedHTMLbuff bytes.Buffer
			err = html.Render(&mergedHTMLbuff, mergedTree)
			if err != nil {
				parallelErrors <- err
				return
			}
			mergedHTML := mergedHTMLbuff.Bytes()
			pfx := []byte("<html><head></head><body>")
			sfx := []byte("</body></html>")
			if bytes.HasPrefix(mergedHTML, pfx) && bytes.HasSuffix(mergedHTML, sfx) {
				mergedHTML = bytes.TrimSuffix(bytes.TrimPrefix(mergedHTML, pfx), sfx)
				mergedHTMLs[m] = string(mergedHTML)
				parallelErrors <- nil
				return
			}
			parallelErrors <- errors.New("correct render wrapper HTML not found: " + string(mergedHTML))
		}(m)
	}
	for _ = range mergedHTMLs {
		if err := <-parallelErrors; err != nil {
			return nil, err
		}
	}
	return mergedHTMLs, nil
}

func (c *Config) walkChanges(changesDetail []diff.Change, ap, bp *[]treeRune) (*html.Node, error) {
	var text string
	var proto *html.Node
	mergedTree, err := html.Parse(strings.NewReader("<html><head></head><body></body></html>"))
	if err != nil {
		return nil, err
	}
	a := *ap
	b := *bp
	changes := diff.Granular(c.Granuality, changesDetail)
	aIdx, bIdx := 1, 1 // entry 0 is the "<html><head></head><body>" entry
	for _ /*i*/, change := range changes {
		//fmt.Printf("Change %d %#v\n", i, change)
		text = ""
		for aIdx < change.A && bIdx < change.B {
			if a[aIdx].letter == 0 {
				c.append('=', "", a[aIdx].leaf, mergedTree)
			} else {
				text += string(a[aIdx].letter)
			}
			proto = a[aIdx].leaf
			aIdx++
			bIdx++
		}
		if text != "" {
			c.append('=', text, proto, mergedTree)
		}
		if change.Del == change.Ins && change.Del > 0 {
			for i := 0; i < change.Del; i++ {
				if a[aIdx+i].letter != b[bIdx+i].letter {
					goto textDifferent
				}
			}
			text = ""
			for i := 0; i < change.Del; i++ {
				if a[aIdx].letter == 0 {
					c.append('~', "", a[aIdx].leaf, mergedTree)
				} else {
					text += string(a[aIdx].letter)
				}
				proto = b[bIdx].leaf // use the edited formatting
				aIdx++
				bIdx++
			}
			c.append('~', text, proto, mergedTree)
			goto textSame
		}
	textDifferent:
		text = ""
		for i := 0; i < change.Del; i++ {
			if a[aIdx].letter == 0 {
				c.append('-', "", a[aIdx].leaf, mergedTree)
			} else {
				text += string(a[aIdx].letter)
			}
			proto = a[aIdx].leaf
			aIdx++
		}
		if text != "" {
			c.append('-', text, proto, mergedTree)
		}
		text = ""
		for i := 0; i < change.Ins; i++ {
			if b[bIdx].letter == 0 {
				c.append('+', "", b[bIdx].leaf, mergedTree)
			} else {
				text += string(b[bIdx].letter)
			}
			proto = b[bIdx].leaf
			bIdx++
		}
		if text != "" {
			c.append('+', text, proto, mergedTree)
		}
	textSame:
	}
	text = ""
	for aIdx < len(a) && bIdx < len(b) {
		if a[aIdx].letter == 0 {
			c.append('=', "", a[aIdx].leaf, mergedTree)
		} else {
			text += string(a[aIdx].letter)
		}
		proto = a[aIdx].leaf
		aIdx++
		bIdx++
	}
	if text != "" {
		c.append('=', text, proto, mergedTree)
	}
	return mergedTree, nil
}

func (c *Config) append(action rune, text string, proto, target *html.Node) {
	//fmt.Println(action, text, proto, target)
	appendPoint, protoAncestor := lastMatchingLeaf(target, proto)
	if appendPoint == nil || protoAncestor == nil {
		panic("nil append point or protoAncestor") // TODO make no-op, or return error
	}
	newLeaf := new(html.Node)
	copyNode(newLeaf, proto)
	if proto.Type == html.TextNode {
		newLeaf.Data = text // TODO
	}
	if action != '=' {
		insertNode := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Span,
			Data:     "span",
		}
		switch action {
		case '+':
			insertNode.Attr = c.InsertedSpan
		case '-':
			insertNode.Attr = c.DeletedSpan
		case '~':
			insertNode.Attr = c.FormattedSpan
		}
		insertNode.AppendChild(newLeaf)
		newLeaf = insertNode
	}
	//fmt.Println("proto", proto, "ancestor", protoAncestor)
	for proto.Parent != protoAncestor {
		//fmt.Println("proto add", proto)
		above := new(html.Node)
		copyNode(above, proto.Parent)
		above.AppendChild(newLeaf)
		newLeaf = above
		proto = proto.Parent
	}
	appendPoint.AppendChild(newLeaf)
}

func lastMatchingLeaf(tree, proto *html.Node) (appendPoint, protoAncestor *html.Node) {
	var candidates []*html.Node
	for cand := tree; cand != nil; cand = cand.LastChild {
		candidates = append([]*html.Node{cand}, candidates...)
	}
	//for cni, cn := range candidates {
	//	fmt.Println("candidates", cni, cn.Data)
	//}

	for _, can := range candidates {
		for anc := proto; anc != nil; anc = anc.Parent {
			if leavesEqual(anc, can) {
				return can, anc
			}
		}
	}

	return nil, nil
}

func leavesEqual(a, b *html.Node) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !nodeEqual(a, b) {
		return false
	}
	return leavesEqual(a.Parent, b.Parent)
}

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
	for a := range comp.Attr {
		if comp.Attr[a].Key != base.Attr[a].Key ||
			comp.Attr[a].Namespace != base.Attr[a].Namespace ||
			comp.Attr[a].Val != base.Attr[a].Val {
			return false
		}
	}
	return true
}
