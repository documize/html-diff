package htmldiff

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type appendContext struct {
	c                             *Config
	target, targetBody, lastProto *html.Node
	lastText                      string
	lastAction                    rune
	lastPos                       posT
	amended                       amendedT
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
	if proto == nil {
		return
	}
	//fmt.Println(action, text, proto, pos)
	appendPoint, protoAncestor := ap.lastMatchingLeaf(proto, action, pos)
	/*
		fmt.Println("targetTree: ", VizTree(ap.target))
		fmt.Println("appendPoint: ", VizTree(appendPoint))
		fmt.Println("protoAncestor: ", VizTree(appendPoint))
	*/
	if appendPoint == nil || protoAncestor == nil {
		panic("nil append point or protoAncestor") // TODO review ... make no-op, or return error?
		// return // NoOp
	}
	if appendPoint.DataAtom != protoAncestor.DataAtom {
		//fmt.Println("BAD Append:\n", vizTree(protoAncestor, proto, nil), "To:\n", vizTree(ap.target, appendPoint, ap.amended))
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
			insertNode.Attr = ap.c.FormattedSpan
		}
		insertNode.AppendChild(newLeaf)
		ap.amended[newLeaf] = action
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
		ap.amended[newLeaf] = action
		newLeaf = above
	}
	appendPoint.AppendChild(newLeaf)
	ap.amended[newLeaf] = action
	/*if action != '+' {
		// mark previously inserted parental nodes as normal
		for apt := appendPoint; apt != nil; apt = apt.Parent {
			if ap.amended[apt] == '+' {
				ap.amended[apt] = '='
			}
		}
	}*/
}

func (ap *appendContext) matchingNodes(tree, match *html.Node, pos posT, action rune) []*html.Node {
	ret := []*html.Node{}
	if len(pos) > 0 {
		//skip := 0
		lastPos := len(pos) - 1
		for ch := tree.FirstChild; ch != nil; ch = ch.NextSibling {
			//if skip >= pos[lastPos].nodesBefore {
			if ch.Type == html.ElementNode {
				ret = append(ap.matchingNodes(ch, match, pos[:lastPos], action), ret...)
			}
			//}
			//if ch.Type == html.ElementNode {
			//skip++
			//}
		}
	}
	if nodeEqual(tree, match) {
		//fmt.Printf("matchingNodes %#v %#v\n", tree.Data, match.Data)
		ret = append(ret, tree)
	}
	return ret
}

func (ap *appendContext) lastMatchingLeaf(proto *html.Node, action rune, pos posT) (appendPoint, protoAncestor *html.Node) {
	//fmt.Println("lastMatchingLeaf", proto, action, pos)
	if ap.targetBody == nil {
		ap.targetBody = findBody(ap.target)
	}
	candidates := []*html.Node{}
	if action == '+' {
		for p := range pos {
			//fmt.Println("match", pos[p].node.Data)
			candidates = append(candidates, ap.matchingNodes(ap.targetBody, pos[p].node, pos, action)...)
			//	/*fmt.Printf("level %d created candidates ", p)
			//	for _, cc := range candidates {
			//		fmt.Printf("%v ", cc.Data)
			//	}
			//	fmt.Println("")*/
		}
	} else {
		for cand := ap.target; cand != nil; cand = cand.LastChild {
			candidates = append([]*html.Node{cand}, candidates...)
		}
	}
	candidates = append(candidates, ap.targetBody) // longstop
	/*fmt.Printf("All candidates: ")
	for _, cc := range candidates {
		d := cc.Data
		if len(d) > 6 {
			d = d[6:]
		}
		d = strings.Replace(d, "\n", "", -1)
		fmt.Printf("%v ", d)
	}
	fmt.Println("")*/

	for cni, can := range candidates {
		_ = cni
		gpa := getPos(can, ap.amended) // what we are building // TODO still used???
		gpaNil := getPos(can, nil)     // what we are building
		//fmt.Println("candidate", cni, can.Data)
		for anc := proto; anc.Parent != nil; anc = anc.Parent {
			if anc.Type == html.ElementNode && anc.DataAtom == atom.Html {
				break
			}
			//fmt.Println("comparing candidate", cni, can, anc)
			gpb := getPos(anc, nil) // what we are adding in
			if ap.leavesEqual(can, anc, action, gpa, gpaNil, gpb) {
				//fmt.Println("found + candidate", cni, can, anc)
				return can, anc
			}
		}
	}
	//	fmt.Println("Did not find candidate", proto, action, pos)
	return ap.targetBody, proto
}

func (ap *appendContext) leavesEqual(a, b *html.Node, action rune, gpa, gpaNil, gpb posT) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type != html.ElementNode || a.Type != html.ElementNode {
		return false // they must both be element nodes to do a comparison
	}
	if a.DataAtom == atom.Body && b.DataAtom == atom.Body {
		return true // body nodes are always equal
	}
	if !nodeEqual(a, b) {
		return false
	}
	if action == '+' {
		if !posEqual(gpaNil, gpb) {
			return false
		}
	} else {
		if !posEqualDepth(gpaNil, gpb) {
			return false
		}
		if len(gpaNil) > 0 && gpaNil[0].nodesBefore < gpb[0].nodesBefore {
			return false
		}
	}
	return true // ap.leavesEqual(a.Parent, b.Parent, action, gpa, gpaNil, gpb)
}
