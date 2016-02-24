package htmldiff

import (
	"sort"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type appendContext struct {
	c                             *Config
	target, targetBody, lastProto *html.Node
	lastText                      string
	lastAction                    rune
	lastPos                       posT
	editList                      []editEntry
}

type editEntry struct {
	action  rune
	text    string
	proto   *html.Node
	pos     posT
	origSeq int
}

// Len is part of sort.Interface.
func (ap *appendContext) Len() int {
	return len(ap.editList)
}

// Swap is part of sort.Interface.
func (ap *appendContext) Swap(i, j int) {
	ap.editList[i], ap.editList[j] = ap.editList[j], ap.editList[i]
}

// Less is part of sort.Interface.
func (ap *appendContext) Less(i, j int) bool {
	ip := ap.editList[i].pos
	jp := ap.editList[j].pos
	ii := len(ip) - 1
	jj := len(jp) - 1
	for ii > 0 && jj > 0 {
		if ip[ii].nodesBefore < jp[jj].nodesBefore {
			return true
		}
		if ip[ii].nodesBefore > jp[jj].nodesBefore {
			return false
		}
		ii--
		jj--
	}
	// run out of one or the other
	return ap.editList[i].origSeq < ap.editList[j].origSeq
}

func (ap *appendContext) append(action rune, tr treeRune) {
	if tr.leaf == nil {
		return
	}
	// return if we should not be appending this type of node
	switch tr.leaf.Type {
	case html.DocumentNode:
		return
	case html.ElementNode:
		switch tr.leaf.DataAtom {
		case atom.Html:
			return
		}
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
	os := len(ap.editList)
	ap.editList = append(ap.editList, editEntry{action, text, proto, pos, os})
}

func (ap *appendContext) sortAndWrite() {
	sort.Stable(ap)
	for _, e := range ap.editList {
		ap.append1(e.action, e.text, e.proto, e.pos)
	}
}

func (ap *appendContext) append1(action rune, text string, proto *html.Node, pos posT) {
	if proto == nil {
		return
	}
	appendPoint, protoAncestor := ap.lastMatchingLeaf(proto, action, pos)
	if appendPoint == nil || protoAncestor == nil {
		return
	}
	if appendPoint.DataAtom != protoAncestor.DataAtom {
		return
	}
	newLeaf := new(html.Node)
	copyNode(newLeaf, proto)
	if proto.Type == html.TextNode {
		newLeaf.Data = text
	}
	if action != '=' {
		insertNode := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Span,
			Data:     "span",
		}
		switch action {
		case '+':
			insertNode.Attr = ap.c.InsertedSpan
		case '-':
			insertNode.Attr = ap.c.DeletedSpan
		case '~':
			insertNode.Attr = ap.c.ReplacedSpan
		}
		insertNode.AppendChild(newLeaf)
		newLeaf = insertNode
	}
	for proto = proto.Parent; proto != nil && proto != protoAncestor; proto = proto.Parent {
		above := new(html.Node)
		copyNode(above, proto)
		above.AppendChild(newLeaf)
		newLeaf = above
	}
	appendPoint.AppendChild(newLeaf)
}

func lastNonSpaceSibling(n *html.Node) *html.Node {
	var ret *html.Node
	for sib := n.FirstChild; sib != nil; sib = sib.NextSibling {
		if sib.Type == html.TextNode && strings.TrimSpace(sib.Data) == "" {
			// ignore it
		} else {
			ret = sib
		}
	}
	return ret
}

func (ap *appendContext) lastMatchingLeaf(proto *html.Node, action rune, pos posT) (appendPoint, protoAncestor *html.Node) {
	if ap.targetBody == nil {
		ap.targetBody = findBody(ap.target)
	}
	candidates := []*html.Node{}
	for cand := ap.target; cand != nil; cand = lastNonSpaceSibling(cand) {
		candidates = append([]*html.Node{cand}, candidates...)
	}
	candidates = append(candidates, ap.targetBody) // longstop
	for cni, can := range candidates {
		_ = cni
		gpa := getPos(can) // what we are building
		for anc := proto; anc.Parent != nil; anc = anc.Parent {
			if anc.Type == html.ElementNode && anc.DataAtom == atom.Html {
				break
			}
			gpb := getPos(anc) // what we are adding in
			if ap.leavesEqual(can, anc, action, gpa, gpb) {
				return can, anc
			}
		}
	}
	return ap.targetBody, proto
}

func (ap *appendContext) leavesEqual(a, b *html.Node, action rune, gpa, gpb posT) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type != html.ElementNode || b.Type != html.ElementNode {
		return false // they must both be element nodes to do a comparison
	}
	if a.DataAtom == atom.Body && b.DataAtom == atom.Body {
		return true // body nodes are always equal
	}
	if !nodeEqual(a, b) {
		return false
	}
	if !posEqualDepth(gpa, gpb) {
		return false
	}
	for i := 0; i < len(gpb); i++ {
		if gpa[i].nodesBefore < gpb[i].nodesBefore {
			return false
		}
		if gpa[i].node.DataAtom != gpb[i].node.DataAtom {
			return false
		}
	}
	return true
}
