package gtkutils

import (
	"html"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"

	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/skratchdot/open-golang/open"
)

func Margin4(widget gtk.Widgetter, top, bottom, left, right int) {
	w := gtk.BaseWidget(widget)
	w.SetMarginTop(top)
	w.SetMarginBottom(bottom)
	w.SetMarginStart(left)
	w.SetMarginEnd(right)
}

func Margin2(widget gtk.Widgetter, top, left int) {
	Margin4(widget, top, top, left, left)
}

func Margin(widget gtk.Widgetter, sz int) {
	Margin2(widget, sz, sz)
}

func TransferMargin(dst, src gtk.Widgetter) {
	d := gtk.BaseWidget(dst)
	s := gtk.BaseWidget(src)

	d.SetMarginBottom(s.MarginBottom())
	d.SetMarginEnd(s.MarginEnd())
	d.SetMarginStart(s.MarginStart())
	d.SetMarginTop(s.MarginTop())
	// reset src to 0
	Margin(s, 0)
}

func AsContainer(w gtk.Widgetter) *gtk.Container {
	container, ok := w.(gtk.Containerer)
	if !ok {
		return nil
	}
	return gtk.BaseContainer(container)
}

func NthChildren(container gtk.Containerer, i int) gtk.Widgetter {
	list := gtk.BaseContainer(container).Children()
	if list == nil {
		return nil
	}
	return list[i]
}

func HasProperty(obj glib.Objector, name string) bool {
	v := obj.ObjectProperty(name)
	return v != nil && v != coreglib.InvalidValue
}

// fn() == true => break
func TraverseWidget(container gtk.Containerer, fn func(gtk.Widgetter)) {
	for _, wd := range gtk.BaseContainer(container).Children() {
		fn(wd)

		// Recurse
		if c := AsContainer(wd); c != nil {
			TraverseWidget(c, fn)
		}
	}
}

func WrapBox(orient gtk.Orientation, widgets ...gtk.Widgetter) *gtk.Box {
	b := gtk.NewBox(orient, 0)
	for _, w := range widgets {
		b.Add(w)
	}
	b.ShowAll()
	return b
}

func InjectCSS(w gtk.Widgetter, class, CSS string) {
	style := gtk.BaseWidget(w).StyleContext()

	if class != "" {
		style.AddClass(class)
	}

	if CSS != "" {
		AddCSS(style, CSS)
	}
}

func AddCSS(style *gtk.StyleContext, CSS string) {
	f := CSSAdder(CSS)
	f(style)
}

func CSSAdder(CSS string) func(style *gtk.StyleContext) {
	css := gtk.NewCSSProvider()
	if err := css.LoadFromData(CSS); err != nil {
		log.Errorln("failed to load CSS:", err)
		return func(*gtk.StyleContext) { log.Errorln("use of erroneous CSS provider") }
	}
	return func(s *gtk.StyleContext) {
		s.AddProvider(css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	}
}

// OnMap binds f to w's mapping signals.
func OnMap(w gtk.Widgetter, f func() func()) {
	widget := gtk.BaseWidget(w)
	if widget.Mapped() {
		panic("OnMap: called with mapped widget")
	}

	var cancel func()

	widget.ConnectMap(func() {
		cancel = f()
	})
	widget.ConnectUnmap(func() {
		cancel()
	})
}

func Escape(str string) string {
	return html.EscapeString(str)
}

func Bold(str string) string {
	return "<b>" + Escape(str) + "</b>"
}

func KeyIsASCII(key uint) bool {
	return key >= gdk.KEY_exclam && key <= gdk.KEY_asciitilde
}

func DiffClass(old *string, new string, style *gtk.StyleContext) {
	if *old == new {
		return
	}

	if *old != "" {
		style.RemoveClass(*old)
	}

	*old = new

	if new == "" {
		return
	}

	style.AddClass(new)
}

// EventIsRightClick see EventIsMouseButton.
func EventIsRightClick(ev *gdk.Event) bool {
	return EventIsMouseButton(ev, gdk.BUTTON_SECONDARY)
}

// EventIsLeftClick see EventIsMouseButton.
func EventIsLeftClick(ev *gdk.Event) bool {
	return EventIsMouseButton(ev, gdk.BUTTON_PRIMARY)
}

// EventIsMouseButton returns true if the given event is a button press event on
// the given button.
func EventIsMouseButton(ev *gdk.Event, mouseBtn uint) bool {
	return ev.AsType() == gdk.ButtonPressType && ev.AsButton().Button() == mouseBtn
}

// OpenURI, TODO: deprecate this
func OpenURI(uri string) {
	/* TODO: INSPECT ME */ go func() {
		if err := open.Run(uri); err != nil {
			log.Errorln("Failed to open URI:", err)
		}
	}()
}

// PangoAttrs is a way to declaratively create a pango.AttrList.
func PangoAttrs(attrs ...*pango.Attribute) *pango.AttrList {
	list := pango.NewAttrList()
	for _, attr := range attrs {
		list.Insert(attr)
	}
	return list
}
