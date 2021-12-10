package channel

import (
	"html"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
)

type Channel struct {
	gtk.Widgetter

	Row   *gtk.ListBoxRow
	Style *gtk.StyleContext

	Label *gtk.Label

	ID       discord.ChannelID
	Guild    discord.GuildID
	Name     string
	Topic    string
	Category bool

	stateClass string
	unread     bool
}

func createChannelRead(ch *discord.Channel, s *ningen.State) (w *Channel) {
	w = newChannel(ch)

	if ch.Type == discord.GuildCategory {
		return
	}

	if s.MutedState.Channel(ch.ID) {
		w.stateClass = "muted"
		w.Style.AddClass("muted")
		return
	}

	if rs := s.ReadState.FindLast(ch.ID); rs != nil {
		w.unread = ch.LastMessageID != rs.LastMessageID
		pinged := w.unread && rs.MentionCount > 0

		if !w.unread && pinged {
			pinged = false
		}

		switch {
		case pinged:
			w.stateClass = "pinged"
		case w.unread:
			w.stateClass = "unread"
		}

		if w.stateClass != "" {
			w.Style.AddClass(w.stateClass)
		}
	}

	return
}

func newChannel(ch *discord.Channel) *Channel {
	switch ch.Type {
	case discord.GuildText:
		return newChannelRow(ch)
	case discord.GuildCategory:
		return newCategory(ch)
	}

	log.Panicln("Unknown channel type", ch.Type)
	return nil
}

func newCategory(ch *discord.Channel) (chw *Channel) {
	name := `<span font_size="smaller">` + html.EscapeString(strings.ToUpper(ch.Name)) + "</span>"

	l := gtk.NewLabel(name)
	l.Show()
	l.SetUseMarkup(true)
	l.SetXAlign(0.0)
	l.SetMarginStart(8)
	l.SetMarginTop(8)
	l.SetEllipsize(pango.EllipsizeEnd)
	l.SetSingleLineMode(true)
	l.SetMaxWidthChars(40)

	r := gtk.NewListBoxRow()
	r.Show()
	r.SetSelectable(false)
	r.SetSensitive(false)
	r.Add(l)

	s := r.StyleContext()
	s.AddClass("category")

	chw = &Channel{
		Widgetter: r,

		Row:      r,
		Style:    s,
		Label:    l,
		ID:       ch.ID,
		Guild:    ch.GuildID,
		Name:     ch.Name,
		Topic:    ch.Topic,
		Category: true,
	}

	return chw
}

func newChannelRow(ch *discord.Channel) (chw *Channel) {
	name := `<span weight="bold">` + html.EscapeString(ch.Name) + `</span>`

	hash := gtk.NewLabel(`<span size="x-large" weight="bold">#</span>`)
	hash.Show()
	hash.SetUseMarkup(true)
	hash.SetVAlign(gtk.AlignCenter)
	hash.SetHAlign(gtk.AlignStart)
	hash.SetMarginStart(8)
	hash.SetMarginEnd(8)

	l := gtk.NewLabel(name)
	l.Show()
	l.SetVAlign(gtk.AlignCenter)
	l.SetHAlign(gtk.AlignStart)
	l.SetEllipsize(pango.EllipsizeEnd)
	l.SetUseMarkup(true)

	b := gtk.NewBox(gtk.OrientationHorizontal, 0)
	b.Show()
	b.SetHAlign(gtk.AlignStart)
	b.Add(hash)
	b.Add(l)

	r := gtk.NewListBoxRow()
	r.SetSizeRequest(-1, 16)
	r.Show()
	r.Add(b)

	s := r.StyleContext()
	s.AddClass("channel")

	chw = &Channel{
		Widgetter: r,

		Row:      r,
		Style:    s,
		Label:    l,
		ID:       ch.ID,
		Guild:    ch.GuildID,
		Name:     ch.Name,
		Topic:    ch.Topic,
		Category: false,
	}

	return chw
}

func (ch *Channel) ChannelID() discord.ChannelID {
	return ch.ID
}

func (ch *Channel) GuildID() discord.GuildID {
	return ch.Guild
}

func (ch *Channel) ChannelInfo() (name, topic string) {
	return ch.Name, ch.Topic
}

func (ch *Channel) setClass(class string) {
	gtkutils.DiffClass(&ch.stateClass, class, ch.Style)
}

func (ch *Channel) setUnread(unread, pinged bool) {
	if ch.stateClass == "muted" {
		return
	}

	ch.unread = unread

	if !unread && pinged {
		pinged = false
	}

	switch {
	case pinged:
		ch.setClass("pinged")
	case unread:
		ch.setClass("unread")
	default:
		ch.setClass("")
	}
}
