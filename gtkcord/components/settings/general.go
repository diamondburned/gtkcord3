package settings

import (
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

type General struct {
	*handy.PreferencesPage `json:"-"`

	Customization Customization `json:"customization"`
}

func DefaultGeneral() *General {
	return &General{
		Customization: Customization{
			CSSFile:         "",
			HighlightScheme: "monokai",
		},
	}
}

func (p *General) Initialize() {
	p.PreferencesPage = handy.PreferencesPageNew()
	p.PreferencesPage.Show()
	p.PreferencesPage.SetIconName("preferences-system-symbolic")
	p.PreferencesPage.SetTitle("General")

	p.Customization.Initialize()
	p.PreferencesPage.Add(p.Customization)
}

type Customization struct {
	*handy.PreferencesGroup `json:"-"`

	// only 1 file since files can import others.
	CSSFile string `json:"css_files"`

	// https://xyproto.github.io/splash/docs/all.html
	HighlightScheme string `json:"highlight_scheme"`
}

func (p *Customization) Initialize() {
	p.PreferencesGroup = handy.PreferencesGroupNew()
	p.PreferencesGroup.Show()
	p.PreferencesGroup.SetTitle("Customization")

	// Permit only CSS files by MIME type.
	cssFilter, _ := gtk.FileFilterNew()
	cssFilter.SetName("CSS Files")
	cssFilter.AddMimeType("text/css")

	cfileW, _ := gtk.FileChooserButtonNew("Select CSS", gtk.FILE_CHOOSER_ACTION_OPEN)
	cfileW.AddFilter(cssFilter)
	bindFileChooser(cfileW, &p.CSSFile)

	p.PreferencesGroup.Add(row(
		"Custom CSS File",
		"Refer to the Gtk+ CSS Reference Manual for more information.",
		cfileW,
	))

	hlW, _ := gtk.EntryNew()
	bindEntry(hlW, &p.HighlightScheme)

	p.PreferencesGroup.Add(row(
		"Highlight color scheme",
		"Refer to https://xyproto.github.io/splash/docs/all.html",
		hlW,
	))
}
