package gtkcord

import (
	"io"
	"os"
	"unsafe"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Uploader struct {
	*gtk.FileChooserNativeDialog
	callback   func(absolutePath []string)
	defaultDir string
}

func SpawnUploader(callback func(absolutePath []string)) {
	dialog, err := gtk.FileChooserNativeDialogNew(
		"Upload File", App.Window,
		gtk.FILE_CHOOSER_ACTION_OPEN, "Upload", "Cancel")
	if err != nil {
		log.Panicln("Failed to spawn a native file chooser")
	}

	defaultDir := glib.GetUserDataDir()
	must(dialog.SetCurrentFolder, defaultDir)
	must(dialog.SetSelectMultiple, true)

	resCode := dialog.Run()
	if gtk.ResponseType(resCode) != gtk.RESPONSE_ACCEPT {
		return
	}

	// Glib's shitty singly linked list:
	slist := must(dialog.GetFilenames).(*glib.SList)
	var names = make([]string, 0, int(slist.Length()))
	slist.Foreach(func(ptr unsafe.Pointer) {
		names = append(names, *(*string)(ptr))
	})

	go callback(names)
}

// Spawn should be running in a goroutine.
func (u *Uploader) Spawn() {
	resCode := u.Run()
	if gtk.ResponseType(resCode) != gtk.RESPONSE_ACCEPT {
		return
	}

	// Glib's shitty singly linked list:
	slist := must(u.GetFilenames).(*glib.SList)
	var names = make([]string, 0, int(slist.Length()))
	slist.Foreach(func(ptr unsafe.Pointer) {
		names = append(names, *(*string)(ptr))
	})

	go u.callback(names)
}

type MessageUploader struct {
	*gtk.Box
	progresses []*ProgressUploader
}

func NewMessageUploader(paths []string) (*MessageUploader, error) {
	var m = &MessageUploader{}

	main := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL).(*gtk.Box)
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

	must(func(m *MessageUploader) {
		for _, p := range m.progresses {
			m.Box.Add(p)
			p.ShowAll()
		}
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
}

func NewProgressUploader(Name string, r io.ReadCloser, s int64) *ProgressUploader {
	box := must(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL).(*gtk.Box)
	bar := must(gtk.ProgressBarNew).(*gtk.ProgressBar)
	name := must(gtk.LabelNew, Name).(*gtk.Label)

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
	must(p.bar.SetFraction, float64(n)/p.s)
	return n, err
}

func (p *ProgressUploader) Close() error {
	return p.r.Close()
}
