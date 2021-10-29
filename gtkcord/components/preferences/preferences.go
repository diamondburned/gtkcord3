package preferences

import (
	"strconv"

	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
)

func Row(title, subtitle string, w gtk.Widgetter) *handy.ActionRow {
	r := handy.NewActionRow()
	r.SetTitle(title)
	r.SetSubtitle(subtitle)
	r.Show()

	// Set the proper orientation:

	if child, ok := r.Child().(gtk.Containerer); ok {
		child.SetObjectProperty("orientation", gtk.OrientationHorizontal)
	}

	// Set all labels to have markup:
	gtkutils.TraverseWidget(r, func(w gtk.Widgetter) {
		label, ok := w.(*gtk.Label)
		if ok {
			label.SetLineWrapMode(pango.WrapWordChar)
			label.SetEllipsize(pango.EllipsizeNone)
			label.SetUseMarkup(true)
		}
	})

	if w == nil {
		return r
	}

	r.Add(w)

	// Properly align the children:
	base := gtk.BaseWidget(w)
	base.SetVAlign(gtk.AlignCenter)
	base.SetMarginEnd(12)

	return r
}

// Permit only CSS files by MIME type.
func CSSFilter() *gtk.FileFilter {
	cssFilter := gtk.NewFileFilter()
	cssFilter.SetName("CSS Files")
	cssFilter.AddMIMEType("text/css")
	return cssFilter
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
		*s = fsb.Filename()
		update(updaters)
	})
}

func BindEntry(e *gtk.Entry, s *string, updaters ...func()) {
	e.SetHExpand(true)
	e.SetText(*s)
	update(updaters)

	e.Connect("changed", func() {
		*s = e.Text()
		update(updaters)
	})
}

func BindNumberEntry(e *gtk.Entry, input *int, updaters ...func()) {
	e.SetHExpand(true)
	e.SetInputPurpose(gtk.InputPurposeNumber)
	e.SetText(strconv.Itoa(*input))
	update(updaters)

	e.Connect("changed", func() {
		text := e.Text()
		log.Println("Input:", text)

		i, err := strconv.Atoi(text)
		if err != nil {
			EntryError(e, err)
			return
		}

		EntryError(e, err)
		*input = i
		update(updaters)
	})
}

func EntryError(entry *gtk.Entry, err error) {
	if err != nil {
		entry.SetIconFromIconName(gtk.EntryIconSecondary, "dialog-error")
		entry.SetIconTooltipText(gtk.EntryIconSecondary, err.Error())
	} else {
		entry.SetIconFromIconName(gtk.EntryIconSecondary, "")
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
