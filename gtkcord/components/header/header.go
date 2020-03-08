package header

import (
	"html"

	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

type Header struct {
	gtkutils.ExtendedWidget
	Main *gtk.Box

	// Grid 1, on top of guilds
	Hamburger *Hamburger

	// Grid 2, on top of channels
	GuildName *gtk.Label
	// GuildButton TODO

	// Grid 3, on top of messages
	ChannelName *gtk.Label
	// Separator ---
	ChannelTopic *gtk.Label
}

func NewHeader(s *ningen.State) (*Header, error) {
	v, err := semaphore.Idle(newHeader, s)
	if err != nil {
		return nil, err
	}
	return v.(*Header), nil
}

func newHeader(s *ningen.State) (*Header, error) {
	g, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create grid")
	}

	/*
	 * Grid 1
	 */

	hamburger, err := NewHeaderMenu(s)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create hamburger")
	}
	hamseparator, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create ham separator")
	}
	g.PackStart(hamburger, false, false, 0)
	g.PackStart(hamseparator, false, false, 0)

	/*
	 * Grid 2
	 */

	label, err := gtk.LabelNew("gtkcord3")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create guild name label")
	}
	label.SetXAlign(0.0)
	label.SetMarginStart(15)
	label.SetSizeRequest(HeaderWidth-15, -1)
	label.SetLines(1)
	label.SetLineWrap(true)
	label.SetEllipsize(pango.ELLIPSIZE_END)
	lblseparator, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create ham separator")
	}
	g.PackStart(label, false, false, 0)
	g.PackStart(lblseparator, false, false, 0)

	/*
	 * Grid 3
	 */

	chname, err := gtk.LabelNew("")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create channel name")
	}
	chname.SetLines(1)
	chname.SetLineWrap(true)
	chname.SetEllipsize(pango.ELLIPSIZE_END)
	chname.SetXAlign(0.0)
	chname.SetMarginStart(20)
	chtopic, err := gtk.LabelNew("")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create channel topic")
	}
	chtopic.SetLines(1)
	chtopic.SetLineWrap(true)
	chtopic.SetEllipsize(pango.ELLIPSIZE_END)
	chtopic.SetXAlign(0.0)
	chtopic.SetMarginStart(20)
	chtopic.SetSingleLineMode(true)

	g.PackStart(chname, false, false, 0)
	g.PackStart(chtopic, false, false, 0)

	return &Header{
		ExtendedWidget: g,
		Main:           g,
		Hamburger:      hamburger,
		GuildName:      label,
		ChannelName:    chname,
		ChannelTopic:   chtopic,
	}, nil
}

func (h *Header) UpdateGuild(name string) {
	semaphore.IdleMust(h.GuildName.SetMarkup,
		`<span weight="bold">`+html.EscapeString(name)+`</span>`)
}

func (h *Header) UpdateChannel(name, topic string) {
	if name != "" {
		name = `<span weight="bold">` + "#" + html.EscapeString(name) + `</span>`
	}

	semaphore.IdleMust(func() {
		h.ChannelName.SetMarkup(name)
		h.ChannelTopic.SetText(topic)
	})
}
