package md

import (
	"bytes"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var (
	messageCtx = parser.NewContextKey()
	sessionCtx = parser.NewContextKey()

	ChannelPressed func(ev PressedEvent, ch *discord.Channel)
	UserPressed    func(ev PressedEvent, user *discord.GuildUser)
)

func ParseMessageContent(dst *gtk.TextView, s state.Store, m *discord.Message) {
	parseMessage([]byte(m.Content), dst, s, m, true)
}

func ParseWithMessage(content []byte, dst *gtk.TextView, s state.Store, m *discord.Message) {
	parseMessage(content, dst, s, m, false)
}

func parseMessage(b []byte, dst *gtk.TextView, s state.Store, m *discord.Message, msg bool) {
	// Context to pass down messages:
	ctx := parser.NewContext()
	ctx.Set(messageCtx, m)
	ctx.Set(sessionCtx, s)

	var inlineParsers []util.PrioritizedValue
	if msg {
		inlineParsers = InlineParsers()
	} else {
		inlineParsers = InlineParserWithLink()
	}

	p := parser.NewParser(
		parser.WithBlockParsers(BlockParsers()...),
		parser.WithInlineParsers(inlineParsers...),
	)

	r := NewRenderer(dst)
	renderToBuf(NewRenderer(dst), b, p.Parse(text.NewReader(b), parser.WithContext(ctx)))

	// Is the not message edited? (Or if we don't want a timestamp)
	if !msg || !m.EditedTimestamp.Valid() {
		return
	}

	// Insert a timestamp
	footer := "  (edited " + humanize.TimeAgo(m.EditedTimestamp.Time()) + ")"
	r.insertWithTag([]byte(footer), r.tags.timestamp())
}

func Parse(content []byte, dst *gtk.TextView, opts ...parser.ParseOption) {
	p := parser.NewParser(
		parser.WithBlockParsers(BlockParsers()...),
		parser.WithInlineParsers(InlineParsers()...),
	)

	node := p.Parse(text.NewReader(content), opts...)
	renderToBuf(NewRenderer(dst), content, node)
}

func renderToBuf(r *Renderer, src []byte, node ast.Node) {
	// Wipe the buffer clean
	r.Buffer.Delete(r.Buffer.GetStartIter(), r.Buffer.GetEndIter())

	r.Render(nil, src, node)

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

func ParseToMarkupWithMessage(content []byte, s state.Store, m *discord.Message) []byte {
	// Context to pass down messages:
	ctx := parser.NewContext()
	ctx.Set(messageCtx, m)
	ctx.Set(sessionCtx, s)

	p := parser.NewParser(
		parser.WithBlockParsers(BlockParsers()...),
		parser.WithInlineParsers(InlineParsers()...),
	)

	node := p.Parse(text.NewReader(content), parser.WithContext(ctx))

	var buf bytes.Buffer
	NewMarkupRenderer().Render(&buf, content, node)

	return bytes.TrimSpace(buf.Bytes())
}

func ParseToSimpleMarkupWithMessage(content []byte, s state.Store, m *discord.Message) []byte {
	// Context to pass down messages:
	ctx := parser.NewContext()
	ctx.Set(messageCtx, m)
	ctx.Set(sessionCtx, s)

	p := parser.NewParser(
		parser.WithBlockParsers(BlockParsers()...),
		parser.WithInlineParsers(InlineParsers()...),
	)

	node := p.Parse(text.NewReader(content), parser.WithContext(ctx))

	var buf bytes.Buffer
	NewSimpleMarkupRenderer().Render(&buf, content, node)

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

func WrapTag(tv *gtk.TextView, props map[string]interface{}) {
	bf, _ := tv.GetBuffer()

	var name string
	for key := range props {
		name += key + " "
	}

	bf.ApplyTag(bf.CreateTag(name, props), bf.GetStartIter(), bf.GetEndIter())
}
