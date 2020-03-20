package md

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type blockquote struct{}

// process the line
func (b blockquote) process(reader text.Reader) bool {
	line, _ := reader.PeekLine()
	w, pos := util.IndentWidth(line, reader.LineOffset())

	// If line doesn't start with >
	if w > 3 || pos >= len(line) || line[pos] != '>' {
		return false
	}

	pos++

	// What the fuck is this?
	if pos >= len(line) || line[pos] == '\n' {
		reader.Advance(pos)
		return true
	}

	// Valid behavior: >(space)Thing
	if util.IsSpace(line[pos]) {
		pos++
	}

	reader.Advance(pos)
	return true
}

func (b blockquote) Trigger() []byte {
	return []byte{'>'}
}

func (b blockquote) Open(p ast.Node, r text.Reader, pc parser.Context) (ast.Node, parser.State) {
	if b.process(r) {
		_, seg := r.PeekLine()

		node := ast.NewBlockquote()

		para := ast.NewParagraph()
		para.Lines().Append(seg)

		node.AppendChild(node, para)

		return node, parser.NoChildren
	}

	return nil, parser.NoChildren
}

func (b blockquote) Continue(node ast.Node, r text.Reader, pc parser.Context) parser.State {
	if b.process(r) {
		_, seg := r.PeekLine()

		para := node.FirstChild().(*ast.Paragraph)
		para.Lines().Append(seg)

		return parser.Continue
	}

	return parser.Close
}

func (b blockquote) Close(node ast.Node, r text.Reader, pc parser.Context) {
	para := node.FirstChild().(*ast.Paragraph)

	lines := para.Lines()

	if length := lines.Len(); length > 0 {
		// Trim last whitespace
		last := lines.At(length - 1)
		lines.Set(length-1, last.TrimRightSpace(r.Source()))
	}
}

func (b blockquote) CanInterruptParagraph() bool {
	return true
}

func (b blockquote) CanAcceptIndentedLine() bool {
	return false
}
