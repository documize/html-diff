package htmldiff_test

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/documize/html-diff"
)

var cfg = &htmldiff.Config{
	Granularity:   5,
	InsertedSpan:  []html.Attribute{{Key: "style", Val: "background-color: palegreen; text-decoration: underline;"}},
	DeletedSpan:   []html.Attribute{{Key: "style", Val: "background-color: lightpink; text-decoration: line-through;"}},
	FormattedSpan: []html.Attribute{{Key: "style", Val: "background-color: lightskyblue; text-decoration: overline;"}},
}

type simpleTest struct {
	versions, diffs []string
}

var simpleTests = []simpleTest{
/**/
	{[]string{"chinese中文", "chinese中文", "中文", "chinese"},
		[]string{"chinese中文",
			`<span style="background-color: lightpink; text-decoration: line-through;">chinese</span>中文`,
			`chinese<span style="background-color: lightpink; text-decoration: line-through;">中文</span>`}},

	{[]string{"hElLo is that documize!", "Hello is that Documize?"},
		[]string{`<span style="background-color: lightpink; text-decoration: line-through;">hE</span><span style="background-color: palegreen; text-decoration: underline;">Hel</span>l<span style="background-color: lightpink; text-decoration: line-through;">L</span>o is that <span style="background-color: lightpink; text-decoration: line-through;">d</span><span style="background-color: palegreen; text-decoration: underline;">D</span>ocumize<span style="background-color: lightpink; text-decoration: line-through;">!</span><span style="background-color: palegreen; text-decoration: underline;">?</span>`}},

	{[]string{"abc", "<i>abc</i>", "<h1><i>abc</i></h1>"},
		[]string{`<i><span style="` + cfg.FormattedSpan[0].Val + `">abc</span></i>`,
			`<h1><i><span style="` + cfg.FormattedSpan[0].Val + `">abc</span></i></h1>`}},

	{[]string{"<p><span>def</span></p>", "def"},
		[]string{`<span style="` + cfg.FormattedSpan[0].Val + `">def</span>`}},

	{[]string{`Documize Logo:<img src="http://documize.com/img/documize-logo.png" alt="Documize">`,
		"Documize Logo:", `<img src="http://documize.com/img/documize-logo.png" alt="Documize">`},
		[]string{`Documize Logo:<span style="background-color: lightpink; text-decoration: line-through;"><img src="http://documize.com/img/documize-logo.png" alt="Documize"/></span>`,
			`<span style="background-color: lightpink; text-decoration: line-through;">Documize Logo:</span><img src="http://documize.com/img/documize-logo.png" alt="Documize"/>`}},

	{[]string{"<ul><li>1</li><li>2</li><li>3</li></ul>",
		"<ul><li>one</li><li>two</li><li>three</li></ul>",
		"<ul><li>1</li><li><i>2</i></li><li>3</li><li>4</li></ul>"},
		[]string{`<ul><li><span style="background-color: lightpink; text-decoration: line-through;">1</span></li><li><span style="background-color: lightpink; text-decoration: line-through;">2</span></li><li><span style="background-color: lightpink; text-decoration: line-through;">3</span></li><li><span style="background-color: palegreen; text-decoration: underline;">one</span></li><li><span style="background-color: palegreen; text-decoration: underline;">two</span></li><li><span style="background-color: palegreen; text-decoration: underline;">three</span></li></ul>`,
			`<ul><li>1</li><li><i><span style="background-color: lightskyblue; text-decoration: overline;">2</span></i></li><li>3</li><li><span style="background-color: palegreen; text-decoration: underline;">4</span></li></ul>`}},

	{[]string{doc1 + doc2 + doc3 + doc4, doc1 + doc2 + doc3 + doc4, doc1 + doc3 + doc4, doc1 + "<i>" + doc2 + "</i>" + doc3 + doc4,
		doc1 + doc2 + "inserted" + doc3 + doc4, doc1 + doc2 + doc3 + "<div><p>New Div</p></div>" + doc4},
		[]string{``,
			`<li><span style="background-color: lightpink; text-decoration: line-through;">Automated document formatting</span></li>`,
			`<span style="background-color: lightskyblue; text-decoration: overline;">Automated document formatting</span>`,
			`<span style="background-color: palegreen; text-decoration: underline;">inserted</span>`,
			``}},

	{[]string{bbcNews1 + bbcNews2, bbcNews1 + "<div><i>HTML-Diff-Inserted</i></div>" + bbcNews2},
		[]string{`<div><i><span style="` + cfg.InsertedSpan[0].Val + `">HTML-Diff-Inserted</span></i></div>`}},
/**/
	{[]string{`<table border="1" style="width:100%">
  <tr>
    <td>Jack</td>
    <td>and</td> 
    <td>Jill</td>
  </tr>
  <tr>
    <td>Derby</td>
    <td>and</td> 
    <td>Joan</td>
  </tr>
</table>`, /**/
		`<table border="1" style="width:100%">
  <tr>
    <td>Jack</td>
    <td><b>and</b></td> 
    <td>Vera</td>
  </tr>
  <tr>
    <td>Derby</td>
    <td><i>locomotive</i></td> 
    <td>works</td>
  </tr>
</table>`,
		`<table border="1" style="width:100%">
  <tr>
    <td>Jack</td>
    <td>and</td> 
    <td>Jill</td>
  </tr>
  <tr>
    <td>Samson</td>
    <td>and</td> 
    <td>Delilah</td>
  </tr>
  <tr>
    <td>Derby</td>
    <td>and</td> 
    <td>Joan</td>
  </tr>
</table>`, /**/
		`<table border="1" style="width:100%">
  <tr>
    <td>Jack</td>
    <td>and</td> 
    <td>Jill</td>
  </tr>
  <tr>
    <td>Samson</td>
    <td>and</td> 
    <td>Delilah</td>
  </tr>
  <tr>
    <td>Derby</td>
    <td>and</td> 
    <td>Joan</td>
  </tr>
  <tr>
    <td>Tweedledum</td>
    <td>and</td> 
    <td>Tweedledee</td>
  </tr>
</table>`},
		[]string{ /**/ `<table border="1" style="width:100%">
  <tbody><tr>
    <td>Jack</td>
    <td><b><span style="background-color: lightskyblue; text-decoration: overline;">and</span></b></td> 
    <td><span style="background-color: lightpink; text-decoration: line-through;">Jill</span><span style="background-color: palegreen; text-decoration: underline;">Vera</span></td>
  </tr>
  <tr>
    <td>Derby</td>
    <td><span style="background-color: lightpink; text-decoration: line-through;">and</span><i><span style="background-color: palegreen; text-decoration: underline;">locomotive</span></i></td> 
    <td><span style="background-color: lightpink; text-decoration: line-through;">J</span><span style="background-color: palegreen; text-decoration: underline;">w</span>o<span style="background-color: lightpink; text-decoration: line-through;">an</span><span style="background-color: palegreen; text-decoration: underline;">rks</span></td>
  </tr>
</tbody></table>`,
			`<table border="1" style="width:100%">
  <tbody><tr>
    <td>Jack</td>
    <td>and</td> 
    <td>Jill</td>
  </tr>
  <tr>
    <td><span style="background-color: palegreen; text-decoration: underline;">Samson</span></td><span style="background-color: palegreen; text-decoration: underline;">
    </span><td><span style="background-color: palegreen; text-decoration: underline;">and</span></td><span style="background-color: palegreen; text-decoration: underline;"> 
    </span><td><span style="background-color: palegreen; text-decoration: underline;">Delilah</span></td><span style="background-color: palegreen; text-decoration: underline;">
  </span></tr><span style="background-color: palegreen; text-decoration: underline;">
  </span><tr><span style="background-color: palegreen; text-decoration: underline;">
    </span><td>Derby</td>
    <td>and</td> 
    <td>Joan</td>
  </tr>
</tbody></table>`,/**/
			`<table border="1" style="width:100%">
  <tbody><tr>
    <td>Jack</td>
    <td>and</td> 
    <td>Jill</td>
  </tr>
  <tr>
    <td><span style="background-color: palegreen; text-decoration: underline;">Samson</span></td><span style="background-color: palegreen; text-decoration: underline;">
    </span><td><span style="background-color: palegreen; text-decoration: underline;">and</span></td><span style="background-color: palegreen; text-decoration: underline;"> 
    </span><td>De<span style="background-color: palegreen; text-decoration: underline;">lilah</span></td><span style="background-color: palegreen; text-decoration: underline;">
  </span></tr><span style="background-color: palegreen; text-decoration: underline;">
  </span><tr><span style="background-color: palegreen; text-decoration: underline;">
    </span><td><span style="background-color: palegreen; text-decoration: underline;">De</span>rby</td>
    <td>and</td> 
    <td>Joan</td><span style="background-color: palegreen; text-decoration: underline;">
  </span></tr><span style="background-color: palegreen; text-decoration: underline;">
  </span><tr><span style="background-color: palegreen; text-decoration: underline;">
    </span><td><span style="background-color: palegreen; text-decoration: underline;">Tweedledum</span></td><span style="background-color: palegreen; text-decoration: underline;">
    </span><td><span style="background-color: palegreen; text-decoration: underline;">and</span></td><span style="background-color: palegreen; text-decoration: underline;"> 
    </span><td><span style="background-color: palegreen; text-decoration: underline;">Tweedledee</span></td>
  </tr>
</tbody></table>`}},
}

func TestSimple(t *testing.T) {

	for s, st := range simpleTests {
		res, err := cfg.Find(st.versions)
		if err != nil {
			t.Errorf("Simple test %d had error %v", s, err)
		}
		for d := range st.diffs {
			if d < len(res) {
				fn := fmt.Sprintf("testout/simple%d-%d.html", s, d)
				err := ioutil.WriteFile(fn, []byte(res[d]), 0777)
				if err != nil {
					t.Error(err)
				}
				if !strings.Contains(res[d], st.diffs[d]) {
					if len(res[d]) < 1024 {
						t.Errorf("Simple test %d diff %d wanted: `%s` got: `%s`", s, d, st.diffs[d], res[d])
					} else {
						t.Errorf("Simple test %d diff %d failed see file: `%s`", s, d, fn)
					}
				}
			}
		}
	}

}
