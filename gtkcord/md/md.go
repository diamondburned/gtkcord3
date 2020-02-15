package md

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

var regexes = []string{
	// codeblock
	`(?:\n?\x60\x60\x60 *(\w*\n?)([\s\S]*?)\n?\x60\x60\x60\n?)`,
	// blockquote
	`((?:(?:^|\n)^>\s+.*)+)\n`,
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

var HighlightStyle = "monokai"

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

	table *gtk.TextTagTable
	theme *gtk.IconTheme
	icons sync.Map
}

func NewParser(s *state.State) *Parser {
	log.Debugln("REGEX:", strings.Join(regexes, "|"))

	i, err := gtk.IconThemeGetDefault()
	if err != nil {
		// We can panic here, as nothing would work if this ever panics.
		log.Panicln("Couldn't get default GTK Icon Theme:", err)
	}

	t, err := gtk.TextTagTableNew()
	if err != nil {
		log.Panicln("Failed to create a new text tag table:", err)
	}

	p := &Parser{
		State: s,
		theme: i,
		table: t,
	}
	p.pool = newPool(p)

	return p
}

func (p *Parser) NewTextBuffer() (*gtk.TextBuffer, error) {
	return gtk.TextBufferNew(p.table)
}

func (p *Parser) GetIcon(name string, size int) *gdk.Pixbuf {
	var key = name + "#" + strconv.Itoa(size)

	if v, ok := p.icons.Load(key); ok {
		return v.(*gdk.Pixbuf)
	}

	pb, err := p.theme.LoadIcon(name, size, gtk.ICON_LOOKUP_FORCE_SIZE)
	if err != nil {
		log.Errorln("Markdown: Failed to load icon", name+":", err)
		return nil
	}

	p.icons.Store(key, pb)
	return pb
}

func (p *Parser) Parse(md []byte, buf *gtk.TextBuffer) {
	p.ParseMessage(nil, md, buf)
}

func (p *Parser) ParseMessage(m *discord.Message, md []byte, buf *gtk.TextBuffer) {
	s := p.pool.Get().(*mdState)
	s.use(buf, md)

	var tree func(i int)
	if s == nil || m == nil {
		tree = s.switchTree
	} else {
		tree = s.switchTreeMessage(m)
	}

	s.iterMu.Lock()
	defer s.iterMu.Unlock()

	// Wipe the buffer clean
	semaphore.IdleMust(func(buf *gtk.TextBuffer) {
		buf.Delete(buf.GetStartIter(), buf.GetEndIter())
	}, buf)

	for i := 0; i < len(s.matches); i++ {
		s.prev = md[s.last:s.matches[i][0].from]
		s.last = s.getLastIndex(i)

		s.insertWithTag(s.prev, nil)
		tree(i)
	}

	s.insertWithTag(md[s.last:], nil)

	go func() {
		s.iterWg.Wait()
		s.buf = nil
		p.pool.Put(s)
	}()
}
