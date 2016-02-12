package htmldiff

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/documize/html-diff/diff"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Config describes the way that Find() works.
type Config struct {
	Granularity                              int // TODO
	InsertedSpan, DeletedSpan, FormattedSpan []html.Attribute
}

type posT []uint64

type treeRune struct {
	leaf   *html.Node
	letter rune
	pos    posT
}

type addedT map[*html.Node]bool

func getPos(n *html.Node, m addedT) posT {
	if n == nil {
		return nil
	}
	var ret posT
	for root := n; root.Parent != nil && root.DataAtom != atom.Body; root = root.Parent {
		var before uint64
		for sib := root.Parent.FirstChild; sib != root; sib = sib.NextSibling {
			if m[sib] {
				//fmt.Println("getPos skipped for ", sib, m[sib])
			} else {
				before++
			}
		}

		ret = append(ret, before)
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
		if aa != b[i] {
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
		if aa != bb && aa != bb+1 {
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
	for a := range comp.Attr {
		if comp.Attr[a].Key != base.Attr[a].Key ||
			comp.Attr[a].Namespace != base.Attr[a].Namespace ||
			comp.Attr[a].Val != base.Attr[a].Val {
			return false
		}
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

// Find all the differences in the versions of the HTML snippits, versions[0] is the original, all other versions are the edits to be compared.
// The resulting merged HTML snippits are as many as there are edits to compare.
func (c *Config) Find(versions []string) ([]string, error) {
	if len(versions) < 2 {
		return nil, errors.New("there must be at least two versions to diff, the 0th element is the base")
	}
	sourceTrees := make([]*html.Node, len(versions))
	sourceTreeRunes := make([]*[]treeRune, len(versions))
	firstLeaves := make([]int, len(versions))
	parallelErrors := make(chan error, len(versions))
	for v, vv := range versions {
		go func(v int, vv string) {
			var err error
			sourceTrees[v], err = html.Parse(strings.NewReader(vv))
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
	ctx := &appendContext{c: c, target: mergedTree, added: make(addedT)}
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

type appendContext struct {
	c                 *Config
	target, lastProto *html.Node
	lastText          string
	lastAction        rune
	lastPos           posT
	added             addedT
}

func (ap *appendContext) append(action rune, tr treeRune) {
	if tr.leaf == nil {
		return
	}
	var text string
	if tr.letter > 0 {
		text = string(tr.letter)
	}
	if ap.lastProto == tr.leaf && ap.lastAction == action && tr.leaf.Type == html.TextNode && text != "" && posEqualDepth(ap.lastPos, tr.pos) {
		ap.lastText += text
		return
	}
	ap.flush0(action, tr.leaf, tr.pos)
	if tr.leaf.Type == html.TextNode { // reload the buffer
		ap.lastText = text
		return
	}
	ap.append0(action, "", tr.leaf, tr.pos)
}

func (ap *appendContext) flush() {
	ap.flush0(0, nil, nil)
}

func (ap *appendContext) flush0(action rune, proto *html.Node, pos posT) {
	if ap.lastText != "" {
		ap.append0(ap.lastAction, ap.lastText, ap.lastProto, ap.lastPos) // flush the buffer
	}
	// reset the buffer
	ap.lastProto = proto
	ap.lastAction = action
	ap.lastPos = pos
	ap.lastText = ""
}

func (ap *appendContext) append0(action rune, text string, proto *html.Node, pos posT) {
	if proto == nil {
		return
	}
	//fmt.Println(action, text, proto)
	appendPoint, protoAncestor := ap.lastMatchingLeaf(proto, action)
	/*
		fmt.Println("targetTree: ", VizTree(ap.target))
		fmt.Println("appendPoint: ", VizTree(appendPoint))
		fmt.Println("protoAncestor: ", VizTree(appendPoint))
	*/
	if appendPoint == nil || protoAncestor == nil {
		panic("nil append point or protoAncestor") // TODO review ... make no-op, or return error?
		return
	}
	newLeaf := new(html.Node)
	copyNode(newLeaf, proto)
	if proto.Type == html.TextNode {
		newLeaf.Data = text
	}
	add := false
	if action != '=' {
		insertNode := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Span,
			Data:     "span",
		}
		switch action {
		case '+':
			insertNode.Attr = ap.c.InsertedSpan
			add = true
		case '-':
			insertNode.Attr = ap.c.DeletedSpan
		case '~':
			insertNode.Attr = ap.c.FormattedSpan
		}
		insertNode.AppendChild(newLeaf)
		if add {
			ap.added[newLeaf] = true
			//fmt.Println("Incr0", newLeaf, ap.added[newLeaf])
		}
		newLeaf = insertNode
	}
	//fmt.Println("proto", proto, "ancestor", protoAncestor)
	for proto = proto.Parent; proto != nil && proto != protoAncestor; proto = proto.Parent {
		//if proto.DataAtom == atom.Tbody {
		//	fmt.Println("proto add Tbody", VizTree(proto))
		//	fmt.Println("proto add AppendPoint", VizTree(appendPoint))
		//}
		above := new(html.Node)
		copyNode(above, proto)
		above.AppendChild(newLeaf)
		if add {
			ap.added[newLeaf] = true
			//fmt.Println("Incr1", newLeaf, ap.added[newLeaf])
		}
		newLeaf = above
	}
	appendPoint.AppendChild(newLeaf)
	if add {
		ap.added[newLeaf] = true
		//fmt.Println("Incr2", newLeaf, ap.added[newLeaf])
	} else {
		for apt := appendPoint; apt != nil; apt = apt.Parent {
			if ap.added[apt] {
				//fmt.Println("Appending non-Add to created", apt, ap.added[apt])
				delete(ap.added, apt)
			}
		}
	}
}

func (ap *appendContext) lastMatchingLeaf(proto *html.Node, action rune) (appendPoint, protoAncestor *html.Node) {
	var candidates []*html.Node
	for cand := ap.target; cand != nil; cand = cand.LastChild {
		candidates = append([]*html.Node{cand}, candidates...)
	}
	for cni, can := range candidates {
		_ = cni
		//fmt.Println("candidate", cni, can.Data)
		for anc := proto; anc.Parent != nil; anc = anc.Parent {
			if ap.leavesEqual(can, anc, action) {
				//fmt.Println("found candidate", cni, can, anc)
				return can, anc
			}
		}
	}
	return nil, nil
}

func (ap *appendContext) leavesEqual(a, b *html.Node, action rune) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.DataAtom == atom.Body && b.DataAtom == atom.Body {
		return true // body nodes are always equal
	}
	if !nodeEqual(a, b) {
		return false
	}
	gpa := getPos(a, ap.added) // what we are building
	gpb := getPos(b, nil)      // what we are adding in
	if action == '+' {
		if !posEqual(gpa, gpb) {
			//fmt.Println("leaves not equal", a, gpa, b, gpb)
			return false
		}
	} else {
		if !posSoftMatch(gpa, gpb) {
			//fmt.Println("leaves not equal", a, gpa, b, gpb)
			return false
		}
	}
	return ap.leavesEqual(a.Parent, b.Parent, action)
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

// findBody finds the body HTML node if it exists in the tree. Required to bypass the page title text.
func findBody(n *html.Node) *html.Node {
	var found *html.Node
	if n.DataAtom == atom.Body {
		found = n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r := findBody(c)
		if r != nil {
			return r
		}
	}
	return found
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
