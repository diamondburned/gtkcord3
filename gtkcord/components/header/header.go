package header

import (
	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
)

type Header struct {
	*handy.HeaderBar
	Body *handy.Leaflet

	// Left: hamburger and guild name:
	LeftSide  *gtk.Box
	Hamburger *MainHamburger
	GuildName *gtk.Label

	Separator *gtk.Separator

	// Right: channel name only.
	RightSide   *gtk.Box
	Back        *Back
	ChannelName *gtk.Label
	ChMenuBtn   *ChMenuButton

	// Unused
	// Controller  *controller.Container
}

func NewHeader() *Header {
	l := handy.NewLeaflet()
	l.SetTransitionType(handy.LeafletTransitionTypeSlide)
	l.SetModeTransitionDuration(150)
	l.Container.SetHExpand(true)
	l.Container.Show()

	header := handy.NewHeaderBar()
	header.SetShowCloseButton(true)
	header.SetObjectProperty("spacing", 0)
	header.SetCustomTitle(empty())
	header.Add(&l.Container)
	header.Show()

	/*
	 * Left side
	 */

	left := gtk.NewBox(gtk.OrientationHorizontal, 0)
	left.Show()
	l.Add(left)

	hamburger := newMainHamburger()
	left.PackStart(hamburger, false, false, 0)

	hamseparator := gtk.NewSeparator(gtk.OrientationVertical)
	hamseparator.Show()
	left.PackStart(hamseparator, false, false, 0)

	// Calculate width for both:
	width := channel.ChannelsWidth - 15

	label := gtk.NewLabel("gtkcord3")
	label.Show()
	label.SetXAlign(0.0)
	label.SetMarginStart(15)
	label.SetSizeRequest(width, -1)
	label.SetLines(1)
	label.SetLineWrap(false)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetAttributes(gtkutils.PangoAttrs(
		pango.NewAttrWeight(pango.WeightSemibold),
	))
	left.PackStart(label, true, true, 0)

	lblseparator := gtk.NewSeparator(gtk.OrientationVertical)
	lblseparator.Show()
	left.PackStart(lblseparator, false, false, 0)

	/*
	 * Right side
	 */

	right := gtk.NewBox(gtk.OrientationHorizontal, 0)
	right.Show()
	l.Add(right)

	// Back button:
	back := NewBack()

	// Channel name:
	chname := gtk.NewLabel("")
	chname.SetLines(1)
	chname.SetLineWrap(false)
	chname.SetEllipsize(pango.EllipsizeEnd)
	chname.SetHExpand(true)
	chname.SetXAlign(0.0)
	chname.SetAttributes(gtkutils.PangoAttrs(
		pango.NewAttrWeight(pango.WeightSemibold),
	))
	chname.Show()

	// Channel menu button:
	btn := NewChMenuButton()

	rsep := gtk.NewSeparator(gtk.OrientationVertical)
	rsep.Show()
	gtkutils.Margin2(rsep, 0, 4)

	right.PackStart(back, false, false, 0)
	right.PackStart(chname, true, true, 0)
	right.PackStart(btn, false, false, 0)
	right.PackStart(rsep, false, false, 0)

	/*
	 * Grid 4
	 */

	// Button container for controls suck as Search, Members, etc.
	// cont := controller.New()
	// right.Add(cont)

	return &Header{
		HeaderBar:   header,
		Body:        l,
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
}

func (h *Header) Fold(folded bool) {
	// If folded, we reveal the back button.
	h.Back.SetRevealChild(folded)
	h.Separator.SetVisible(!folded)

	// Fold the title:
	if folded {
		h.GuildName.SetSizeRequest(-1, -1)
	} else {
		h.GuildName.SetSizeRequest(channel.ChannelsWidth-15, -1)
	}
}

func (h *Header) Cleanup() {
	h.GuildName.SetText("")
	h.ChannelName.SetText("")
}

func (h *Header) UpdateGuild(name string) {
	h.GuildName.SetText(name)
}

func (h *Header) UpdateChannel(name string) {
	h.ChannelName.SetText(name)
}

func empty() *gtk.Box {
	b := gtk.NewBox(gtk.OrientationHorizontal, 0)
	return b
}
