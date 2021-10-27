package extras

import (
	"html"
	"io"
	"mime"
	"os"
	"path"
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/pkg/errors"
)

func NewAnyAttachment(name, url string, size uint64) gtk.Widgetter {
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

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.SetHAlign(gtk.AlignStart)
	box.SetSizeRequest(clampWidth(400), -1)
	box.Show()

	img := gtk.NewImageFromIconName(icon, 0)
	img.SetPixelSize(40)
	img.SetHAlign(gtk.AlignCenter)
	gtkutils.Margin2(img, 4, 0)
	img.Show()

	labels := gtk.NewBox(gtk.OrientationVertical, 0)
	labels.Show()
	labels.SetVAlign(gtk.AlignCenter)
	labels.SetHExpand(true)

	headerText := `<a href="` + url + `">` + html.EscapeString(name) + `</a>`
	header := gtk.NewLabel(headerText)
	header.SetUseMarkup(true)
	header.SetHAlign(gtk.AlignStart)
	header.SetSingleLineMode(true)
	header.SetEllipsize(pango.EllipsizeEnd)
	header.Show()

	subText := `<span size="smaller">` + humanize.Size(size) + `</span>`
	sub := gtk.NewLabel(subText)
	sub.SetEllipsize(pango.EllipsizeEnd)
	sub.SetHAlign(gtk.AlignStart)
	sub.SetUseMarkup(true)
	sub.Show()

	// TODO: progress bar for download button
	dl := gtk.NewButtonFromIconName("folder-download-symbolic", int(gtk.IconSizeLargeToolbar))
	dl.SetSizeRequest(35, 35)
	dl.SetVAlign(gtk.AlignCenter)
	dl.SetRelief(gtk.ReliefNone)
	dl.Connect("clicked", NewSaver(name, func(filename string) {
		// Reset the label (if failed).
		sub.SetMarkup(subText)

		// Only allow downloading once at a time.
		dl.SetSensitive(false)

		startDownload(url, filename, func(err error) {
			dl.SetSensitive(false)
			if err != nil {
				sub.SetMarkup(`<span color="red">` + "Error: " + err.Error() + `</span>`)
				log.Errorln("failed to download:", err)
			}
		})
	}))
	dl.Show()

	labels.Add(header)
	labels.Add(sub)

	box.PackStart(img, false, false, 5)
	box.PackStart(labels, true, true, 0)
	box.PackStart(dl, false, false, 3)

	gtkutils.InjectCSS(box, "attachment", `
		.attachment { background-color: @darker; }
	`)

	return box
}

func startDownload(url, dest string, doneFunc func(error)) {
	go func() {
		done := func(err error) {
			glib.IdleAdd(func() { doneFunc(err) })
		}

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
// 	r := gtk.NewRevealer()
// 	r.SetHExpand(true)
// 	r.Show()

// 	p := gtk.NewProgressBar()
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
// 	r := gtk.NewRevealer()
// 	r.Show()

// 	p := gtk.NewProgressBar()
// }

func NewSaver(filename string, onSave func(string)) func() {
	return func() {
		// Prompt the user
		d := gtk.NewFileChooserNative(
			"Save As", &window.Window.Window, gtk.FileChooserActionSave, "", "",
		)
		d.SetFilename(filename)

		if resp := d.Run(); gtk.ResponseType(resp) != gtk.ResponseAccept {
			return
		}

		onSave(d.Filename())
	}
}

// func DownloadTo(url string, onDone func()) {
// 	// Prompt the user
// 	d, err := gtk.NewFileChooserDialogWith1Button(
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
