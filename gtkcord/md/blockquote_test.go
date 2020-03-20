package md

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

const _blockquote = `lol message
test
asdasd
dadadasd
asdsad
> be me
> discord markdown **bad**
> suffer
discord sucks.`
const _blockquoteHTML = `<p>lol message
test
asdasd
dadadasd
asdsad</p>
<blockquote>
<p>be me
discord markdown <strong>bad</strong>
suffer</p>
</blockquote>
<p>discord sucks.</p>
`

func TestBlockquote(t *testing.T) {
	// Make a fenced only parser:
	p := parser.NewParser(
		parser.WithBlockParsers(
			util.Prioritized(blockquote{}, 700),
			util.Prioritized(parser.NewParagraphParser(), 500),
		),
		parser.WithInlineParsers(
			util.Prioritized(parser.NewEmphasisParser(), 200),
		),
		// parser.WithBlockParsers(util.Prioritized(blockquote{}, 700)),
	)

	// Make a default new markdown renderer:
	md := goldmark.New(
		goldmark.WithParser(p),
	)

	// Results
	var buf bytes.Buffer

	// Test inline:
	if err := md.Convert([]byte(_blockquote), &buf); err != nil {
		t.Fatal("Failed to parse fenced inline:", err)
	}

	strcmp(t, "blockquote", buf.String(), _blockquoteHTML)
}
