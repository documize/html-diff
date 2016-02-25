package htmldiff

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type posTT struct {
	nodesBefore int
	node        *html.Node
}

type posT []posTT

func getPos(n *html.Node) posT {
	if n == nil {
		return nil
	}
	var ret posT
	for root := n; root.Parent != nil && root.DataAtom != atom.Body; root = root.Parent {
		var before int
		for sib := root.Parent.FirstChild; sib != root; sib = sib.NextSibling {
			if sib.Type == html.ElementNode {
				before++
			}
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
