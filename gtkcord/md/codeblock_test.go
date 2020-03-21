package md

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

const _fencedInline = "```thing"
const _fencedInlineHTML = `<pre><code class="language-thing"></code></pre>
`

const _fencedLanguage = "```go" + `
package main

func main() {
	fmt.Println("Hello, 世界！")
}
` + "```"
const _fencedLanguageHTML = `<pre><code class="language-go">package main

func main() {
	fmt.Println(&quot;Hello, 世界！&quot;)
}
</code></pre>
`

const _fencedBroken = "`````go" + `
package main
` + "````"
const _fencedBrokenHTML = `<pre><code class="language-go">package main
</code></pre>
`

func TestFenced(t *testing.T) {
	// Make a fenced only parser:
	p := parser.NewParser(
		// parser.WithBlockParsers(util.Prioritized(parser.NewFencedCodeBlockParser(), 700)),
		parser.WithBlockParsers(util.Prioritized(&fenced{}, 700)),
	)

	// Make a default new markdown renderer:
	md := goldmark.New(
		goldmark.WithParser(p),
	)

	var tests = []struct {
		md, html, name string
	}{
		{_fencedInline, _fencedInlineHTML, "inline"},
		{_fencedLanguage, _fencedLanguageHTML, "language"},
		{_fencedBroken, _fencedBrokenHTML, "broken"},
	}

	// Results
	var buf bytes.Buffer

	for _, test := range tests {
		if err := md.Convert([]byte(test.md), &buf); err != nil {
			t.Fatal("Failed to parse fenced "+test.name+":", err)
		}

		strcmp(t, "fenced "+test.name, buf.String(), test.html)
		buf.Reset()
	}
}
