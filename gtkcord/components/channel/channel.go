package channel

import (
	"html"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/internal/log"
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
	l.Show()
	l.SetUseMarkup(true)
	l.SetXAlign(0.0)
	l.SetMarginStart(8)
	l.SetMarginTop(8)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.SetSingleLineMode(true)
	l.SetMaxWidthChars(40)

	r, _ := gtk.ListBoxRowNew()
	r.Show()
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

	hash, _ := gtk.LabelNew(`<span size="x-large" weight="bold">#</span>`)
	hash.Show()
	hash.SetUseMarkup(true)
	hash.SetVAlign(gtk.ALIGN_CENTER)
	hash.SetHAlign(gtk.ALIGN_START)
	hash.SetMarginStart(8)
	hash.SetMarginEnd(8)

	l, _ := gtk.LabelNew(name)
	l.Show()
	l.SetVAlign(gtk.ALIGN_CENTER)
	l.SetHAlign(gtk.ALIGN_START)
	l.SetUseMarkup(true)

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.Show()
	b.SetHAlign(gtk.ALIGN_START)
	b.Add(hash)
	b.Add(l)

	r, _ := gtk.ListBoxRowNew()
	r.SetSizeRequest(-1, 20)
	r.Show()
	r.Add(b)

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

func (ch *Channel) ChannelID() discord.Snowflake {
	return ch.ID
}

func (ch *Channel) GuildID() discord.Snowflake {
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
