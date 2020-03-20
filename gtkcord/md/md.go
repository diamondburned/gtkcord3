package md

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func Parse(content []byte, dst *gtk.TextBuffer) error {
	p := parser.NewParser(
		parser.WithBlockParsers(BlockParsers()...),
		parser.WithInlineParsers(InlineParsers()...),
	)

	r := NewRenderer(dst, nil)
	node := p.Parse(text.NewReader(content))
	return r.Render(nil, content, node)
}
