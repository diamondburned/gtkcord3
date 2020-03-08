package channel

import (
	"html"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type Channel struct {
	gtkutils.ExtendedWidget

	Row   *gtk.ListBoxRow
	Style *gtk.StyleContext

	Label *gtk.Label

	ID       discord.Snowflake
	Guild    discord.Snowflake
	Name     string
	Topic    string
	Category bool

	unread     bool
	stateClass string
}

func createChannelRead(ch *discord.Channel, s *ningen.State) (w *Channel) {
	w = newChannel(ch)

	if ch.Type == discord.GuildCategory {
		return
	}

	if s.ChannelMuted(ch.ID) {
		w.stateClass = "muted"
		w.Style.AddClass("muted")
		return
	}

	if rs := s.FindLastRead(ch.ID); rs != nil {
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

	l, _ := gtk.LabelNew(name)
	l.SetUseMarkup(true)
	l.SetXAlign(0.0)
	l.SetMarginStart(15)
	l.SetMarginTop(15)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.SetSingleLineMode(true)
	l.SetMaxWidthChars(40)

	r, _ := gtk.ListBoxRowNew()
	r.SetSelectable(false)
	r.SetSensitive(false)
	r.Add(l)

	s, _ := r.GetStyleContext()
	s.AddClass("category")

	chw = &Channel{
		ExtendedWidget: r,

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

	l, _ := gtk.LabelNew(ChannelHash + name)
	l.SetUseMarkup(true)
	l.SetXAlign(0.0)
	l.SetMarginStart(8)

	r, _ := gtk.ListBoxRowNew()
	r.Add(l)

	s, _ := r.GetStyleContext()
	s.AddClass("channel")

	chw = &Channel{
		ExtendedWidget: r,

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
