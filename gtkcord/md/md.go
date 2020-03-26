package md

import (
	"bytes"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func init() {
	// Refresh code highlighting styles
	refreshStyle()
}

var (
	messageCtx = parser.NewContextKey()
	sessionCtx = parser.NewContextKey()

	ChannelPressed func(ev PressedEvent, ch *discord.Channel)
	UserPressed    func(ev PressedEvent, user *discord.GuildUser)
)

func ParseMessageContent(dst *gtk.TextBuffer, s state.Store, m *discord.Message) {
	parseMessage([]byte(m.Content), dst, s, m, true)
}

func ParseWithMessage(content []byte, dst *gtk.TextBuffer, s state.Store, m *discord.Message) {
	parseMessage(content, dst, s, m, false)
}

func parseMessage(b []byte, dst *gtk.TextBuffer, s state.Store, m *discord.Message, ts bool) {
	// Context to pass down messages:
	ctx := parser.NewContext()
	ctx.Set(messageCtx, m)
	ctx.Set(sessionCtx, s)

	r := NewRenderer(dst)
	parse([]byte(b), r, parser.WithContext(ctx))

	// Is the not message edited? (Or if we don't want a timestamp)
	if !ts || !m.EditedTimestamp.Valid() {
		return
	}

	// Insert a timestamp
	footer := "  (edited " + humanize.TimeAgo(m.EditedTimestamp.Time()) + ")"
	r.insertWithTag([]byte(footer), r.tags.timestamp())
}

func Parse(content []byte, dst *gtk.TextBuffer, opts ...parser.ParseOption) {
	parse(content, NewRenderer(dst), opts...)
}

func parse(content []byte, r *Renderer, opts ...parser.ParseOption) {
	p := parser.NewParser(
		parser.WithBlockParsers(BlockParsers()...),
		parser.WithInlineParsers(InlineParsers()...),
	)

	node := p.Parse(text.NewReader(content), opts...)

	// Wipe the buffer clean
	r.Buffer.Delete(r.Buffer.GetStartIter(), r.Buffer.GetEndIter())

	r.Render(nil, content, node)

	// Remove trailing space:
	end := r.Buffer.GetEndIter()
	back := r.Buffer.GetEndIter()
	back.BackwardChar()

	r.Buffer.Delete(back, end)
}

func ParseToMarkup(content []byte) []byte {
	p := parser.NewParser(
		parser.WithBlockParsers(BlockParsers()...),
		parser.WithInlineParsers(InlineParsers()...),
	)

	node := p.Parse(text.NewReader(content))

	var buf bytes.Buffer
	NewMarkupRenderer().Render(&buf, content, node)

	return bytes.TrimSpace(buf.Bytes())
}

func getMessage(pc parser.Context) *discord.Message {
	if v := pc.Get(messageCtx); v != nil {
		return v.(*discord.Message)
	}
	return nil
}
func getSession(pc parser.Context) state.Store {
	if v := pc.Get(sessionCtx); v != nil {
		return v.(state.Store)
	}
	return nil
}
