package message

import (
	"fmt"
	"path"
	"strings"

	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

type PreviewDialog struct {
	*handy.Dialog
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

	w := window.Window.GetAllocatedWidth()
	h := window.Window.GetAllocatedHeight()
	w = int(float64(w) * 0.85)
	h = int(float64(h) * 0.8)

	d := handy.DialogNew(window.Window)
	if !d.GetNarrow() {
		d.SetDefaultSize(w, h)
	}

	// Hack for close button
	d.Connect("response", func(_ *glib.Object, resp gtk.ResponseType) {
		if resp == gtk.RESPONSE_DELETE_EVENT {
			d.Hide()
			d.Destroy()
		}
	})

	// Header

	header, _ := gtk.HeaderBarNew()
	header.SetShowCloseButton(true)

	bOriginal, _ := gtk.ButtonNewFromIconName(
		"image-x-generic-symbolic",
		gtk.ICON_SIZE_LARGE_TOOLBAR,
	)
	bOriginal.SetTooltipText("Open Original")
	bOriginal.SetMarginStart(10)
	bOriginal.SetHAlign(gtk.ALIGN_START)

	header.PackStart(bOriginal)
	header.SetTitle(path.Base(imageURL))

	d.SetTitlebar(header)

	// Content box: image

	i, err := gtk.ImageNewFromIconName("image-loading", gtk.ICON_SIZE_DIALOG)
	if err != nil {
		log.Panicln("Icon image-loading not found:", err)
	}

	s, _ := gtk.ScrolledWindowNew(nil, nil)
	s.Add(i)
	s.SetVExpand(true)

	c, _ := d.GetContentArea()
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
		go pd.Open()
	})

	// Calculate the sizee so that the image is just slightly (80%) smaller:
	// w = w * 8 / 10
	h = h * 9 / 10

	go pd.Fetch(w, h)
	d.Run()
}

func (od *PreviewDialog) Open() {
	if err := open.Run(od.URL); err != nil {
		log.Errorln("Failed to open image URL:", err)
	}
}

func (pd *PreviewDialog) Fetch(w, h int) {
	err := cache.SetImageAsync(pd.Proxy, pd.Image, w, h)
	if err == nil {
		return
	}

	err = errors.Wrap(err, "Failed to download the image")
	log.Errorln(err)

	errText := fmt.Sprintf(`<span color="red">%s</span>`, err)

	semaphore.IdleMust(func() {
		l, _ := gtk.LabelNew("")
		l.SetMarkup(errText)

		pd.Content.Remove(pd.Image)
		pd.Content.Add(l)
	})
}
