package md

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var fencedCodeBlockInfoKey = parser.NewContextKey()

type fenced struct{}
type fenceData struct {
	indent int
	length int
	node   ast.Node
}

func (b fenced) Trigger() []byte {
	return []byte{'`'}
}

func (b fenced) Open(p ast.Node, r text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := r.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || (line[pos] != '`') {
		return nil, parser.NoChildren
	}

	findent := pos
	i := pos

	for ; i < len(line) && line[i] == '`'; i++ {
	}

	oFenceLength := i - pos

	// If there are less than 3 backticks:
	if oFenceLength < 3 {
		return nil, parser.NoChildren
	}

	var node = ast.NewFencedCodeBlock(nil)

	// If this isn't the last thing in the line: (```<language>)
	if i < len(line)-1 {
		rest := line[i:]
		left := util.TrimLeftSpaceLength(rest)
		right := util.TrimRightSpaceLength(rest)

		// If not white-space?
		if left < len(rest)-right {
			infoStart, infoStop := segment.Start-segment.Padding+i+left, segment.Stop-right
			if infoStart != infoStop {
				var value = rest[left : len(rest)-right]
				var seg = text.NewSegment(infoStart, infoStop)

				switch {
				case bytes.HasSuffix(value, []byte("```")):
					// Single line code:
					seg.Stop -= 3 // len("```")
					node.Lines().Append(seg)

				case bytes.IndexByte(value, ' ') == -1:
					// If value does not contain spaces, it's probably the language
					// part.
					node.Info = ast.NewTextSegment(seg)

				default:
					// Invalid codeblock, but we're parsing it anyway. It will
					// just render the entire thing as a codeblock according to
					// CommonMark specs.

					node.Lines().Append(seg)
				}
			}
		}
	}

	pc.Set(fencedCodeBlockInfoKey, &fenceData{findent, oFenceLength, node})

	return node, parser.NoChildren
}

func (b fenced) Continue(node ast.Node, r text.Reader, pc parser.Context) parser.State {
	line, segment := r.PeekLine()
	fdata := pc.Get(fencedCodeBlockInfoKey).(*fenceData)
	_, pos := util.IndentWidth(line, r.LineOffset())

	// Crawl i to ```
	i := pos
	for ; i < len(line) && line[i] != '`'; i++ {
	}

	// Is there a string literal? Write it.
	pos, padding := util.DedentPositionPadding(line, r.LineOffset(), segment.Padding, fdata.indent)

	// If there is an end block:
	if i != len(line) {
		segment.Stop -= 3
	}

	seg := text.NewSegmentPadding(segment.Start+pos, segment.Stop, padding)
	node.Lines().Append(seg)
	r.AdvanceAndSetPadding(segment.Stop-segment.Start-pos-2, padding)

	// If found:
	if i != len(line) {
		for ; i < len(line) && line[i] == '`'; i++ {
		}

		length := i - pos
		if length >= fdata.length && util.IsBlank(line[i:]) {
			var newline = 1
			if line[len(line)-1] != '\n' {
				newline = 0
			}

			r.Advance(segment.Stop - segment.Start - newline - segment.Padding)
			return parser.Close
		}
	}

	return parser.Continue | parser.NoChildren
}

func (b fenced) Close(node ast.Node, r text.Reader, pc parser.Context) {
	fdata := pc.Get(fencedCodeBlockInfoKey).(*fenceData)
	if fdata.node == node {
		pc.Set(fencedCodeBlockInfoKey, nil)
	}
}

func (b fenced) CanInterruptParagraph() bool {
	return true
}

func (b fenced) CanAcceptIndentedLine() bool {
	return false
}
