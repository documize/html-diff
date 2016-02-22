package htmldiff

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/documize/html-diff/diff"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Config describes the way that Find() works.
type Config struct {
	Granularity                              int // TODO
	InsertedSpan, DeletedSpan, FormattedSpan []html.Attribute
	CleanTags                                []string
}

type posTT struct {
	nodesBefore int
	 node        *html.Node
}

type posT []posTT

type treeRune struct {
	leaf   *html.Node
	letter rune
	pos    posT
}

type amendedT map[*html.Node]rune

func getPos(n *html.Node, m amendedT) posT {
	if n == nil {
		return nil
	}
	var ret posT
	for root := n; root.Parent != nil && root.DataAtom != atom.Body; root = root.Parent {
		var before int
		for sib := root.Parent.FirstChild; sib != root; sib = sib.NextSibling {
			//if sib.Type == html.ElementNode {
				switch m[sib] {
				case '+':
					//fmt.Println("getPos skipped for ", sib, m[sib])
				default:
					before++
				}
			//}
		}

		ret = append(ret, posTT{before, root})
	}
	return ret
}

func posEqualDepth(a, b posT) bool {
	return len(a) == len(b)
}

func posEqual(a, b posT) bool {
	if !posEqualDepth(a, b) {
		return false
	}
	for i, aa := range a {
		if aa.nodesBefore != b[i].nodesBefore {
			return false
		}
	}
	return true
}

func posSoftMatch(a, b posT) bool {
	if !posEqualDepth(a, b) {
		return false
	}
	for i, aa := range a {
		bb := b[i]
		if aa.nodesBefore > bb.nodesBefore {
			return false
		}
	}
	return true
}

func (tr treeRune) String() string {
	if tr.leaf.Type == html.TextNode {
		return fmt.Sprintf("%s %v", string(tr.letter), tr.pos)
	}
	return fmt.Sprintf("<%s> %v", tr.leaf.Data, tr.pos)
}

type diffData struct {
	a, b *[]treeRune
}

// Equal exists to fulfill the diff.Data interface.
func (dd diffData) Equal(i, j int) bool {
	ii := (*dd.a)[i]
	jj := (*dd.b)[j]
	if ii.letter != jj.letter && ii.letter > 0 && jj.letter > 0 {
		return false
	}
	if !posEqualDepth(ii.pos, jj.pos) {
		return false
	}
	return nodeTreeEqual(ii.leaf, jj.leaf)
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
	//if comp.Data != base.Data && base.Type != html.TextNode {
	//	return false // only test for the same data if not a text node
	//}
	return true
}

func renderTreeRunes(n *html.Node, tr *[]treeRune) {
	p := getPos(n, nil)
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
		//*tr = append(*tr, treeRune{leaf: n, letter: 0, pos: p})
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderTreeRunes(c, tr)
		}
		//*tr = append(*tr, treeRune{})
	}
}

// THIS FUNCITON TODO -- should only concatanate changes for text nodes
func granular(gran int, dd diffData, changes []diff.Change) []diff.Change {
	ret := make([]diff.Change, 0, len(changes))
	/*
		startSame := 0
		changeCount := 0
		lastAleaf, lastBleaf := (*dd.a)[0].leaf, (*dd.b)[0].leaf
	*/
	for c, cc := range changes {
		/*
			if cc.A < len(*dd.a) && cc.B < len(*dd.b) &&
				lastAleaf.Type == html.TextNode && lastBleaf.Type == html.TextNode &&
				(*dd.a)[cc.A].leaf == lastAleaf && (*dd.b)[cc.B].leaf == lastBleaf {
				// do nothing yet, queue it up until there is a difference
				changeCount++
			} else { // no match
				if changeCount > 0 { // flush
					ret = append(ret, diff.Granular(gran, changes[startSame:startSame+changeCount])...)
				}
		*/
		ret = append(ret, cc)
		_ = c
		/*
				startSame = c
				changeCount = 0
				if cc.A < len(*dd.a) && cc.B < len(*dd.b) {
					lastAleaf, lastBleaf = (*dd.a)[cc.A].leaf, (*dd.b)[cc.B].leaf
				}
			}
		*/
	}
	/*
		if changeCount > 0 { // flush
			ret = append(ret, diff.Granular(gran, changes[startSame:])...)
		}
	*/
	return ret
}

func delAttr(attr []html.Attribute, ai int) (ret []html.Attribute) {
	if len(attr) <= 1 {
		return nil
	}
	return append(attr[:ai], attr[ai+1:]...)
}

// clean() obviously removes empty styles and any CleanTags specified,
// but less obviously (as a side-effect of Parse/Render) makes all the character handling (for example "&#160;" as utf-8) the same.
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
				case strings.ToLower(a.Key) == "style" &&
					strings.TrimSpace(a.Val) == "":
					n.Attr = delAttr(n.Attr, ai)
					ai--
				case n.DataAtom == atom.Td &&
					strings.ToLower(a.Key) == "colspan" &&
					strings.TrimSpace(a.Val) == "1":
					n.Attr = delAttr(n.Attr, ai)
					ai--
				}
			}
			//if len(n.Attr) > 0 {
			//	fmt.Println("kept", n.Attr)
			//}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type == html.ElementNode {
				for _, rr := range c.CleanTags {
					if rr == ch.Data {
						//fmt.Println("delete child", ch)
						n.RemoveChild(ch)
						goto removedChild
					}
				}
			}
			f(ch)
		removedChild:
		}
	}
	f(doc)
	var buf bytes.Buffer
	err = html.Render(&buf, doc)
	return &buf, err
}

// Find all the differences in the versions of the HTML snippits, versions[0] is the original, all other versions are the edits to be compared.
// The resulting merged HTML snippits are as many as there are edits to compare.
func (c *Config) Find(versionsRaw []string) ([]string, error) {
	if len(versionsRaw) < 2 {
		return nil, errors.New("there must be at least two versions to diff, the 0th element is the base")
	}
	versions := make([]string, len(versionsRaw))
	parallelErrors := make(chan error, len(versions))
	sourceTrees := make([]*html.Node, len(versions))
	sourceTreeRunes := make([]*[]treeRune, len(versions))
	firstLeaves := make([]int, len(versions))
	for v, vvr := range versionsRaw {
		go func(v int, vvr string) {
			vv, err := c.clean(vvr)
			if err == nil {
				sourceTrees[v], err = html.Parse(vv)
				if err == nil {
					//fmt.Println(VizTree(sourceTrees[v]))
					tr := make([]treeRune, 0, 1024)
					sourceTreeRunes[v] = &tr
					renderTreeRunes(sourceTrees[v], &tr)
					//for x, y := range tr {
					//	fmt.Printf("Tree Runes rendered: %d %s %#v %#v\n", x, string(y.letter), y.leaf.Type, y.pos)
					//}
					leaf1, ok := firstLeaf(findBody(sourceTrees[v]))
					//fmt.Printf("First Leaf: %#v %v\n", leaf1, ok)
					if leaf1 == nil || !ok {
						firstLeaves[v] = 0 // probably wrong
						//fmt.Printf("First Leaf is nil or !ok: %d %v %v\n", v, leaf1, ok)
					} else {
						for x, y := range tr {
							//	fmt.Printf("Tree Runes: %d %s %#v\n", x, string(y.letter), y.leaf.Type)
							if y.leaf == leaf1 {
								firstLeaves[v] = x
								//fmt.Printf("First Leaf: %d %d %#v\n", v, x, y.leaf)
								break
							}
						}
					}
				}
			}
			parallelErrors <- err
		}(v, vvr)
	}
	for _ = range versions {
		if err := <-parallelErrors; err != nil {
			return nil, err
		}
	}
    //for v := range versions {
      //  fmt.Println("Tree:",v,vizTree(sourceTrees[v],nil,nil))
    //}
    
	// now all the input trees are buit, we can do the merge
	mergedHTMLs := make([]string, len(versions)-1)

	for m := range mergedHTMLs {
		go func(m int) {
			dd := diffData{a: sourceTreeRunes[0], b: sourceTreeRunes[m+1]}
			changes := diff.Diff(len(*sourceTreeRunes[0]), len(*sourceTreeRunes[m+1]), dd)
			//fmt.Printf("Changes: %d %#v\n", m, changes)
			/* POSSIBLE FUTURE ENHANCEMENT, BUT HEADER MUST BE REMOVED FIRST
			if len(changes) == 0 { // no changes, so just return the original version
				mergedHTMLs[m] = versions[0]
				parallelErrors <- nil
				return
			}
			*/
			changes = granular(c.Granularity, dd, changes)
			mergedTree, err := c.walkChanges(changes, sourceTreeRunes[0], sourceTreeRunes[m+1], firstLeaves[0], firstLeaves[m+1])
			if err != nil {
				parallelErrors <- err
				return
			}
			//fmt.Printf("SourceTree:\n%s\n", VizTree(sourceTrees[0]))
			//fmt.Printf("MergedTree %d:\n%s\n", m, VizTree(mergedTree))
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

func (c *Config) walkChanges(changes []diff.Change, ap, bp *[]treeRune, aIdx, bIdx int) (*html.Node, error) {
	mergedTree, err := html.Parse(strings.NewReader("<html><head></head><body></body></html>"))
	if err != nil {
		return nil, err
	}
	a := *ap
	b := *bp
	ctx := &appendContext{c: c, target: mergedTree, amended: make(amendedT)}
	for _ /*i*/, change := range changes {
		//fmt.Printf("Change %d %#v\n", i, change)
		for aIdx < change.A && bIdx < change.B {
			ctx.append('=', a[aIdx])
			aIdx++
			bIdx++
		}
		if change.Del == change.Ins && change.Del > 0 {
			for i := 0; i < change.Del; i++ {
				if a[aIdx+i].letter != b[bIdx+i].letter {
					goto textDifferent
				}
			}
			for i := 0; i < change.Del; i++ {
				ctx.append('~', b[bIdx])
				aIdx++
				bIdx++
			}
			goto textSame
		}
	textDifferent:
		for i := 0; i < change.Del; i++ {
			ctx.append('-', a[aIdx])
			aIdx++
		}
		for i := 0; i < change.Ins; i++ {
			ctx.append('+', b[bIdx])
			bIdx++
		}
	textSame:
	}
	for aIdx < len(a) && bIdx < len(b) {
		ctx.append('=', a[aIdx])
		aIdx++
		bIdx++
	}
	ctx.flush()
	return mergedTree, nil
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
	return nil, false
}
