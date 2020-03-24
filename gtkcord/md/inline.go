package md

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type Inline struct {
	ast.BaseInline
	Attr Attribute
}

var KindInline = ast.NewNodeKind("Inline")

// Kind implements Node.Kind.
func (e *Inline) Kind() ast.NodeKind {
	return KindInline
}

// Dump implements Node.Dump
func (e *Inline) Dump(source []byte, level int) {
	ast.DumpHelper(e, source, level, nil, nil)
}

var inlineTriggers = []byte{'*', '_', '|', '~', '`'}

type inlineDelimiterProcessor struct {
	char byte
}

func (p *inlineDelimiterProcessor) IsDelimiter(b byte) bool {
	for _, t := range inlineTriggers {
		if t == b {
			p.char = b
			return true
		}
	}

	return false
}

func (p *inlineDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *inlineDelimiterProcessor) OnMatch(consumes int) ast.Node {
	var node = &Inline{
		BaseInline: ast.BaseInline{},
	}
	switch {
	case p.char == '_' && consumes == 2:
		node.Attr = AttrUnderline
	case p.char == '_' && consumes == 1:
		fallthrough
	case p.char == '*' && consumes == 1:
		node.Attr = AttrItalics
	case p.char == '*' && consumes == 2:
		node.Attr = AttrBold
	case p.char == '|' && consumes == 2:
		node.Attr = AttrSpoiler
	case p.char == '~' && consumes == 2:
		node.Attr = AttrStrikethrough
	case p.char == '`' && consumes == 1:
		node.Attr = AttrMonospace
	}

	return node
}

type inline struct{}

func (inline) Trigger() []byte {
	return inlineTriggers
}

func (inline) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()

	node := parser.ScanDelimiter(line, before, 1, &inlineDelimiterProcessor{})
	if node == nil {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)

	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)

	return node
}
