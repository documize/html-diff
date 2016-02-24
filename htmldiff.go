package htmldiff

import (
	"bytes"
	"errors"
	"strings"

	"github.com/documize/html-diff/diff"

	"golang.org/x/net/html"
)

// Config describes the way that HTMLdiff() works.
type Config struct {
	Granularity                             int              // how many letters to put together for a change, if possible
	InsertedSpan, DeletedSpan, ReplacedSpan []html.Attribute // the attributes for the span tags wrapping changes
	CleanTags                               []string         // HTML tags to clean from the input
}

// HTMLdiff finds all the differences in the versions of HTML snippits,
// versionsRaw[0] is the original, all other versions are the edits to be compared.
// The resulting merged HTML snippits are as many as there are edits to compare.
func (c *Config) HTMLdiff(versionsRaw []string) ([]string, error) {
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
					tr := make([]treeRune, 0, 1024)
					sourceTreeRunes[v] = &tr
					renderTreeRunes(sourceTrees[v], &tr)
					leaf1, ok := firstLeaf(findBody(sourceTrees[v]))
					if leaf1 == nil || !ok {
						firstLeaves[v] = 0 // probably wrong
					} else {
						for x, y := range tr {
							if y.leaf == leaf1 {
								firstLeaves[v] = x
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

	// now all the input trees are buit, we can do the merge
	mergedHTMLs := make([]string, len(versions)-1)

	for m := range mergedHTMLs {
		go func(m int) {
			dd := diffData{a: sourceTreeRunes[0], b: sourceTreeRunes[m+1]}
			changes := diff.Diff(len(*sourceTreeRunes[0]), len(*sourceTreeRunes[m+1]), dd)
			changes = granular(c.Granularity, dd, changes)
			mergedTree, err := c.walkChanges(changes, sourceTreeRunes[0], sourceTreeRunes[m+1], firstLeaves[0], firstLeaves[m+1])
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

func (c *Config) walkChanges(changes []diff.Change, ap, bp *[]treeRune, aIdx, bIdx int) (*html.Node, error) {
	mergedTree, err := html.Parse(strings.NewReader("<html><head></head><body></body></html>"))
	if err != nil {
		return nil, err
	}
	a := *ap
	b := *bp
	ctx := &appendContext{c: c, target: mergedTree}
	for _, change := range changes {
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
	ctx.sortAndWrite()
	return mergedTree, nil
}
