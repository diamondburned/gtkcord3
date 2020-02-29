package md

import (
	"bytes"
	"sync"
	"time"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
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

	m *discord.Message
	d Discord

	// We need to lock the iterators so they can't be invalidated while we're
	// using them.
	iterMu sync.Mutex
	iterWg sync.WaitGroup

	buf *gtk.TextBuffer
	ttt *gtk.TextTagTable

	last    int32
	prev    []byte
	matches [][]match
	used    []int

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
	if bytes.Count(s.prev, []byte(`\`))%2 != 0 {
		s.insertWithTag(s.matches[i][0].str, nil)
		return
	}

	switch {
	case len(s.matches[i][1].str) > 0, len(s.matches[i][2].str) > 0:
		code := string(s.renderCodeBlock(
			s.matches[i][1].str,
			s.matches[i][2].str,
		))

		if i == 0 {
			code = code[1:] // trim trailing newline
		}

		semaphore.IdleMust(func() {
			s.buf.InsertMarkup(s.buf.GetEndIter(), code)
		})

	case len(s.matches[i][3].str) > 0:
		// blockquotes, greentext
		s.insertWithTag(append(s.matches[i][3].str, '\n'), s.tagWith(AttrQuoted))

	case len(s.matches[i][4].str) > 0:
		// inline stuff
		if token := s.tagAttr(s.matches[i][4].str); token != nil {
			s.insertWithTag(token, nil)
		}

	case len(s.matches[i][5].str) > 0:
		s.insertWithTag(
			s.matches[i][5].str,
			s.Hyperlink(string(s.matches[i][5].str)),
		)

	case len(s.matches[i][9].str) > 0:
		// Emojis
		var animated = len(s.matches[i][10].str) > 0
		s.InsertAsyncPixbuf(EmojiURL(string(s.matches[i][11].str), animated))

	default:
		s.insertWithTag(s.matches[i][0].str, nil)
	}
}

func (s *mdState) switchTreeMessage(i int) {
	switch {
	case len(s.matches[i][6].str) > 0:
		// user mentions
		s.InsertUserMention(s.matches[i][6].str)

	case len(s.matches[i][7].str) > 0:
		// role mentions
		s.insertWithTag(s.matches[i][7].str, nil)
		// s.chunk = RoleNameHTML(d, m, s.matches[i][8].str)

	case len(s.matches[i][8].str) > 0:
		// channel mentions
		s.InsertChannelMention(s.matches[i][8].str)

	default:
		s.switchTree(i)
	}
}

func (s *mdState) addEditedStamp(date time.Time) {
	semaphore.IdleMust(func() {
		v, err := s.ttt.Lookup("timestamp")
		if err != nil {
			v, err = gtk.TextTagNew("timestamp")
			if err != nil {
				log.Panicln("Failed to create a new timestamp tag:", err)
			}

			v.SetProperty("scale", 0.84)
			v.SetProperty("scale-set", true)
			v.SetProperty("foreground", "#808080")

			s.ttt.Add(v)
		}

		edited := "  (edited " + humanize.TimeAgo(date) + ")"
		s.buf.InsertWithTag(s.buf.GetEndIter(), edited, v)
	})
}

func (s *mdState) insertWithTag(content []byte, tag *gtk.TextTag) {
	if tag == nil {
		tag = s.tag
	}

	semaphore.IdleMust(func() {
		s.buf.InsertWithTag(s.buf.GetEndIter(), string(content), tag)
	})
}

func (s *mdState) getLastIndex(currentIndex int) int32 {
	if currentIndex >= len(s.matches) {
		return 0
	}

	return s.matches[currentIndex][0].to
}

func (s *mdState) use(buf *gtk.TextBuffer, input []byte, d Discord, msg *discord.Message) {
	found := regex.FindAllSubmatchIndex(input, -1)

	s.buf = buf
	s.tagTable()

	s.d = d
	s.m = msg

	s.tag = s.ColorTag(s.attr, s.color)
	s.fmtter.Reset()

	// We're not clearing s.matches

	var m = match{-1, -1, nil}
	var matchesList = s.matches[:0]

	// used for emoji / hasText
	var last int32 = 0

	// used for optimization
	var index = 0

	var ok bool

	for i := 0; i < len(found); i++ {
		// If the match is an inline markup symbol:
		if found[i][4*2] > -1 {
			// If the pair isn't already matched with a pair prior, and
			// if we could not find a next matching pair:
			if s.used, ok = findPairs(found, i, 4, s.used); !ok {
				continue
			}
		}

		var matches []match

		if index < len(s.matches) {
			matches = s.matches[index][:0]
			index++
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

		if !s.hasText {
			if i := len(matchesList) - 1; i > 0 {
				last = matchesList[i-1][0].to
			}

			if len(input[last:matches[0].from]) > 0 {
				s.hasText = true
			}
		}

		matchesList = append(matchesList, matches)
	}

	s.matches = matchesList
}

func findPairs(found [][]int, start, match int, used []int) ([]int, bool) {
	for _, u := range used {
		if u == start {
			// seen
			return used, true
		}
	}

	match *= 2
	start += 1

	for j := start; j < len(found); j++ {
		if found[j][match] > -1 {
			used = append(used, j)
			return used, true
		}
	}

	return used, false
}

func (s *mdState) renderCodeBlock(lang, content []byte) []byte {
	var lexer chroma.Lexer

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
	}

	if lexer == nil {
		lexer = lexers.Fallback
		content = append(lang, content...)
	}

	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return content
	}

	s.buffer.Reset()
	s.buffer.WriteByte('\n')
	if err := fmtter.Format(s.buffer, iterator); err != nil {
		return content
	}
	s.buffer.WriteByte('\n')

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
