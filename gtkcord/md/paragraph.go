package md

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type paragraph struct{}

func (b paragraph) Trigger() []byte {
	return nil
}

func (b paragraph) Open(p ast.Node, r text.Reader, pc parser.Context) (ast.Node, parser.State) {
	_, segment := r.PeekLine()
	// segment = segment.TrimLeftSpace(r.Source())
	if segment.IsEmpty() {
		return nil, parser.NoChildren
	}

	node := ast.NewParagraph()
	node.Lines().Append(segment)
	r.Advance(segment.Len() - 1)

	return node, parser.NoChildren
}

func (b paragraph) Continue(node ast.Node, r text.Reader, pc parser.Context) parser.State {
	_, segment := r.PeekLine()
	// segment = segment.TrimLeftSpace(r.Source())
	if segment.IsEmpty() {
		return parser.Close
	}

	node.Lines().Append(segment)
	r.Advance(segment.Len() - 1)

	return parser.Continue | parser.NoChildren
}

func (b paragraph) Close(node ast.Node, r text.Reader, pc parser.Context) {
	p := node.Parent()
	if p == nil {
		// paragraph has been transformed
		return
	}

	lines := node.Lines()
	if lines.Len() != 0 {
		// trim trailing spaces
		length := lines.Len()
		lastLine := node.Lines().At(length - 1)
		node.Lines().Set(length-1, lastLine.TrimRightSpace(r.Source()))
	}

	if lines.Len() == 0 {
		node.Parent().RemoveChild(node.Parent(), node)
		return
	}
}

func (b paragraph) CanInterruptParagraph() bool {
	return false
}

func (b paragraph) CanAcceptIndentedLine() bool {
	return false
}
