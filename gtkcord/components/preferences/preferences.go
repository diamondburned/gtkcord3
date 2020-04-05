package preferences

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

func Row(title, subtitle string, w gtk.IWidget) *handy.ActionRow {
	r := handy.ActionRowNew()
	r.SetTitle(title)
	r.SetSubtitle(subtitle)
	r.Show()

	// Set the proper orientation:
	if w, err := r.GetChild(); err == nil {
		w.SetProperty("orientation", gtk.ORIENTATION_HORIZONTAL)
		// Set all labels to have markup:
		gtkutils.TraverseWidget(r, func(w *gtk.Widget) {
			// Labels have use-markup
			if !gtkutils.HasProperty(w, "use-markup") {
				log.Println("Not label")
				return
			}

			w.SetProperty("use-markup", true)
		})
	}

	r.Add(w)

	// Properly align the children:
	if a, ok := w.(interface{ SetVAlign(gtk.Align) }); ok {
		a.SetVAlign(gtk.ALIGN_CENTER)
	}
	if m, ok := w.(gtkutils.Marginator); ok {
		m.SetMarginEnd(12)
	}

	return r
}

// func FileChooser()

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
