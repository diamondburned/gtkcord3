package header

import (
	"html"

	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/controller"
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

	// Box that contains guild name and separators
	GuildBox  *gtk.Box
	GuildName *gtk.Label

	// Grid 3, on top of messages
	ChannelName  *gtk.Label
	ChannelTopic *gtk.Label

	Controller *controller.Container
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
	g.SetHExpand(true)
	g.Show()

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

	gb, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	gb.Show()
	gb.SetVExpand(true)
	g.PackStart(gb, false, false, 0)

	label, err := gtk.LabelNew("gtkcord3")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create guild name label")
	}
	label.SetXAlign(0.0)
	label.SetMarginStart(15)
	label.SetSizeRequest(channel.ChannelsWidth-15, -1)
	label.SetLines(1)
	label.SetLineWrap(true)
	label.SetEllipsize(pango.ELLIPSIZE_END)
	lblseparator, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create ham separator")
	}
	gb.PackStart(label, false, false, 0)
	gb.PackStart(lblseparator, false, false, 0)

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

	// Show all before adding the controller:
	g.ShowAll()

	/*
	 * Grid 4
	 */

	// Button container for controls suck as Search, Members, etc.
	cont := controller.New()
	g.PackStart(cont, true, true, 0)

	return &Header{
		ExtendedWidget: g,
		Main:           g,
		Hamburger:      hamburger,
		GuildBox:       gb,
		GuildName:      label,
		ChannelName:    chname,
		ChannelTopic:   chtopic,
		Controller:     cont,
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
