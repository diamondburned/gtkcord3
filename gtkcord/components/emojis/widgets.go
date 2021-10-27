package emojis

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/roundimage"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
)

type RevealerBox struct {
	*gtk.Box
	Revealer *gtk.Revealer
}

func (r *RevealerBox) ConnectRevealChild(f func(revealed bool)) {
	r.Revealer.Connect("notify::reveal-child", func() {
		f(r.Revealer.RevealChild())
	})
}

func newRevealerBox(btn *gtk.ToggleButton, reveal gtk.Widgetter) *RevealerBox {
	r := gtk.NewRevealer()
	r.Show()
	r.SetRevealChild(false)
	r.Add(reveal)

	// Wrap both the widget child and the revealer
	b := gtk.NewBox(gtk.OrientationVertical, 0)
	b.Show()
	b.Add(btn)
	b.Add(r)

	btn.ConnectToggled(func() {
		r.SetRevealChild(btn.Active())
	})

	return &RevealerBox{b, r}
}

func newHeaderButton(name string, imgURL string) *gtk.ToggleButton {
	i := roundimage.NewImage(0)
	gtkutils.Margin(i, 4)

	l := gtk.NewLabel(name)
	l.SetMarginStart(4)
	l.SetHAlign(gtk.AlignStart)

	box := gtk.NewBox(gtk.OrientationHorizontal, 4)
	box.Add(i)
	box.Add(l)

	b := gtk.NewToggleButton()
	b.SetRelief(gtk.ReliefNone)
	b.Add(box)
	b.ShowAll()

	if imgURL == "" {
		i.SetInitials(name)
		return b
	}

	// ?size=64 is from the left-bar guilds icon.
	cache.SetImageURLScaled(i, imgURL+"?size=64", Size, Size)

	return b
}

func newStaticViewport() *gtk.Viewport {
	adj := gtk.NewAdjustment(0, 0, 0, 0, 0, 0)

	v := gtk.NewViewport(nil, nil)
	v.SetFocusHAdjustment(adj)
	v.SetFocusVAdjustment(adj)

	return v
}

func newFlowBox() *gtk.FlowBox {
	f := gtk.NewFlowBox()
	f.SetHomogeneous(true)
	f.SetSelectionMode(gtk.SelectionSingle)
	f.SetActivateOnSingleClick(true)
	f.SetMaxChildrenPerLine(10) // from Discord
	f.SetMinChildrenPerLine(10) // from Discord Mobile
	f.Show()

	return f
}
