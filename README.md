# html-diff

Calculate difference between two HTML snippets.

An active work-in-progress, very simple tests pass, but an incomplete solution as yet.

Only deals with body HTML, so no headers, only what is within the body element.

Vendors "github.com/mb0/diff" in the diff directory.

Does not currently vendor "golang.org/x/net/html" or "golang.org/x/net/html/atom".

Running the tests will create output files in testout/*.html.
