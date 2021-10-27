package md

import (
	"bytes"
	"sort"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/state/store"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/ningen/v2"
	"github.com/diamondburned/ningen/v2/md"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
)

var (
	messageCtx = parser.NewContextKey()
	sessionCtx = parser.NewContextKey()

	ChannelPressed func(ev PressedEvent, ch *discord.Channel)
	UserPressed    func(ev PressedEvent, user *discord.GuildUser)
)

func ParseMessageContent(dst *gtk.TextView, s *ningen.State, m *discord.Message) {
	parseMessage([]byte(m.Content), dst, s, m, true)
}

func ParseWithMessage(content []byte, dst *gtk.TextView, s *ningen.State, m *discord.Message) {
	parseMessage(content, dst, s, m, false)
}

func parseMessage(b []byte, dst *gtk.TextView, s *ningen.State, m *discord.Message, msg bool) {
	node := md.ParseWithMessage(b, s.Cabinet, m, msg)

	r := NewRenderer(dst)
	renderToBuf(NewRenderer(dst), b, node)

	// Is the not message edited? (Or if we don't want a timestamp)
	if !msg || !m.EditedTimestamp.IsValid() {
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
	r.Buffer.SetText("")

	r.Render(nil, src, node)

	// Remove trailing space:
	end := r.Buffer.EndIter()
	back := r.Buffer.EndIter()
	back.BackwardChar()

	r.Buffer.Delete(back, end)
}

func ParseToMarkup(content []byte) []byte {
	var buf bytes.Buffer
	NewMarkupRenderer().Render(&buf, content, md.Parse(content))

	return bytes.TrimSpace(buf.Bytes())
}

func ParseToMarkupWithMessage(content []byte, s store.Cabinet, m *discord.Message) []byte {
	node := md.ParseWithMessage(content, s, m, false)

	var buf bytes.Buffer
	NewMarkupRenderer().Render(&buf, content, node)

	return bytes.TrimSpace(buf.Bytes())
}

func ParseToSimpleMarkupWithMessage(content []byte, s store.Cabinet, m *discord.Message) []byte {
	node := md.ParseWithMessage(content, s, m, false)

	var buf bytes.Buffer
	NewSimpleMarkupRenderer().Render(&buf, content, node)

	return bytes.TrimSpace(buf.Bytes())
}

func WrapTag(tv *gtk.TextView, props map[string]interface{}) {
	bf := tv.Buffer()

	names := make([]string, 0, len(props))
	for key := range props {
		names = append(names, key)
	}
	sort.Strings(names)

	tag := gtk.NewTextTag(strings.Join(names, " "))
	for key, value := range props {
		tag.SetObjectProperty(key, value)
	}

	table := bf.TagTable()

	if table.Add(tag) {
		bf.ApplyTag(tag, bf.StartIter(), bf.EndIter())
	}
}
