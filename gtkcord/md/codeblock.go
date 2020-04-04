package md

import (
	"bytes"
	"errors"
	"strings"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var (
	lexerMap = map[string]chroma.Lexer{}
	lexerMut = sync.Mutex{}

	fmtter = Formatter{}

	css      = map[chroma.TokenType]Tag{}
	style    = ""
	styleMut = sync.RWMutex{}
)

func ChangeStyle(styleName string) error {
	s := styles.Get(styleName)

	styleMut.Lock()
	defer styleMut.Unlock()

	css = styleToCSS(s)

	if s == styles.Fallback {
		return errors.New("Unknown style")
	}
	return nil
}

func getLexer(_lang []byte) chroma.Lexer {
	if _lang == nil {
		return nil
	}

	var lang = string(_lang)

	lexerMut.Lock()
	defer lexerMut.Unlock()

	v, ok := lexerMap[lang]
	if ok {
		return v
	}

	lexerMut.Unlock()
	v = lexers.Get(lang)
	lexerMut.Lock()

	if v != nil {
		lexerMap[lang] = v
		return v
	}

	return nil
}

func (r *Renderer) renderCodeBlock(node *ast.FencedCodeBlock, source []byte) {
	var lang = node.Language(source)
	var code = []byte{}

	var lexer = getLexer(lang)

	if lexer == nil {
		lexer = lexers.Fallback
		code = lang
	}

	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		// Prepend 2 spaces at the start of line to fake indentation:
		code = append(code, append([]byte{' ', ' '}, line.Value(source)...)...)
	}

	iterator, err := lexer.Tokenise(nil, string(code))
	if err != nil {
		// Write the raw code block without any highlighting:
		r.insertWithTag(source, r.tags.colorTag(Tag{
			Attr: AttrMonospace,
		}))

		return
	}

	fmtter.Format(r, iterator)
}

// Formatter that generates Pango markup.
type Formatter struct {
	highlightRanges [][2]int
}

func (f *Formatter) reset() {
	f.highlightRanges = f.highlightRanges[:0]
}

func (f *Formatter) Format(r *Renderer, iterator chroma.Iterator) {
	f.reset()

	tokens := iterator.Tokens()
	lines := chroma.SplitTokensIntoLines(tokens)
	highlightIndex := 0

	var empty = Tag{
		Attr: AttrMonospace,
	}

	var attr = empty

	highlightIndex = 0
	for index, tokens := range lines {
		// 1-based line number.
		line := 1 + index
		highlight, next := f.shouldHighlight(highlightIndex, line)
		if next {
			highlightIndex++
		}

		if highlight {
			attr = f.styleAttr(chroma.LineHighlight)
		}

		for _, token := range tokens {
			code := strings.Replace(token.String(), "\t", "    ", -1)
			attr := attr.Combine(f.styleAttr(token.Type))

			r.insertWithTag([]byte(code), r.tags.colorTag(attr))
		}

		if highlight {
			attr = empty
		}
	}
}

func (f *Formatter) shouldHighlight(highlightIndex, line int) (bool, bool) {
	var next = false

	for highlightIndex < len(f.highlightRanges) && line > f.highlightRanges[highlightIndex][1] {
		highlightIndex++
		next = true
	}

	if highlightIndex < len(f.highlightRanges) {
		hrange := f.highlightRanges[highlightIndex]

		if line >= hrange[0] && line <= hrange[1] {
			return true, next
		}
	}

	return false, next
}

func (f *Formatter) styleAttr(tt chroma.TokenType) Tag {
	styleMut.RLock()
	defer styleMut.RUnlock()

	if _, ok := css[tt]; !ok {
		tt = tt.SubCategory()
	}
	if _, ok := css[tt]; !ok {
		tt = tt.Category()
	}
	if t, ok := css[tt]; ok {
		return t
	}

	return EmptyTag
}

func styleToCSS(style *chroma.Style) map[chroma.TokenType]Tag {
	classes := map[chroma.TokenType]Tag{}
	bg := style.Get(chroma.Background)

	for t := range chroma.StandardTypes {
		var entry = style.Get(t)
		if t != chroma.Background {
			entry = entry.Sub(bg)
		}
		if entry.IsZero() {
			continue
		}
		classes[t] = styleEntryToTag(entry)
	}
	return classes
}

func styleEntryToTag(e chroma.StyleEntry) Tag {
	var attr = AttrMonospace
	var color string

	if e.Colour.IsSet() {
		color = e.Colour.String()
	}
	if e.Bold == chroma.Yes {
		attr |= AttrBold
	}
	if e.Italic == chroma.Yes {
		attr |= AttrItalics
	}
	if e.Underline == chroma.Yes {
		attr |= AttrUnderline
	}

	return Tag{attr, color}
}

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

func (b fenced) Parse(p ast.Node, r text.Reader, pc parser.Context) ast.Node {
	n, _ := b.Open(p, r, pc)
	if n == nil {
		return nil
	}

	// Crawl until b.Continue is done:
	for state := parser.Continue; state != parser.Close; state = b.Continue(n, r, pc) {
	}

	// Close:
	b.Close(n, r, pc)

	return n
}

func (b fenced) Open(p ast.Node, r text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := r.PeekLine()
	pos := 0 // TODO: figure out what pc.BlockOffset() is

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

		// If not white-space?
		if len(rest) > 0 {
			infoStart, infoStop := segment.Start-segment.Padding+i, segment.Stop
			if infoStart != infoStop {
				switch {
				case bytes.HasSuffix(rest, []byte("```")):
					// Single line code:
					seg := text.NewSegment(infoStart, infoStop)
					seg.Stop -= 3 // len("```")
					node.Lines().Append(seg)

				case bytes.IndexByte(bytes.TrimSpace(rest), ' ') == -1:
					// Account for the trailing whitespaces:
					left := util.TrimLeftSpaceLength(rest)
					right := util.TrimRightSpaceLength(rest)
					// If value does not contain spaces, it's probably the language
					// part.
					if left < right {
						node.Info = ast.NewTextSegment(
							text.NewSegment(infoStart+left, infoStop-right),
						)
					}

				default:
					// Invalid codeblock, but we're parsing it anyway. It will
					// just render the entire thing as a codeblock according to
					// CommonMark specs.
					node.Lines().Append(text.NewSegment(infoStart, infoStop))
				}
			}
		}
	}

	pc.Set(fencedCodeBlockInfoKey, &fenceData{findent, oFenceLength, node})
	r.Advance(segment.Len() - pos)

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

	// start+i accounts for everything before end (```)
	var start, stop = segment.Start + pos, segment.Start + i

	// Since we're assigning this segment a Start, IsEmpty() would fail if
	// seg.End is not touched.
	var seg = text.Segment{
		Start:   start,
		Padding: padding,
	}

	// If there's text:
	if start < stop {
		seg.Stop = stop
		r.AdvanceAndSetPadding(stop-start, padding)
	}

	defer func() {
		// Append this at the end of the function, as the block below might
		// reuse our text segment.
		if !seg.IsEmpty() {
			node.Lines().Append(seg)
		}
	}()

	// If found:
	if i != len(line) {
		// Update the starting position:
		pos = i

		// Iterate until we're out of backticks:
		for ; i < len(line) && line[i] == '`'; i++ {
		}

		// Do we have enough (3 or more) backticks?
		// If yes, end the codeblock properly.
		if length := i - pos; length >= fdata.length {
			r.Advance(length)
			return parser.Close
		} else {
			// No, treat the rest as text:
			seg.Stop = segment.Stop
			r.Advance(segment.Stop - stop)
		}
	}

	return parser.Continue | parser.NoChildren
}

func (b fenced) Close(node ast.Node, r text.Reader, pc parser.Context) {
	fdata := pc.Get(fencedCodeBlockInfoKey).(*fenceData)
	if fdata.node == node {
		pc.Set(fencedCodeBlockInfoKey, nil)
	}

	lines := node.Lines()

	if length := lines.Len(); length > 0 {
		// Trim first whitespace
		first := lines.At(0)
		lines.Set(0, first.TrimLeftSpace(r.Source()))

		// Trim last whitespace
		last := lines.At(length - 1)
		lines.Set(length-1, last.TrimRightSpace(r.Source()))
	}
}

func (b fenced) CanInterruptParagraph() bool {
	return true
}

func (b fenced) CanAcceptIndentedLine() bool {
	return false
}
