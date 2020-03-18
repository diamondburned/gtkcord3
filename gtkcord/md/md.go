package md

import (
	"regexp"
	"strings"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/styles"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

var regexes = []string{
	// codeblock
	`(?:\n?\x60\x60\x60 *(\S*)\n?([\s\S]*?)\n?\x60\x60\x60\n?)`,
	// blockquote
	`((?:(?:^|\n)^>\s+.*)+)\n?`,
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
	pool     = newPool()
)

type PressedEvent struct {
	*gdk.EventButton
	TextView *gtk.TextView
}

var ChannelPressed func(ev PressedEvent, ch discord.Channel)
var UserPressed func(ev PressedEvent, user discord.GuildUser)

func init() {
	log.Debugln("REGEX:", strings.Join(regexes, "|"))
	refreshStyle()
}

func ChangeStyle(styleName string) {
	HighlightStyle = styleName
	refreshStyle()
}

func refreshStyle() {
	style = styles.Get(HighlightStyle)
	if style == nil {
		panic("Unknown highlighting style: " + HighlightStyle)
	}
	css = styleToCSS(style)
}

func Parse(md []byte, buf *gtk.TextBuffer) {
	ParseMessage(nil, nil, md, buf)
}

type Discord interface {
	Channel(discord.Snowflake) (*discord.Channel, error)
	Member(guild, user discord.Snowflake) (*discord.Member, error)
}

func ParseMessage(d *ningen.State, m *discord.Message, md []byte, buf *gtk.TextBuffer) {
	// Boolean for message content:
	var content = false
	if m != nil && md == nil {
		content = true
		md = []byte(m.Content)
	}

	s := pool.Get().(*mdState)
	s.d = d
	s.m = m
	s.buf = buf
	s.fmtter.Reset()

	s.parseInput(md)

	var tree func(i int)
	if d == nil || m == nil {
		tree = s.switchTree
	} else {
		tree = s.switchTreeMessage
	}

	s.iterMu.Lock()

	semaphore.IdleMust(func() {
		// Populate the tags:
		s.tagTable()
		s.tag = s.colorTag(s.attr, s.color)

		// Wipe the buffer clean
		buf.Delete(buf.GetStartIter(), buf.GetEndIter())

		for i := 0; i < len(s.matches); i++ {
			s.prev = md[s.last:s.matches[i][0].from]
			s.last = s.getLastIndex(i)

			s.insertWithTag(s.prev, nil)
			tree(i)
		}

		s.insertWithTag(md[s.last:], nil)

		// Check if the message is edited:
		if m != nil && content && m.EditedTimestamp.Valid() {
			s.addEditedStamp(m.EditedTimestamp.Time())
		}
	})

	s.iterMu.Unlock()
	s.iterWg.Wait()

	s.d = nil
	s.m = nil
	s.buf = nil
	s.ttt = nil
	s.tag = nil
	s.last = 0
	s.prev = s.prev[:0]
	s.used = s.used[:0]
	s.hasText = false
	s.attr = 0
	s.color = ""

	pool.Put(s)
}
