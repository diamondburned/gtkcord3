package md

import (
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
	// This is actually inline code
	// `(?:\x60([^\x60].*?)\x60)`,
	// Inline markup stuff
	`(__|\x60|\*\*\*|\*\*|[_*]|~~|\|\|)`,
	// Hyperlinks
	`<?(https?:\/\S+(?:\.|:)[^>\s]+)>?`,
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
	log.Println("Regex", strings.Join(regexes, "|"))
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

	s.state.Use(buf)
	s.use(buf, md)

	var tree func(i int)
	if s == nil || m == nil {
		tree = s.switchTree
	} else {
		tree = s.switchTreeMessage(m)
	}

	s.iterMu.Lock()

	for i := 0; i < len(s.matches); i++ {
		s.prev = md[s.last:s.matches[i][0].from]
		s.last = s.getLastIndex(i)

		s.insertWithTag(s.prev, nil)
		tree(i)
	}

	s.insertWithTag(md[s.last:], nil)

	s.iterMu.Unlock()

	go func() {
		s.iterWg.Wait()
		s.buf = nil
		p.pool.Put(s)
	}()
}
