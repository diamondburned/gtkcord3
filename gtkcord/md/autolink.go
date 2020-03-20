package md

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type autolink struct{}

func (s autolink) Trigger() []byte {
	// return []byte("http")
	return []byte{' '}
}

func (s autolink) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()

	stop := util.FindURLIndex(line)
	typ := ast.AutoLinkURL

	if stop < 0 {
		return nil
	}

	value := ast.NewTextSegment(text.NewSegment(segment.Start, segment.Start+stop))
	block.Advance(stop + 1)
	return ast.NewAutoLink(typ, value)
}
