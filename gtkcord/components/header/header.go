package header

import (
	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
)

type Header struct {
	// Left: hamburger and guild name:
	Left      *handy.HeaderBar
	Hamburger *MainHamburger
	GuildName *gtk.Label

	Separator *gtk.Separator

	// Right: channel name only.
	Right       *handy.HeaderBar
	Back        *Back
	ChannelName *gtk.Label
	ChMenuBtn   *ChMenuButton

	// Unused
	// Controller  *controller.Container
}

func NewHeader() *Header {
	// l := handy.NewLeaflet()
	// l.SetTransitionType(handy.LeafletTransitionTypeSlide)
	// l.SetModeTransitionDuration(150)
	// l.Container.SetHExpand(true)
	// l.Container.Show()

	// header := handy.NewHeaderBar()
	// header.SetShowCloseButton(true)
	// header.SetObjectProperty("spacing", 0)
	// header.SetCustomTitle(empty())
	// header.Add(&l.Container)
	// header.Show()

	/*
	 * Left side
	 */

	hamburger := newMainHamburger()

	hamseparator := gtk.NewSeparator(gtk.OrientationVertical)
	hamseparator.Show()

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

	lblseparator := gtk.NewSeparator(gtk.OrientationVertical)
	lblseparator.Show()

	left := handy.NewHeaderBar()
	left.SetCustomTitle(empty())
	left.SetObjectProperty("spacing", 0)
	left.PackStart(hamburger)
	left.PackStart(hamseparator)
	left.PackStart(label)
	left.PackEnd(lblseparator)
	left.Show()

	/*
	 * Right side
	 */

	// rightBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	// rightBox.SetHExpand(true)
	// rightBox.Show()

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

	right := handy.NewHeaderBar()
	right.SetCustomTitle(empty())
	right.SetObjectProperty("spacing", 0)
	right.SetShowCloseButton(true)
	right.SetHExpand(true)
	right.PackStart(back)
	right.PackStart(chname)
	right.PackEnd(btn)
	right.Show()

	/*
	 * Grid 4
	 */

	// Button container for controls suck as Search, Members, etc.
	// cont := controller.New()
	// right.Add(cont)

	return &Header{
		Left:      left,
		Hamburger: hamburger,
		GuildName: label,
		Separator: lblseparator,

		Right:       right,
		Back:        back,
		ChannelName: chname,
		ChMenuBtn:   btn,
		// Controller:  cont,
	}
}

func (h *Header) Fold(folded bool) {
	// If folded, we reveal the back button.
	h.Back.SetRevealChild(folded)
	h.Left.SetShowCloseButton(folded)
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
