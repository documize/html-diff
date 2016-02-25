# html-diff

Calculate difference between two HTML snippets.

Usage (see example):
```
	var cfg = &htmldiff.Config{
		Granularity:  5,
		InsertedSpan: []html.Attribute{{Key: "style", Val: "background-color: palegreen;"}},
		DeletedSpan:  []html.Attribute{{Key: "style", Val: "background-color: lightpink;"}},
		ReplacedSpan: []html.Attribute{{Key: "style", Val: "background-color: lightskyblue;"}},
		CleanTags:    []string{""},
	}
	res, err := cfg.HTMLdiff([]string{previousHTML, latestHTML})
    mergedHTML := res[0]
```

First working version, pull requests welcome.

Only deals with body HTML, so no headers, only what is within the body element.

Vendors "github.com/mb0/diff" in the diff directory. Does not currently vendor "golang.org/x/net/html" or "golang.org/x/net/html/atom".

Running the tests will create output files in testout/*.html.

For fuzz-testing using https://github.com/dvyukov/go-fuzz , the Fuzz() function is in fuzz.go .
