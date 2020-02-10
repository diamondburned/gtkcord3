package md

import (
	"bytes"
	"regexp"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/diamondburned/arikawa/discord"
	"github.com/gotk3/gotk3/gtk"
)

func newPool(p *Parser) sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			return newState(p)
		},
	}
}

type match struct {
	from, to int32
	str      []byte
}

type mdState struct {
	p *Parser

	// We need to lock the iterators so they can't be invalidated while we're
	// using them.
	iterMu sync.Mutex
	iterWg sync.WaitGroup

	last    int32
	chunk   []byte
	prev    []byte
	matches [][]match
	context [][]byte

	// Used to determine whether or not emojis should be large:
	hasText bool

	fmtter *Formatter
	buffer *bytes.Buffer
}

func newState(p *Parser) *mdState {
	return &mdState{
		p:      p,
		fmtter: &Formatter{},
		buffer: &bytes.Buffer{},
	}
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
		return tags[2]
	} else {
		s.context = append(s.context, token)
		return tags[1]
	}
}

func (s *mdState) switchTree(buf *gtk.TextBuffer) func(i int) {
	return func(i int) {
		switch {
		case len(s.matches[i][2].str) > 0:
			// codeblock
			s.chunk = s.renderCodeBlock(
				s.matches[i][1].str,
				s.matches[i][2].str,
			)
		case len(s.matches[i][3].str) > 0:
			// blockquotes, greentext
			s.chunk = renderBlockquote(s.matches[i][3].str)
		case len(s.matches[i][4].str) > 0:
			// inline code
			s.chunk = bytes.Join([][]byte{codeSpan[0], s.matches[i][4].str, codeSpan[1]}, nil)
		case len(s.matches[i][5].str) > 0:
			// inline stuff
			s.chunk = s.tag(s.matches[i][5].str)
		case len(s.matches[i][6].str) > 0:
			// URLs
			s.chunk = s.matches[i][6].str
		case len(s.matches[i][10].str) > 0:
			// Emojis
			var animated = len(s.matches[i][11].str) > 0

			if err := s.InsertAsyncPixbuf(buf,
				EmojiURL(string(s.matches[i][12].str), animated)); err != nil {

				s.chunk = s.matches[i][10].str
			}

		case bytes.Count(s.prev, []byte(`\`))%2 != 0:
			// escaped, print raw
			s.chunk = escape(s.matches[i][0].str)
		default:
			s.chunk = escape(s.matches[i][0].str)
		}
	}
}

func (s *mdState) switchTreeMessage(buf *gtk.TextBuffer, m *discord.Message) func(i int) {
	normal := s.switchTree(buf)

	return func(i int) {
		switch {
		case len(s.matches[i][7].str) > 0:
			// user mentions
			// s.chunk = UserNicknameHTML(d, m, s.matches[i][7].str)
		case len(s.matches[i][8].str) > 0:
			// role mentions
			// s.chunk = RoleNameHTML(d, m, s.matches[i][8].str)
		case len(s.matches[i][9].str) > 0:
			// channel mentions
			// s.chunk = ChannelNameHTML(d, m, s.matches[i][9].str)
		default:
			normal(i)
		}
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

		// If we still don't know if there are texts:
		if !s.hasText {
			// We know 1-10 are not emojis:
			for i := 1; i < len(matches) && i < 10; i++ {
				if len(matches[i].str) > 0 && matches[i].str[0] != '\n' {
					s.hasText = true
				}
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
			if l := lexers.Get(lang); l != nil {
				lexer = l
				lexerMap.Store(lang, lexer)
			}
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
