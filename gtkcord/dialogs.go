package gtkcord

import (
	"fmt"
	"path"
	"strings"

	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/gtkcord/window"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

type PreviewDialog struct {
	*gtk.Dialog
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
	w = int(float64(w) / 1.5)
	h = int(float64(h) / 1.5)

	d, _ := gtk.DialogNew()
	d.SetModal(true)
	d.SetTransientFor(window.Window)
	d.SetDefaultSize(w, h)

	// Header

	header, _ := gtk.HeaderBarNew()

	bOriginal, _ := gtk.ButtonNewFromIconName(
		"image-x-generic-symbolic",
		gtk.ICON_SIZE_LARGE_TOOLBAR,
	)
	bOriginal.SetTooltipText("Open Original")
	bOriginal.SetMarginStart(10)
	bOriginal.SetHAlign(gtk.ALIGN_START)

	bExit, _ := gtk.ButtonNewFromIconName(
		"window-close-symbolic",
		gtk.ICON_SIZE_LARGE_TOOLBAR,
	)
	bExit.SetTooltipText("Close")
	bExit.SetMarginEnd(10)
	bExit.SetHAlign(gtk.ALIGN_END)

	header.PackStart(bOriginal)
	header.SetTitle(path.Base(imageURL))
	header.PackEnd(bExit)

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
	c.Add(s)
	d.ShowAll()

	pd := PreviewDialog{
		Dialog: d,
		Image:  i,

		OpenOriginal: bOriginal,
		ImageView:    s,

		Proxy: proxy,
		URL:   imageURL,
	}

	bOriginal.Connect("clicked", func() {
		go pd.Open()
	})
	bExit.Connect("clicked", func() {
		pd.Dialog.Hide()
		pd.Dialog.Destroy()
	})

	go pd.Fetch(w, h)

	d.Run()
}

func (od *PreviewDialog) Open() {
	if err := open.Run(od.URL); err != nil {
		log.Errorln("Failed to open image URL:", err)
	}
}

func (pd *PreviewDialog) Fetch(w, h int) {
	err := cache.SetImage(pd.Proxy, pd.Image, cache.Resize(w, h))
	if err == nil {
		return
	}
	err = errors.Wrap(err, "Failed to download the image")
	log.Errorln(err)

	errText := fmt.Sprintf(`<span color="red">%s</span>`, err)

	semaphore.IdleMust(func() {
		l, _ := gtk.LabelNew("")
		l.SetMarkup(errText)

		pd.Dialog.Remove(pd.ImageView)
		pd.Dialog.Add(l)
	})
}
