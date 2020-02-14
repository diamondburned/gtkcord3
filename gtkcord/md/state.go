package md

import (
	"bytes"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/diamondburned/arikawa/discord"
	"github.com/gotk3/gotk3/gtk"
)

type match struct {
	from, to int32
	str      []byte
}

func newPool(p *Parser) sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			return &mdState{
				p:      p,
				fmtter: &Formatter{},
				buffer: &bytes.Buffer{},
			}
		},
	}
}

type mdState struct {
	p *Parser

	// We need to lock the iterators so they can't be invalidated while we're
	// using them.
	iterMu sync.Mutex
	iterWg sync.WaitGroup

	buf *gtk.TextBuffer

	last    int32
	prev    []byte
	matches [][]match

	tag   *gtk.TextTag
	color string
	attr  Attribute

	// Used to determine whether or not emojis should be large:
	hasText bool

	fmtter *Formatter
	buffer *bytes.Buffer
}

func (s *mdState) tagAttr(token []byte) []byte {
	attr := TagAttribute(token)
	if attr == 0 {
		return token
	}

	if s.attr.Has(attr) {
		s.tagRemove(attr)
		return nil
	}

	// If the current token starts inline code, we don't want anything else.
	if attr == AttrMonospace {
		s.tagReset()
	}
	// If the state is already an inline code, we don't want any markup. Treat
	// tokens as plain text.
	if s.attr.Has(AttrMonospace) {
		return token
	}

	s.tagAdd(attr)
	return nil
}

func (s *mdState) switchTree(i int) {
	switch {
	case len(s.matches[i][2].str) > 0:
		code := string(s.renderCodeBlock(
			s.matches[i][1].str,
			s.matches[i][2].str,
		))

		s.buf.InsertMarkup(s.buf.GetEndIter(), code)

	case len(s.matches[i][3].str) > 0:
		// blockquotes, greentext
		s.insertWithTag(append(s.matches[i][3].str, '\n'), s.tagWith(AttrQuoted))

	case len(s.matches[i][4].str) > 0:
		// inline stuff
		if token := s.tagAttr(s.matches[i][4].str); token != nil {
			s.insertWithTag(token, nil)
		}

	case len(s.matches[i][5].str) > 0:
		// TODO URLs
		s.insertWithTag(s.matches[i][5].str, s.tagWithColor("cyan"))

	case len(s.matches[i][9].str) > 0:
		// Emojis
		var animated = len(s.matches[i][10].str) > 0
		s.InsertAsyncPixbuf(EmojiURL(string(s.matches[i][11].str), animated))

	case bytes.Count(s.prev, []byte(`\`))%2 != 0:
		// Escaped:
		fallthrough
	default:
		s.insertWithTag(s.matches[i][0].str, nil)
	}
}

func (s *mdState) switchTreeMessage(m *discord.Message) func(i int) {
	return func(i int) {
		switch {
		case len(s.matches[i][6].str) > 0:
			// user mentions
			s.insertWithTag(s.matches[i][6].str, nil)
			// s.chunk = UserNicknameHTML(d, m, s.matches[i][7].str)

		case len(s.matches[i][7].str) > 0:
			// role mentions
			s.insertWithTag(s.matches[i][7].str, nil)
			// s.chunk = RoleNameHTML(d, m, s.matches[i][8].str)

		case len(s.matches[i][8].str) > 0:
			// channel mentions
			s.insertWithTag(s.matches[i][8].str, nil)
			// s.chunk = ChannelNameHTML(d, m, s.matches[i][9].str)

		default:
			s.switchTree(i)
		}
	}
}

func (s *mdState) insertWithTag(content []byte, tag *gtk.TextTag) {
	if tag == nil {
		tag = s.tag
	}
	s.buf.InsertWithTag(s.buf.GetEndIter(), string(content), tag)
}

func (s *mdState) getLastIndex(currentIndex int) int32 {
	if currentIndex >= len(s.matches) {
		return 0
	}

	return s.matches[currentIndex][0].to
}

func (s *mdState) use(buf *gtk.TextBuffer, input []byte) {
	found := regex.FindAllSubmatchIndex(input, -1)

	s.buf = buf
	s.last = 0
	s.prev = s.prev[:0]
	s.hasText = false
	s.attr = 0
	s.color = ""
	s.tag = s.p.ColorTag(s.attr, s.color)
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
			for i := 1; i < len(matches) && i < 9; i++ {
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
		content = bytes.Join([][]byte{lang, content}, []byte("\n"))
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
