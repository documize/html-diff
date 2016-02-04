package htmldiff_test

import (
	"testing"

	"golang.org/x/net/html"

	"github.com/documize/html-diff"

	//"github.com/mb0/diff"
)

var cfg = &htmldiff.Config{
	Granuality:    5,
	InsertedSpan:  []html.Attribute{{Key: "style", Val: "color: green; text-decoration: underline;"}},
	DeletedSpan:   []html.Attribute{{Key: "style", Val: "color: red; text-decoration: line-through;"}},
	FormattedSpan: []html.Attribute{{Key: "style", Val: "color: blue; text-decoration: overline;"}},
}

type simpleTest struct {
	versions, diffs []string
}

var simpleTests = []simpleTest{
	{[]string{"chinese中文", "chinese中文", "中文", "chinese"},
		[]string{"chinese中文",
			`<span style="color: red; text-decoration: line-through;">chinese</span>中文`,
			`chinese<span style="color: red; text-decoration: line-through;">中文</span>`}},

	{[]string{"hElLo is that documize!", "Hello is that Documize?"},
		[]string{`<span style="color: red; text-decoration: line-through;">hElL</span><span style="color: green; text-decoration: underline;">Hell</span>o is that <span style="color: red; text-decoration: line-through;">d</span><span style="color: green; text-decoration: underline;">D</span>ocumize<span style="color: red; text-decoration: line-through;">!</span><span style="color: green; text-decoration: underline;">?</span>`}},

	{[]string{"abc", "<i>abc</i>", "<h1><i>abc</i></h1>"},
		[]string{`<i><span style="` + cfg.FormattedSpan[0].Val + `">abc</span></i>`,
		`<h1><i><span style="` + cfg.FormattedSpan[0].Val + `">abc</span></i></h1>`}},

	{[]string{"<p><span>def</span></p>", "def"},
		[]string{`<span style="` + cfg.FormattedSpan[0].Val + `">def</span>`}},
}

func TestSimple(t *testing.T) {

	for s, st := range simpleTests {
		res, err := cfg.Find(st.versions)
		if err != nil {
			t.Errorf("Simple test %d had error %v", s, err)
		}
		for d := range st.diffs {
			if d < len(res) {
				if res[d] != st.diffs[d] {
					t.Errorf("Simple test %d diff %d wanted: `%s` got: `%s`", s, d, st.diffs[d], res[d])
					//t.Log([]byte(res[d]))
				}
			}
		}
	}

}

/*
func TestDiff(t *testing.T) {
	a := "hElLo is that the Robin Day phone-in!"
	b := "hello is that the robin day phone in?"
	bb := b
	changes := diff.Granular(5, diff.ByteStrings(a, b)) // ignore small gaps in differences
	for l := len(changes) - 1; l >= 0; l-- {
		change := changes[l]
		b = b[:change.B] + "|" + b[change.B:change.B+change.Ins] + "|" + b[change.B+change.Ins:]
	}
	t.Log(b)

	changes = diff.Granular(5, diff.ByteStrings(a, bb))
	var aIdx, bIdx int
	for i, change := range changes {
		t.Logf("%d %#v\n", i, change)
		for aIdx < change.A && bIdx < change.B {
			t.Log("=", string(rune(a[aIdx])), string(rune(bb[bIdx])))
			aIdx++
			bIdx++
		}
		for i := 0; i < change.Del; i++ {
			t.Log("-", string(rune(a[aIdx])))
			aIdx++
		}
		for i := 0; i < change.Ins; i++ {
			t.Log("+", string(rune(bb[bIdx])))
			bIdx++
		}
	}
	for aIdx < len(a) && bIdx < len(bb) {
		t.Log("=", string(rune(a[aIdx])), string(rune(bb[bIdx])))
		aIdx++
		bIdx++
	}

}
*/
