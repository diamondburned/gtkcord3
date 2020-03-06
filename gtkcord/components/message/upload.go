package message

import (
	"io"
	"os"
	"sync/atomic"
	"unsafe"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/gtkcord/window"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Uploader struct {
	*gtk.FileChooserNativeDialog
	callback   func(absolutePath []string)
	defaultDir string
}

//go:linkname gostring runtime.gostring
func gostring(p unsafe.Pointer) string

func SpawnUploader(callback func(absolutePath []string)) {
	dialog := semaphore.IdleMust(gtk.FileChooserDialogNewWith2Buttons,
		"Upload File", window.Window,
		gtk.FILE_CHOOSER_ACTION_OPEN,
		"Cancel", gtk.RESPONSE_CANCEL,
		"Upload", gtk.RESPONSE_ACCEPT,
	).(*gtk.FileChooserDialog)

	WithPreviewer(dialog)

	defaultDir := glib.GetUserDataDir()
	semaphore.IdleMust(dialog.SetCurrentFolder, defaultDir)
	semaphore.IdleMust(dialog.SetSelectMultiple, true)

	defer semaphore.IdleMust(dialog.Close)

	if res := semaphore.IdleMust(dialog.Run).(gtk.ResponseType); res != gtk.RESPONSE_ACCEPT {
		return
	}

	// Glib's shitty singly linked list:
	slist := semaphore.IdleMust(dialog.GetFilenames).(*glib.SList)
	var names = make([]string, 0, int(slist.Length()))
	slist.Foreach(func(ptr unsafe.Pointer) {
		names = append(names, gostring(ptr))
	})
	slist.Free()

	go callback(names)
}

type MessageUploader struct {
	*gtk.Box
	progresses []*ProgressUploader
}

func NewMessageUploader(paths []string) (*MessageUploader, error) {
	var m = &MessageUploader{}

	main := semaphore.IdleMust(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	m.Box = main

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to open "+path)
		}

		s, err := f.Stat()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to stat "+path)
		}

		m.progresses = append(m.progresses,
			NewProgressUploader(s.Name(), f, s.Size()))
	}

	semaphore.IdleMust(func(m *MessageUploader) {
		for _, p := range m.progresses {
			m.Box.PackEnd(p, false, false, 5)
		}
		m.ShowAll()
	}, m)

	return m, nil
}

func (m *MessageUploader) MakeSendData(message discord.Message) api.SendMessageData {
	s := api.SendMessageData{
		Content: message.Content,
		Nonce:   message.Nonce,
		Files:   make([]api.SendMessageFile, 0, len(m.progresses)),
	}

	for _, p := range m.progresses {
		s.Files = append(s.Files, api.SendMessageFile{
			Name:   p.Name,
			Reader: p,
		})
	}

	return s
}

func (m *MessageUploader) Close() {
	for _, p := range m.progresses {
		p.Close()
	}
}

type ProgressUploader struct {
	*gtk.Box
	bar  *gtk.ProgressBar
	name *gtk.Label
	Name string

	r io.ReadCloser
	s float64 // total
	n uint64
}

func NewProgressUploader(Name string, r io.ReadCloser, s int64) *ProgressUploader {
	box := semaphore.IdleMust(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	bar := semaphore.IdleMust(gtk.ProgressBarNew).(*gtk.ProgressBar)
	name := semaphore.IdleMust(gtk.LabelNew, Name).(*gtk.Label)
	semaphore.IdleMust(name.SetXAlign, float64(0))

	semaphore.IdleMust(box.Add, name)
	semaphore.IdleMust(box.Add, bar)

	return &ProgressUploader{
		Box:  box,
		bar:  bar,
		name: name,
		Name: Name,

		r: r,
		s: float64(s),
	}
}

func (p *ProgressUploader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)

	atomic.AddUint64(&p.n, uint64(n))
	glib.IdleAdd(p.bar.SetFraction, float64(p.n)/p.s)

	return n, err
}

func (p *ProgressUploader) Close() error {
	return p.r.Close()
}

func WithPreviewer(fc *gtk.FileChooserDialog) {
	img := semaphore.IdleMust(gtk.ImageNew).(*gtk.Image)

	semaphore.IdleMust(fc.SetPreviewWidget, img)
	semaphore.IdleMust(fc.Connect, "update-preview",
		func(fc *gtk.FileChooserDialog, img *gtk.Image) {
			file := fc.GetPreviewFilename()

			b, err := gdk.PixbufNewFromFileAtScale(file, 256, 256, true)
			if err != nil {
				fc.SetPreviewWidgetActive(false)
				return
			}

			img.SetFromPixbuf(b)
			fc.SetPreviewWidgetActive(true)
		},
		img,
	)
}