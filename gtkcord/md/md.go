package md

import (
	"bytes"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/gotk3/gotk3/gtk"
)

var regexes = []string{
	// codeblock
	`(?:^\x60\x60\x60 *(\w*)([\s\S]*?)\n?\x60\x60\x60$)`,
	// blockquote
	`((?:(?:^|\n)^>\s+.*)+)\n`,
	// Bullet points, but there's no capture group (disabled)
	`(?:(?:^|\n)(?:[>*+-]|\d+\.)\s+.*)+`,
	// This is actually inline code
	`(?:\x60([^\x60].*?)\x60)`,
	// Inline markup stuff
	`(__|\*\*\*|\*\*|[_*]|~~|\|\|)`,
	// Hyperlinks
	`(https?:\/\S+(?:\.|:)\S+)`,
	// User mentions
	`(?:<@!?(\d+)>)`,
	// Role mentions
	`(?:<@&(\d+)>)`,
	// Channel mentions
	`(?:<#(\d+)>)`,
	// Emojis
	`(<(a?):\w+:(\d+)>)`,
}

var HighlightStyle = "solarized-dark"

var (
	style    = (*chroma.Style)(nil)
	regex    = regexp.MustCompile(`(?m)` + strings.Join(regexes, "|"))
	fmtter   = Formatter{}
	css      = map[chroma.TokenType]string{}
	lexerMap = sync.Map{}
)

type Parser struct {
	pool  sync.Pool
	State *state.State

	ChannelPressed func(id discord.Snowflake)
	UserPressed    func(id discord.Snowflake)
	RolePressed    func(id discord.Snowflake)
	URLPressed     func(url string)

	Error func(err error)

	theme *gtk.IconTheme
}

func NewParser(s *state.State) *Parser {
	i, err := gtk.IconThemeGetDefault()
	if err != nil {
		// We can panic here, as nothing would work if this ever panics.
		panic("Can't get GTK Icon Theme: " + err.Error())
	}

	p := &Parser{
		State: s,
		theme: i,
		Error: func(err error) {
			log.Println("Markdown:", err)
		},
	}
	p.pool = newPool(p)

	return p
}

func (p *Parser) Parse(md []byte, buf *gtk.TextBuffer) {
	p.ParseMessage(nil, md, buf)
}

func (p *Parser) ParseMessage(m *discord.Message, md []byte, buf *gtk.TextBuffer) {
	s := p.pool.Get().(*mdState)
	defer func() {
		go func() {
			s.iterWg.Wait()
			p.pool.Put(s)
		}()
	}()

	s.submatch(regex, md)

	var tree func(i int)
	if s == nil || m == nil {
		tree = s.switchTree(buf)
	} else {
		tree = s.switchTreeMessage(buf, m)
	}

	for i := 0; i < len(s.matches); i++ {
		s.prev = md[s.last:s.matches[i][0].from]
		s.last = s.getLastIndex(i)
		s.chunk = s.chunk[:0] // reset chunk

		tree(i)

		if b := append(escape(s.prev), s.chunk...); len(b) > 0 {
			s.iterMu.Lock()
			buf.InsertMarkup(buf.GetEndIter(), string(b))
			s.iterMu.Unlock()
		}
	}

	s.iterMu.Lock()
	defer s.iterMu.Unlock()

	iter := buf.GetEndIter()

	buf.InsertMarkup(iter, string(md[s.last:]))

	// Flush:
	for len(s.context) > 0 {
		buf.InsertMarkup(iter, string(s.tag(s.context[len(s.context)-1])))
	}
}

func renderBlockquote(body []byte) []byte {
	return bytes.Join([][]byte{quoteSpan[0], escape(body), quoteSpan[1]}, nil)
}

func escape(thing []byte) []byte {
	if len(thing) == 0 {
		return nil
	}

	// escaped := thing[:0]
	escaped := make([]byte, 0, len(thing))

	for i := 0; i < len(thing); i++ {
		switch thing[i] {
		case '&':
			escaped = append(escaped, escapes[0]...)
		case '\'':
			escaped = append(escaped, escapes[1]...)
		case '<':
			escaped = append(escaped, escapes[2]...)
		case '>':
			escaped = append(escaped, escapes[3]...)
		default:
			escaped = append(escaped, thing[i])
		}
	}

	return escaped
}
