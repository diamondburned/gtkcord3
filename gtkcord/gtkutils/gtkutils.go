package gtkutils

import (
	"html"
	"sync"

	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type ExtendedWidget interface {
	gtk.IWidget
	StyleContextGetter

	SetSensitive(bool)
	GetSensitive() bool
	SetOpacity(float64)
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

type Container interface {
	gtk.IWidget
	Add(gtk.IWidget)
	Remove(gtk.IWidget)
}

// Safe-guard
var _ ExtendedWidget = (*gtk.Box)(nil)
var _ WidgetDestroyer = (*gtk.Box)(nil)
var _ WidgetConnector = (*gtk.Box)(nil)

type Marginator interface {
	SetMarginStart(int)
	SetMarginEnd(int)
	SetMarginTop(int)
	SetMarginBottom(int)
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

type StyleContextGetter interface {
	GetStyleContext() (*gtk.StyleContext, error)
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
	css.LoadFromData(CSS)
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

var connectMutex sync.Mutex

type connector interface {
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)
}

func Connect(connector connector, event string, cb interface{}, data ...interface{}) {
	connectMutex.Lock()
	defer connectMutex.Unlock()

	_, err := connector.Connect(event, cb, data...)
	if err != nil {
		log.Panicln("Failed to connect:", err)
	}
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
	GetHeaderBar() *gtk.Widget
	Remove(gtk.IWidget)
}

func HandyDialog(dialog Dialoger, transientFor gtk.IWindow) *handy.Dialog {
	w, _ := dialog.GetContentArea()
	dialog.Remove(w)

	h := dialog.GetHeaderBar()
	dialog.Remove(h)

	d := handy.DialogNew(transientFor)
	d.Show()

	// Hack for close button
	d.Connect("response", func(_ *glib.Object, resp gtk.ResponseType) {
		if resp == gtk.RESPONSE_DELETE_EVENT {
			d.Hide()
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
