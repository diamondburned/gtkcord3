package md

import (
	"bytes"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/ningen/md"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
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
	node := md.ParseWithMessage(b, s, m, msg)

	r := NewRenderer(dst)
	renderToBuf(NewRenderer(dst), b, node)

	// Is the not message edited? (Or if we don't want a timestamp)
	if !msg || !m.EditedTimestamp.Valid() {
		return
	}

	// Insert a timestamp
	footer := "  (edited " + humanize.TimeAgo(m.EditedTimestamp.Time()) + ")"
	r.insertWithTag([]byte(footer), r.tags.timestamp())
}

func Parse(content []byte, dst *gtk.TextView, opts ...parser.ParseOption) {
	node := md.Parse(content, opts...)
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
	var buf bytes.Buffer
	NewMarkupRenderer().Render(&buf, content, md.Parse(content))

	return bytes.TrimSpace(buf.Bytes())
}

func ParseToMarkupWithMessage(content []byte, s state.Store, m *discord.Message) []byte {
	node := md.ParseWithMessage(content, s, m, false)

	var buf bytes.Buffer
	NewMarkupRenderer().Render(&buf, content, node)

	return bytes.TrimSpace(buf.Bytes())
}

func ParseToSimpleMarkupWithMessage(content []byte, s state.Store, m *discord.Message) []byte {
	node := md.ParseWithMessage(content, s, m, false)

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
