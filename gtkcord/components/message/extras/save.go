package extras

import (
	"io"
	"mime"
	"os"
	"path"
	"strings"

	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

func NewAnyAttachmentUnsafe(name, url string, size uint64) gtkutils.ExtendedWidget {
	var icon string

	switch MIME := mime.TypeByExtension(path.Ext(url)); {
	case strings.HasPrefix(MIME, "image"):
		icon = "image-x-generic-symbolic"
	case strings.HasPrefix(MIME, "video"):
		icon = "video-x-generic-symbolic"
	case strings.HasPrefix(MIME, "audio"):
		icon = "audio-x-generic-symbolic"
	default:
		icon = "text-x-generic-symbolic"
	}

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.Show()
	box.SetHAlign(gtk.ALIGN_START)
	box.SetSizeRequest(clampWidth(400), -1)

	img, _ := gtk.ImageNew()
	img.Show()
	img.SetHAlign(gtk.ALIGN_CENTER)
	gtkutils.Margin2(img, 4, 0)
	gtkutils.ImageSetIcon(img, icon, 40)

	labels, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	labels.Show()
	labels.SetVAlign(gtk.ALIGN_CENTER)
	labels.SetHExpand(true)

	header, _ := gtk.LabelNew(name)
	header.Show()
	header.SetHAlign(gtk.ALIGN_START)
	header.SetSingleLineMode(true)
	header.SetEllipsize(pango.ELLIPSIZE_END)

	subText := `<span size="smaller">` + humanize.Size(size) + `</span>`
	sub, _ := gtk.LabelNew(subText)
	sub.Show()
	sub.SetEllipsize(pango.ELLIPSIZE_END)
	sub.SetHAlign(gtk.ALIGN_START)
	sub.SetUseMarkup(true)

	open, _ := gtk.ButtonNewFromIconName("document-open-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR)
	open.Show()
	open.SetSizeRequest(35, 35)
	open.SetVAlign(gtk.ALIGN_CENTER)
	open.SetRelief(gtk.RELIEF_NONE)
	open.Connect("clicked", func() {
		gtkutils.OpenURI(url)
	})

	// TODO: progress bar for download button
	dl, _ := gtk.ButtonNewFromIconName("folder-download-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR)
	dl.Show()
	dl.SetSizeRequest(35, 35)
	dl.SetVAlign(gtk.ALIGN_CENTER)
	dl.SetRelief(gtk.RELIEF_NONE)
	dl.Connect("clicked", NewSaver(name, func(filename string) {
		// Reset the label (if failed).
		sub.SetMarkup(subText)

		// Only allow downloading once at a time.
		dl.SetSensitive(false)

		startDownload(url, filename, func(err error) {
			semaphore.Async(dl.SetSensitive, true)
			if err == nil {
				return
			}

			log.Errorln("Failed to download:", err)
			semaphore.Async(sub.SetMarkup, `<span color="red">`+"Error: "+err.Error()+`</span>`)
		})
	}))

	labels.Add(header)
	labels.Add(sub)

	box.PackStart(img, false, false, 5)
	box.PackStart(labels, true, true, 0)
	box.PackStart(open, false, false, 3)
	box.PackStart(dl, false, false, 3)

	gtkutils.InjectCSSUnsafe(box, "attachment", `
		.attachment { background-color: @darker; }
	`)

	return box
}

func startDownload(url, dest string, done func(error)) {
	go func() {
		f, err := os.Create(dest)
		if err != nil {
			done(errors.Wrap(err, "Failed to create file"))
			return
		}
		defer f.Close()

		r, err := cache.Client.Get(url)
		if err != nil {
			done(errors.Wrap(err, "Failed to GET"))
			return
		}
		defer r.Body.Close()

		if r.StatusCode < 200 || r.StatusCode > 299 {
			done(errors.Errorf("Non-success status code: %d", r.StatusCode))
			return
		}

		if _, err := io.Copy(f, r.Body); err != nil {
			done(errors.Wrap(err, "Failed to write to file"))
		}

		done(nil)
	}()
}

// type ProgressDownloader struct {
// 	*gtk.Revealer
// 	Bar *gtk.ProgressBar

// 	w io.Writer
// 	s float64 // total
// 	n uint64
// }

// func NewProgressDownloader(w io.Writer, s uint64) *ProgressDownloader {
// 	r, _ := gtk.RevealerNew()
// 	r.SetHExpand(true)
// 	r.Show()

// 	p, _ := gtk.ProgressBarNew()
// 	p.Show()

// 	r.SetRevealChild(false)
// 	r.Add(p)

// 	return &ProgressDownloader{
// 		Revealer: r,
// 		Bar:      p,

// 		w: w,
// 		s: float64(s),
// 	}
// }

// func (d *ProgressDownloader) Write(b []byte) (int, error) {
// 	n, err := d.w.Write(b)

// 	atomic.AddUint64(&d.n, uint64(n))
// 	frac := float64(d.n) / d.s

// 	semaphore.Async(func() {
// 		d.Bar.SetFraction(frac)
// 	})

// 	return n, err
// }

// func NewDownloadButton(url string) *gtk.Revealer {
// 	r, _ := gtk.RevealerNew()
// 	r.Show()

// 	p, _ := gtk.ProgressBarNew()
// }

func NewSaver(filename string, onSave func(string)) func() {
	return func() {
		// Prompt the user
		d, err := gtk.FileChooserDialogNewWith1Button(
			"Save As", window.Window, gtk.FILE_CHOOSER_ACTION_SAVE,
			"Save", gtk.RESPONSE_ACCEPT,
		)
		if err != nil {
			log.Panicln("Failed to open download dialog:", err)
		}

		d.SetFilename(filename)

		if resp := d.Run(); resp != gtk.RESPONSE_ACCEPT {
			return
		}

		onSave(d.GetFilename())
	}
}

// func DownloadTo(url string, onDone func()) {
// 	// Prompt the user
// 	d, err := gtk.FileChooserDialogNewWith1Button(
// 		"Save As", window.Window, gtk.FILE_CHOOSER_ACTION_SAVE,
// 		"Save", gtk.RESPONSE_ACCEPT,
// 	)
// 	if err != nil {
// 		log.Panicln("Failed to open download dialog:", err)
// 	}

// 	if resp := d.Run(); resp != gtk.RESPONSE_ACCEPT {
// 		return
// 	}

// 	// Start downloading in the background.
// 	go startDownload(url)
// }
