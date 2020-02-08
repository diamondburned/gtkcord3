package gtkcord

import (
	"github.com/gotk3/gotk3/gtk"
)

type Header struct {
	gtk.IWidget
	Bar  *gtk.HeaderBar
	Main *gtk.Box

	// Grid 1, on top of guilds
	Hamburger *HeaderMenu

	// Grid 2, on top of channels
	GuildName *gtk.Label

	// Grid 3, on top of messages
	ChannelName *gtk.Label
	// Separator ---
	ChannelTopic *gtk.Label
}

type HeaderMenu struct {
	*gtk.MenuButton
	Avatar *gtk.Image
	Name   *gtk.Label

	// About
}

/*
func newHeader() (*Header, error) {
	h, err := gtk.HeaderBarNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create headerbar")
	}

	g, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create grid")
	}
	g.SetOrientation(gtk.ORIENTATION_VERTICAL)
	g.Set

	h.PackStart(g)

	label, err := gtk.LabelNew("")
		if err != nil {
			return errors.Wrap(err, "Failed to create guild name label")
		}
		label.SetXAlign(0.0)
		label.SetMarginStart(20)
		label.SetSizeRequest(ChannelsWidth, -1)
		g.Channels.Name = label

	h := Header{}
}

func (h *Header) hookGuild(a *Application) {}
*/
