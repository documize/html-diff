package htmldiff_test

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/documize/html-diff"
)

var cfgBench = &htmldiff.Config{
	Granularity:  6,
	InsertedSpan: []html.Attribute{{Key: "style", Val: "background-color: palegreen; text-decoration: underline;"}},
	DeletedSpan:  []html.Attribute{{Key: "style", Val: "background-color: lightpink; text-decoration: line-through;"}},
	ReplacedSpan: []html.Attribute{{Key: "style", Val: "background-color: lightskyblue; text-decoration: overline;"}},
	CleanTags:    []string{"documize"},
}

func BenchmarkHTMLdiff(b *testing.B) {
	dir := "." + string(os.PathSeparator) + "testin"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		b.Fatal(err)
	}
	testHTML := make([]string, 0, len(files))
	names := make([]string, 0, len(files))

	for _, file := range files {
		fn := file.Name()
		if strings.HasSuffix(fn, ".html") {
			ffn := dir + string(os.PathSeparator) + fn
			dat, err := ioutil.ReadFile(ffn)
			if err != nil {
				b.Fatal(err)
			}
			testHTML = append(testHTML, string(dat))
			names = append(names, fn)
		}
	}

	for n := 0; n < b.N; n++ {
		bench(testHTML, names, b)
	}
}

func bench(testHTML, names []string, b *testing.B) {
	for f := range testHTML {
		args := []string{testHTML[f], strings.ToLower(testHTML[f])}
		_, err := cfgBench.HTMLdiff(args) // don't care about the result as we are looking for crashes and time-outs
		if err != nil {
			if names[f] != "google.html" && names[f] != "bing.html" {
				b.Errorf("comparing %s with its lower-case self error: %s", names[f], err)
			}
		}
	}
}
