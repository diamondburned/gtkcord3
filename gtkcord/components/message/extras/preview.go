package extras

import (
	"path"
	"strings"

	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

type PreviewDialog struct {
	*gtk.Dialog
	Content *gtk.Box

	Image *gtk.Image

	OpenOriginal *gtk.Button
	ImageView    *gtk.ScrolledWindow

	Proxy string
	URL   string
}

func SpawnPreviewDialog(proxy, imageURL string) {
	// Trim the forms
	proxy = strings.Split(proxy, "?")[0]

	// Main dialog

	w := window.Window.AllocatedWidth()
	h := window.Window.AllocatedHeight()
	w = int(float64(w) * 0.85)
	h = int(float64(h) * 0.8)

	d := gtk.NewDialog()
	d.SetTransientFor(&window.Window.Window)
	d.SetDefaultSize(w, h)

	// Hack for close button
	// ??? unsure where this is triggered
	d.Connect("response", func(_ *glib.Object, resp gtk.ResponseType) {
		if resp == gtk.ResponseDeleteEvent {
			d.Hide()
			d.Destroy()
		}
	})

	// Header

	header := gtk.NewHeaderBar()
	header.SetShowCloseButton(true)

	bOriginal := gtk.NewButtonFromIconName(
		"image-x-generic-symbolic",
		int(gtk.IconSizeLargeToolbar),
	)
	bOriginal.SetTooltipText("Open Original")
	bOriginal.SetMarginStart(10)
	bOriginal.SetHAlign(gtk.AlignStart)

	header.PackStart(bOriginal)
	header.SetTitle(path.Base(imageURL))

	d.SetTitlebar(header)

	// Content box: image

	i := gtk.NewImageFromIconName("image-loading", int(gtk.IconSizeDialog))

	s := gtk.NewScrolledWindow(nil, nil)
	s.Add(i)
	s.SetVExpand(true)

	c := d.ContentArea()
	d.Remove(c)
	d.Add(s)
	d.ShowAll()

	pd := PreviewDialog{
		Dialog:  d,
		Content: c,

		Image: i,

		OpenOriginal: bOriginal,
		ImageView:    s,

		Proxy: proxy,
		URL:   imageURL,
	}

	bOriginal.Connect("clicked", func() {
		gtkutils.OpenURI(pd.URL)
	})

	// Calculate the sizee so that the image is just slightly (80%) smaller:
	// w = w * 8 / 10
	h = h * 9 / 10

	pd.Fetch(w, h)
	d.Run()
}

func (pd *PreviewDialog) Fetch(w, h int) {
	cache.SetImageStreamed(pd.Image, pd.Proxy, w, h)
}
