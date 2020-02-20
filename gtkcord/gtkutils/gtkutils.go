package gtkutils

import (
	"html"

	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
)

type ExtendedWidget interface {
	gtk.IWidget
	SetSensitive(bool)
	GetSensitive() bool
	Show()
	ShowAll()
}

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
	css, _ := gtk.CssProviderNew()
	css.LoadFromData(CSS)

	style, _ := g.GetStyleContext()
	style.AddClass(class)
	style.AddProvider(css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}

func InjectCSS(g StyleContextGetter, class, CSS string) {
	semaphore.IdleMust(InjectCSSUnsafe, g, class, CSS)
}

func Escape(str string) string {
	return html.EscapeString(str)
}

func Bold(str string) string {
	return "<b>" + Escape(str) + "</b>"
}
