package md

import (
	"bytes"
	"regexp"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

type match struct {
	from, to int32
	str      []byte
}

var statePool = sync.Pool{
	New: func() interface{} {
		return &mdState{
			fmtter: &Formatter{},
			buffer: &bytes.Buffer{},
		}
	},
}

type mdState struct {
	last    int32
	output  []byte
	chunk   []byte
	prev    []byte
	matches [][]match
	context [][]byte

	fmtter *Formatter
	buffer *bytes.Buffer
}

var (
	boldItalics = [2][]byte{[]byte(`<span font_style="italic" font_weight="bold">`), []byte("</span>")}
	codeSpan    = [2][]byte{[]byte(`<span font_family="monospace">`), []byte("</span>")}
	quoteSpan   = [2][]byte{[]byte(`<span color="#789922">`), []byte("</span>")}
	newLine     = []byte("\n")
	tagsList    = [...][3][]byte{
		[3][]byte{[]byte("*"), []byte("<i>"), []byte("</i>")},
		[3][]byte{[]byte("_"), []byte("<i>"), []byte("</i>")},
		[3][]byte{[]byte("**"), []byte("<b>"), []byte("</b>")},
		[3][]byte{[]byte("__"), []byte("<u>"), []byte("</u>")},
		[3][]byte{[]byte("***"), boldItalics[0], boldItalics[1]},
		[3][]byte{[]byte("~~"), []byte("<s>"), []byte("</s>")},
		[3][]byte{[]byte("||"), []byte(`<span fgcolor="#808080>"`), []byte("</span>")},
	}
	escapes = [...][]byte{
		[]byte("&amp;"),
		[]byte("&#39;"),
		[]byte("&lt;"),
		[]byte("&gt;"),
	}
)

func (s *mdState) tag(token []byte) []byte {
	var tags [3][]byte

	for _, t := range tagsList {
		if bytecmp(t[0], token) {
			tags = t
		}
	}

	if tags[0] == nil {
		return token
	}

	var index = -1
	for i, t := range s.context {
		if bytecmp(t, token) {
			index = i
			break
		}
	}

	if index >= 0 { // len(context) > 0 always
		s.context = append(s.context[:index], s.context[index+1:]...)
		return tags[1]
	} else {
		s.context = append(s.context, token)
		return tags[0]
	}
}

func (s *mdState) getLastIndex(currentIndex int) int32 {
	if currentIndex >= len(s.matches) {
		return 0
	}

	return s.matches[currentIndex][0].to
}

func (s *mdState) submatch(r *regexp.Regexp, input []byte) {
	found := r.FindAllSubmatchIndex(input, -1)

	s.last = 0
	s.output = s.output[:0]
	s.chunk = s.chunk[:0]
	s.prev = s.prev[:0]
	s.context = s.context[:0]
	s.fmtter.Reset()

	// We're not clearing s.matches

	var m = match{-1, -1, nil}
	var matchesList = s.matches[:0]

	for i := 0; i < len(found); i++ {
		var matches []match

		if i < len(s.matches) {
			matches = s.matches[i][:0]
		} else {
			matches = make([]match, 0, len(found[i])/2)
		}

		for a, b := range found[i] {
			if a%2 == 0 { // first pair
				m.from = int32(b)
			} else {
				m.to = int32(b)

				if m.from >= 0 && m.to >= 0 {
					m.str = input[m.from:m.to]
				} else {
					m.str = nil
				}

				matches = append(matches, m)
			}
		}

		matchesList = append(matchesList, matches)
	}

	s.matches = matchesList
}

func (s *mdState) renderCodeBlock(lang, content []byte) []byte {
	if style == nil {
		style = styles.Get(HighlightStyle)
		if style == nil {
			panic("Unknown highlighting style: " + HighlightStyle)
		}

		css = styleToCSS(style)
	}

	var lexer = lexers.Fallback

	if len(lang) > 0 {
		lang := string(lang)

		v, ok := lexerMap.Load(lang)
		if ok {
			lexer = v.(chroma.Lexer)
		} else {
			lexer = lexers.Get(lang)
			lexerMap.Store(lang, lexer)
		}

	} else {
		content = bytes.Join([][]byte{lang, content}, newLine)
	}

	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return content
	}

	s.buffer.Reset()

	if err := fmtter.Format(s.buffer, iterator); err != nil {
		return content
	}

	return s.buffer.Bytes()
}

func bytecmp(b1, b2 []byte) bool {
	if len(b1) != len(b2) {
		return false
	}

	for i := 0; i < len(b1); i++ {
		if b1[i] != b2[i] {
			return false
		}
	}
	return true
}
