package gtkutils

import (
	"html"
	"unsafe"

	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/skratchdot/open-golang/open"
)

type ExtendedWidget interface {
	gtk.IWidget
	StyleContextGetter

	SetSensitive(bool)
	GetSensitive() bool
	SetOpacity(float64)
	Hide()
	Show()
	ShowAll()
	Destroy()
	GrabFocus()
	SetSizeRequest(w, h int)
}

type WidgetDestroyer interface {
	gtk.IWidget
	Destroy()
}

type WidgetConnector interface {
	gtk.IWidget
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)
}

type WidgetSizeRequester interface {
	gtk.IWidget
	SetSizeRequest(w, h int)
	SetVExpand(bool)
	SetHExpand(bool)
}

type Namer interface {
	GetName() (string, error)
	SetName(string)
}

type Container interface {
	gtk.IWidget
	Add(gtk.IWidget)
	Remove(gtk.IWidget)
	GetChildren() *glib.List
}

type Object interface {
	GetProperty(string) (interface{}, error)
	GetPropertyType(string) (glib.Type, error)
	SetProperty(string, interface{}) error
}

type SizeRequester interface {
	SetSizeRequest(w, h int)
}

// Safe-guard
var _ ExtendedWidget = (*gtk.Box)(nil)
var _ WidgetDestroyer = (*gtk.Box)(nil)
var _ WidgetConnector = (*gtk.Box)(nil)
var _ Object = (*glib.Object)(nil)

type Marginator interface {
	SetMarginStart(int)
	SetMarginEnd(int)
	SetMarginTop(int)
	SetMarginBottom(int)

	GetMarginStart() int
	GetMarginEnd() int
	GetMarginTop() int
	GetMarginBottom() int
}

func Margin4(w Marginator, top, bottom, left, right int) {
	w.SetMarginTop(top)
	w.SetMarginBottom(bottom)
	w.SetMarginStart(left)
	w.SetMarginEnd(right)
}

func Margin2(w Marginator, top, left int) {
	Margin4(w, top, top, left, left)
}

func Margin(w Marginator, sz int) {
	Margin2(w, sz, sz)
}

func TransferMargin(dst, src Marginator) {
	dst.SetMarginBottom(src.GetMarginBottom())
	dst.SetMarginEnd(src.GetMarginEnd())
	dst.SetMarginStart(src.GetMarginStart())
	dst.SetMarginTop(src.GetMarginTop())
	// reset src to 0
	Margin(src, 0)
}

func AsContainer(w gtk.IWidget) *gtk.Container {
	widget := w.ToWidget()
	// Check the property that only Container has:
	if !HasProperty(widget, "border-width") {
		return nil
	}
	return &gtk.Container{Widget: *widget}
}

func NthChildren(container Container, i int) *gtk.Widget {
	list := container.GetChildren()
	if list == nil {
		return nil
	}
	v := list.NthData(0)
	if v == nil {
		return nil
	}
	if w, ok := v.(gtk.IWidget); ok {
		log.Println("NthChildren is widget")
		return w.ToWidget()
	}
	return &gtk.Widget{
		InitiallyUnowned: glib.InitiallyUnowned{
			Object: glib.Take(v.(unsafe.Pointer)),
		},
	}
}

func HasProperty(obj Object, name string) bool {
	t, err := obj.GetPropertyType(name)
	return err == nil && t != glib.TYPE_INVALID
}

// fn() == true => break
func TraverseWidget(container Container, fn func(*gtk.Widget)) {
	list := container.GetChildren()
	if list == nil {
		return
	}
	list.Foreach(func(v interface{}) {
		wd, ok := v.(gtk.IWidget)
		if !ok {
			return
		}

		fn(wd.ToWidget())

		// Recurse
		if c := AsContainer(wd); c != nil {
			TraverseWidget(c, fn)
		}
	})
}

type StyleContextGetter interface {
	GetStyleContext() (*gtk.StyleContext, error)
}

func WrapBox(orient gtk.Orientation, widgets ...gtk.IWidget) *gtk.Box {
	var b, _ = gtk.BoxNew(orient, 0)
	for _, w := range widgets {
		b.Add(w)
	}
	b.ShowAll()
	return b
}

func InjectCSSUnsafe(g StyleContextGetter, class, CSS string) {
	style, _ := g.GetStyleContext()

	if class != "" {
		style.AddClass(class)
	}

	if CSS != "" {
		AddCSSUnsafe(style, CSS)
	}
}

func InjectCSS(g StyleContextGetter, class, CSS string) {
	semaphore.IdleMust(InjectCSSUnsafe, g, class, CSS)
}

func AddCSSUnsafe(style *gtk.StyleContext, CSS string) {
	css, _ := gtk.CssProviderNew()
	if err := css.LoadFromData(CSS); err != nil {
		log.Errorln("Failed to load CSS:", err)
	}
	style.AddProvider(css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
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

type Connector interface {
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)
}

func Connect(connector Connector, event string, cb interface{}, data ...interface{}) {
	semaphore.IdleMust(func() {
		_, err := connector.Connect(event, cb, data...)
		if err != nil {
			log.Panicln("Failed to connect:", err)
		}
	})
}

func DiffClass(old *string, new string, style *gtk.StyleContext) {
	if *old == new {
		return
	}

	if *old != "" {
		semaphore.IdleMust(style.RemoveClass, *old)
	}

	*old = new

	if new == "" {
		return
	}

	semaphore.IdleMust(style.AddClass, new)
}

func DiffClassUnsafe(old *string, new string, style *gtk.StyleContext) {
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

func ImageSetIcon(img *gtk.Image, icon string, px int) {
	img.SetProperty("icon-name", icon)
	img.SetProperty("pixel-size", px)
}

func EventIsRightClick(ev *gdk.Event) bool {
	btn := gdk.EventButtonNewFromEvent(ev)
	return btn.Button() == gdk.BUTTON_SECONDARY
}
func EventIsLeftClick(ev *gdk.Event) bool {
	btn := gdk.EventButtonNewFromEvent(ev)
	return btn.Button() == gdk.BUTTON_PRIMARY
}

type Dialoger interface {
	GetContentArea() (*gtk.Box, error)
	GetHeaderBar() (gtk.IWidget, error)
	Remove(gtk.IWidget)
}

func HandyDialog(dialog Dialoger, transientFor gtk.IWindow) *handy.Dialog {
	w, _ := dialog.GetContentArea()
	dialog.Remove(w)

	h, _ := dialog.GetHeaderBar()
	dialog.Remove(h)

	d := handy.DialogNew(transientFor)
	d.Show()

	// Hack for close button
	d.Connect("response", func(_ *glib.Object, resp gtk.ResponseType) {
		if resp == gtk.RESPONSE_DELETE_EVENT {
			d.Destroy()
		}
	})

	// Delete the existing inner box:
	c, _ := d.GetContentArea()
	d.Remove(c)

	// Give the content box to our new dialog:
	d.Add(w)

	// Set the header:
	d.SetTitlebar(h)

	return d
}

func OpenURI(uri string) {
	go func() {
		if err := open.Run(uri); err != nil {
			log.Errorln("Failed to open URI:", err)
		}
	}()
}
