package preferences

import (
	"strconv"

	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

func Row(title, subtitle string, w gtk.IWidget) *handy.ActionRow {
	r := handy.ActionRowNew()
	r.SetTitle(title)
	r.SetSubtitle(subtitle)
	r.Show()

	// Set the proper orientation:
	if w, err := r.GetChild(); err == nil {
		w.(gtkutils.Object).SetProperty("orientation", gtk.ORIENTATION_HORIZONTAL)
		// Set all labels to have markup:
		gtkutils.TraverseWidget(r, func(w *gtk.Widget) {
			// Labels have use-markup
			if !gtkutils.HasProperty(w, "use-markup") {
				return
			}

			label := &gtk.Label{Widget: *w}
			label.SetLineWrapMode(pango.WRAP_WORD_CHAR)
			label.SetEllipsize(pango.ELLIPSIZE_NONE)
			label.SetUseMarkup(true)
		})
	}

	if w == nil {
		return r
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

// Permit only CSS files by MIME type.
func CSSFilter() *gtk.FileFilter {
	cssFilter, _ := gtk.FileFilterNew()
	cssFilter.SetName("CSS Files")
	cssFilter.AddMimeType("text/css")
	return cssFilter
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
	e.SetHExpand(true)
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

func BindNumberEntry(e *gtk.Entry, input *int, updaters ...func()) {
	e.SetHExpand(true)
	e.SetInputPurpose(gtk.INPUT_PURPOSE_NUMBER)
	e.SetText(strconv.Itoa(*input))
	update(updaters)

	e.Connect("changed", func() {
		t, err := e.GetText()
		if err != nil {
			log.Errorln("Failed to get entry text:", err)
			return
		}

		log.Println("Input:", t)

		i, err := strconv.Atoi(t)
		EntryError(e, err)

		if err != nil {
			return
		}

		*input = i
		update(updaters)
	})
}

func EntryError(entry *gtk.Entry, err error) {
	if err != nil {
		entry.SetIconFromIconName(gtk.ENTRY_ICON_SECONDARY, "dialog-error")
		entry.SetIconTooltipText(gtk.ENTRY_ICON_SECONDARY, err.Error())
	} else {
		entry.RemoveIcon(gtk.ENTRY_ICON_SECONDARY)
	}
}

func BindButton(b *gtk.Button, updaters ...func()) {
	b.Connect("clicked", func() {
		update(updaters)
	})
}

func update(updaters []func()) {
	for _, u := range updaters {
		u()
	}
}
