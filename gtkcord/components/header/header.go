package header

import (
	"html"

	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

type Header struct {
	*handy.Leaflet

	// Left: hamburger and guild name:
	LeftSide  *gtk.HeaderBar
	Hamburger *MainHamburger
	GuildName *gtk.Label

	Separator *gtk.Separator

	// Right: channel name only.
	RightSide   *gtk.HeaderBar
	Back        *Back
	ChannelName *gtk.Label
	ChMenuBtn   *ChMenuButton

	// Unused
	// Controller  *controller.Container
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

	hamburger, err := newMainHamburger()
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

	// Calculate width for both:
	width := channel.ChannelsWidth - 15

	label, err := gtk.LabelNew("gtkcord3")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create guild name label")
	}
	label.Show()
	label.SetXAlign(0.0)
	label.SetMarginStart(15)
	label.SetSizeRequest(width, -1)
	label.SetLines(1)
	label.SetLineWrap(false)
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
	right.SetSizeRequest(width, -1)
	right.SetShowCloseButton(true)
	right.SetProperty("spacing", 0)
	l.Add(right)

	body, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	body.Show()
	body.SetHExpand(true)
	right.SetCustomTitle(body)

	// Back button:
	back := NewBack()

	// Channel name:
	chname, _ := gtk.LabelNew("")
	chname.Show()
	chname.SetLines(1)
	chname.SetLineWrap(false)
	chname.SetEllipsize(pango.ELLIPSIZE_END)
	chname.SetHExpand(true)
	chname.SetXAlign(0.0)

	// Channel menu button:
	btn := NewChMenuButton()

	rsep, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	rsep.Show()
	gtkutils.Margin2(rsep, 0, 4)

	body.Add(back)
	body.Add(chname)
	body.Add(btn)
	body.Add(rsep)

	/*
	 * Grid 4
	 */

	// Button container for controls suck as Search, Members, etc.
	// cont := controller.New()
	// right.Add(cont)

	h := &Header{
		Leaflet:     l,
		LeftSide:    left,
		Hamburger:   hamburger,
		GuildName:   label,
		Separator:   lblseparator,
		RightSide:   right,
		Back:        back,
		ChannelName: chname,
		ChMenuBtn:   btn,
		// Controller:  cont,
	}

	return h, nil
}

func (h *Header) Fold(folded bool) {
	// If folded, we reveal the back button.
	h.Back.SetRevealChild(folded)

	// Fold the title:
	if folded {
		h.GuildName.SetSizeRequest(-1, -1)
		h.Separator.Hide()
		h.LeftSide.SetShowCloseButton(true)
	} else {
		h.GuildName.SetSizeRequest(channel.ChannelsWidth-15, -1)
		h.Separator.Show()
		h.LeftSide.SetShowCloseButton(false)
	}
}

func (h *Header) UpdateGuild(name string) {
	h.GuildName.SetMarkup(`<span weight="bold">` + html.EscapeString(name) + `</span>`)
}

func (h *Header) UpdateChannel(name string) {
	if name != "" {
		name = `<span weight="bold">` + "#" + html.EscapeString(name) + `</span>`
	}

	h.ChannelName.SetMarkup(name)
}

func empty() *gtk.Box {
	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	return b
}
