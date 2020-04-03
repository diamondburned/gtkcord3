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
	// if pos >= len(line) || line[pos] == '\n' {
	// 	reader.Advance(pos)
	// 	return true
	// }

	// Invalid behavior: >Thing
	if pos < len(line) && !util.IsSpace(line[pos]) {
		return false
	}

	reader.Advance(pos + 1)

	return true
}

func (b blockquote) Trigger() []byte {
	return []byte{'>'}
}

func (b blockquote) Open(p ast.Node, r text.Reader, pc parser.Context) (ast.Node, parser.State) {
	if b.process(r) {
		node := ast.NewBlockquote()

		// Try and parse the block as a paragraph:
		para, state := _paragraph.Open(node, r, pc)

		// If there's no paragraph, make a blank one:
		if para == nil {
			para = ast.NewParagraph()
		}

		// Add and return the paragraph anyway, maybe the first line is just empty.
		node.AppendChild(node, para)
		return node, state
	}

	return nil, parser.NoChildren
}

func (b blockquote) Continue(node ast.Node, r text.Reader, pc parser.Context) parser.State {
	if b.process(r) {
		para := node.FirstChild().(*ast.Paragraph)
		return _paragraph.Continue(para, r, pc)
	}

	// TODO: bug
	// This would not render:
	//
	//    Seriously.
	//    > be __discord__
	//    > die
	//    > tfw
	//
	//    	asdasdasdasdasdas
	//
	//    yup. ***lolmao*** ` + "`" + `echo "yeet $HOME"` + "`" + `
	//

	return parser.Close
}

func (b blockquote) Close(node ast.Node, r text.Reader, pc parser.Context) {
	para, ok := node.FirstChild().(*ast.Paragraph)
	if !ok { // if not a paragraph:
		return
	}

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
