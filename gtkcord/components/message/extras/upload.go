package extras

import (
	"io"
	"os"
	"sync/atomic"
	"unsafe"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/sendpart"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"

	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/pkg/errors"
)

type Uploader struct {
	*gtk.FileChooserNative
	callback   func(absolutePath []string)
	defaultDir string
}

//go:linkname gostring runtime.gostring
func gostring(p unsafe.Pointer) string

var defaultDir string

func SpawnUploader(callback func(absolutePath []string)) {
	dialog := gtk.NewFileChooserNative(
		"Upload File", &window.Window.Window, gtk.FileChooserActionOpen,
		"Upload", "",
	)

	WithPreviewer(dialog)

	if defaultDir == "" {
		defaultDir = glib.GetUserDataDir()
	}

	dialog.SetLocalOnly(false)
	dialog.SetCurrentFolder(defaultDir)
	dialog.SetSelectMultiple(true)
	dialog.ConnectResponse(func(res int) {
		if gtk.ResponseType(res) == gtk.ResponseAccept {
			names := dialog.Filenames()
			callback(names)
		}
		dialog.Destroy()
	})
}

type MessageUploader struct {
	*gtk.Box
	progresses []*ProgressUploader
}

func NewMessageUploader(paths []string) (*MessageUploader, error) {
	var m MessageUploader

	main := gtk.NewBox(gtk.OrientationVertical, 0)
	m.Box = main

	go func() {
		type file struct {
			*os.File
			err  error
			name string
			size int64
		}

		files := make([]file, len(paths))
		onErr := func(i int, err error, wrap string) {
			files[i] = file{err: errors.Wrap(err, wrap)}
		}

		for i, path := range paths {
			f, err := os.Open(path)
			if err != nil {
				onErr(i, err, "failed to open")
				continue
			}

			s, err := f.Stat()
			if err != nil {
				onErr(i, err, "failed to stat")
				continue
			}

			files[i] = file{
				File: f,
				name: s.Name(),
				size: s.Size(),
			}
		}

		glib.IdleAdd(func() {
			m.progresses = make([]*ProgressUploader, len(files))
			for i, file := range files {
				m.progresses[i] = NewProgressUploader(file.name, file.File, file.size)
				m.progresses[i].err = file.err

				m.Box.PackEnd(m.progresses[i], false, false, 5)
			}
			m.ShowAll()
		})
	}()

	return &m, nil
}

func (m *MessageUploader) MakeSendData(message *discord.Message) api.SendMessageData {
	s := api.SendMessageData{
		Content: message.Content,
		Nonce:   message.Nonce,
		Files:   make([]sendpart.File, 0, len(m.progresses)),
	}

	for _, p := range m.progresses {
		s.Files = append(s.Files, sendpart.File{
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

	err error
	r   io.ReadCloser
	n   int64

	handle glib.SourceHandle
}

func NewProgressUploader(Name string, r io.ReadCloser, s int64) *ProgressUploader {
	bar := gtk.NewProgressBar()

	name := gtk.NewLabel(Name)
	name.SetXAlign(0)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Add(name)
	box.Add(bar)

	p := ProgressUploader{
		Box:  box,
		bar:  bar,
		name: name,
		Name: Name,

		r: r,
	}

	total := float64(s)

	p.handle = glib.TimeoutAdd(1000/30, func() bool {
		n := atomic.LoadInt64(&p.n)
		bar.SetFraction(float64(n) / total)

		if n < s {
			glib.SourceRemove(p.handle)
			p.handle = 0
			return false
		}

		return true
	})

	return &p
}

func (p *ProgressUploader) error(err error) {
	if err == nil || errors.Is(err, io.EOF) {
		return
	}

	glib.IdleAdd(func() {
		p.name.SetMarkup(p.Name + ` <span color="red">(error)</span>`)
		p.name.SetTooltipText("Error uploading: " + err.Error())
	})
}

func (p *ProgressUploader) done() {
	glib.IdleAdd(func() {
		if p.handle > 0 {
			glib.SourceRemove(p.handle)
			p.handle = 0
		}
	})
}

func (p *ProgressUploader) Read(b []byte) (int, error) {
	if p.r == nil && p.err != nil {
		p.error(p.err)
		return 0, p.err
	}

	n, err := p.r.Read(b)
	atomic.AddInt64(&p.n, int64(n))

	if err != nil {
		p.done()
	}

	return n, err
}

func (p *ProgressUploader) Close() error {
	p.done()
	return p.r.Close()
}

func WithPreviewer(fc *gtk.FileChooserNative) {
	img := gtk.NewImage()

	disable := func() {
		fc.SetPreviewWidgetActive(false)
		img.SetFromPixbuf(nil)
	}

	fc.SetPreviewWidget(img)
	fc.Connect("update-preview", func(fc *gtk.FileChooserDialog, img *gtk.Image) {
		file := fc.PreviewFilename()

		f, err := os.Open(file)
		if err != nil {
			disable()
			return
		}
		defer f.Close()

		l := gdkpixbuf.NewPixbufLoader()
		l.ConnectSizePrepared(func(w, h int) {
			l.SetSize(cache.MaxSize(w, h, 256, 256))
		})

		if _, err := io.Copy(gioutil.PixbufLoaderWriter(l), f); err != nil {
			disable()
			return
		}

		if err := l.Close(); err != nil {
			disable()
			return
		}

		if animation := l.Animation(); animation.IsStaticImage() {
			img.SetFromPixbuf(animation.StaticImage())
		} else {
			img.SetFromAnimation(l.Animation())
		}

		fc.SetPreviewWidgetActive(true)
	})
}
