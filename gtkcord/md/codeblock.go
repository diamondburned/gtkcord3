package md

import (
	"errors"
	"strings"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/diamondburned/ningen/v2/md"
	"github.com/yuin/goldmark/ast"
)

var (
	lexerMap = map[string]chroma.Lexer{}
	lexerMut = sync.Mutex{}

	fmtter = Formatter{}

	css      = map[chroma.TokenType]Tag{}
	styleMut = sync.RWMutex{}
)

func ChangeStyle(styleName string) error {
	s := styles.Get(styleName)

	// styleName == "" => no highlighting, not an error
	if s == styles.Fallback && styleName != "" {
		return errors.New("Unknown style")
	}

	styleMut.Lock()
	defer styleMut.Unlock()

	css = styleToCSS(s)
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
	}

	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		code = append(code, line.Value(source)...)
	}

	iterator, err := lexer.Tokenise(nil, string(code))
	if err != nil {
		// Write the raw code block without any highlighting:
		r.insertWithTag(source, r.tags.colorTag(Tag{
			Attr: md.AttrMonospace,
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
		Attr: md.AttrMonospace,
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
	var attr = md.AttrMonospace
	var color string

	if e.Colour.IsSet() {
		color = e.Colour.String()
	}
	if e.Bold == chroma.Yes {
		attr |= md.AttrBold
	}
	if e.Italic == chroma.Yes {
		attr |= md.AttrItalics
	}
	if e.Underline == chroma.Yes {
		attr |= md.AttrUnderline
	}

	return Tag{attr, color}
}
