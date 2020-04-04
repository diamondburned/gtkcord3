package preferences

import (
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

func Row(title, subtitle string, w gtk.IWidget) *handy.ActionRow {
	r := handy.ActionRowNew()
	r.SetTitle(title)
	r.SetSubtitle(subtitle)
	r.SetActivatableWidget(w)
	r.Add(w)
	r.ShowAll()

	return r
}

func BindSwitch(s *gtk.Switch, b *bool, updaters ...func()) {
	s.SetActive(*b)
	update(updaters)

	s.Connect("state-set", func(_ *gtk.Switch, state bool) {
		*b = state
		update(updaters)
	})
}

func BindFileChooser(fsb *gtk.FileChooserButton, s *string, updaters ...func()) {
	fsb.SetFilename(*s)
	update(updaters)

	fsb.Connect("file-set", func() {
		*s = fsb.GetFilename()
		update(updaters)
	})
}

func BindEntry(e *gtk.Entry, s *string, updaters ...func()) {
	e.SetText(*s)
	update(updaters)

	e.Connect("changed", func() {
		t, err := e.GetText()
		if err != nil {
			log.Errorln("Failed to get entry text:", err)
			return
		}

		*s = t
		update(updaters)
	})
}

func update(updaters []func()) {
	for _, u := range updaters {
		u()
	}
}
