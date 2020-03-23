package header

import (
	"html"

	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/controller"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

type Header struct {
	*handy.Leaflet

	// Left: hamburger and guild name:
	LeftSide  *gtk.HeaderBar
	Hamburger *Hamburger
	GuildName *gtk.Label

	// Right: channel name only.
	RightSide   *gtk.HeaderBar
	ChannelName *gtk.Label
	Controller  *controller.Container
}

func NewHeader() (*Header, error) {
	l := handy.LeafletNew()
	l.SetTransitionType(handy.LEAFLET_TRANSITION_TYPE_SLIDE)
	l.SetModeTransitionDuration(150)
	l.SetHExpand(true)
	l.Show()

	/*
	 * Left side
	 */

	left, _ := gtk.HeaderBarNew()
	left.SetShowCloseButton(false)
	left.SetProperty("spacing", 0)
	left.Show()
	left.SetCustomTitle(empty())
	l.Add(left)

	hamburger, err := NewHeaderMenu()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create hamburger")
	}
	left.Add(hamburger)

	hamseparator, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create ham separator")
	}
	hamseparator.Show()
	left.Add(hamseparator)

	label, err := gtk.LabelNew("gtkcord3")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create guild name label")
	}
	label.Show()
	label.SetXAlign(0.0)
	label.SetMarginStart(15)
	label.SetSizeRequest(channel.ChannelsWidth-15, -1)
	label.SetLines(1)
	label.SetLineWrap(true)
	label.SetEllipsize(pango.ELLIPSIZE_END)
	left.Add(label)

	lblseparator, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create ham separator")
	}
	lblseparator.Show()
	left.Add(lblseparator)

	/*
	 * Right side
	 */

	right, _ := gtk.HeaderBarNew()
	right.Show()
	right.SetHExpand(true)
	right.SetShowCloseButton(true)
	right.SetProperty("spacing", 0)
	right.SetCustomTitle(empty())
	l.Add(right)

	chname, _ := gtk.LabelNew("")
	chname.Show()
	chname.SetLines(1)
	chname.SetLineWrap(true)
	chname.SetEllipsize(pango.ELLIPSIZE_END)
	chname.SetXAlign(0.0)
	chname.SetMarginStart(20)
	right.Add(chname)

	/*
	 * Grid 4
	 */

	// Button container for controls suck as Search, Members, etc.
	cont := controller.New()
	right.Add(cont)

	return &Header{
		Leaflet:     l,
		LeftSide:    left,
		Hamburger:   hamburger,
		GuildName:   label,
		RightSide:   right,
		ChannelName: chname,
		Controller:  cont,
	}, nil
}

func (h *Header) UpdateGuild(name string) {
	semaphore.IdleMust(h.GuildName.SetMarkup,
		`<span weight="bold">`+html.EscapeString(name)+`</span>`)
}

func (h *Header) UpdateChannel(name string) {
	if name != "" {
		name = `<span weight="bold">` + "#" + html.EscapeString(name) + `</span>`
	}

	semaphore.IdleMust(h.ChannelName.SetMarkup, name)
}

func empty() *gtk.Box {
	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	return b
}
