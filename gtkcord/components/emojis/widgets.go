package emojis

import (
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
)

type RevealerBox struct {
	*gtk.Box
	Revealer *gtk.Revealer
}

func newRevealerBox(btn *gtk.Button, reveal gtk.IWidget, click func()) *RevealerBox {
	r, _ := gtk.RevealerNew()
	r.Show()
	r.SetRevealChild(false)
	r.Add(reveal)

	// Wrap both the widget child and the revealer
	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	b.Show()
	b.Add(btn)
	b.Add(r)

	btn.Connect("clicked", click)

	return &RevealerBox{b, r}
}

func newHeader(name string, imgURL string) *gtk.Button {
	i, _ := gtk.ImageNew()
	i.Show()

	gtkutils.Margin(i, 4)
	gtkutils.ImageSetIcon(i, "image-missing", Size)

	l, _ := gtk.LabelNew(name)
	l.Show()
	l.SetMarginStart(4)
	l.SetHAlign(gtk.ALIGN_START)

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 4)
	box.Show()
	box.Add(i)
	box.Add(l)

	b, _ := gtk.ButtonNew()
	b.Show()
	b.SetRelief(gtk.RELIEF_NONE)
	b.Add(box)

	if imgURL == "" {
		return b
	}

	// ?size=64 is from the left-bar guilds icon.
	cache.AsyncFetchUnsafe(imgURL+"?size=64", i, Size, Size, cache.Round)

	return b
}

// func disableFocusScroll(s *gtk.ScrolledWindow) {
// 	// Make a custom viewport to prevent scroll to focus.
// 	w, _ := s.GetChild()
// 	c := &gtk.Container{Widget: *w}

// 	adj, _ := gtk.AdjustmentNew(0, 0, 0, 0, 0, 0)
// 	c.SetFocusHAdjustment(adj)
// 	c.SetFocusVAdjustment(adj)
// }

func newStaticViewport() *gtk.Viewport {
	adj, _ := gtk.AdjustmentNew(0, 0, 0, 0, 0, 0)

	v, _ := gtk.ViewportNew(nil, nil)
	v.SetFocusHAdjustment(adj)
	v.SetFocusVAdjustment(adj)

	return v
}

func newFlowBox() *gtk.FlowBox {
	f, _ := gtk.FlowBoxNew()
	f.Show()
	f.SetHomogeneous(true)
	f.SetSelectionMode(gtk.SELECTION_SINGLE)
	f.SetActivateOnSingleClick(true)
	f.SetMaxChildrenPerLine(10) // from Discord
	f.SetMinChildrenPerLine(10) // from Discord Mobile

	return f
}
