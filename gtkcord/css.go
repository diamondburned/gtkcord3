package gtkcord

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const CSS = `
headerbar { padding: 0; }
headerbar button { box-shadow: none; }
textview, textview > text { background-color: transparent; }
`

var CustomCSS string

// I don't like this:
// list row:selected { box-shadow: inset 2px 0 0 0 white; }

func (a *Application) loadCSS() error {
	css, err := gtk.CssProviderNew()
	if err != nil {
		return errors.Wrap(err, "Failed to make a CSS provider")
	}

	if err := css.LoadFromData(CSS + CustomCSS); err != nil {
		return errors.Wrap(err, "Failed to parse CSS")
	}

	a.css = css

	d, err := gdk.DisplayGetDefault()
	if err != nil {
		return errors.Wrap(err, "Failed to get default GDK display")
	}
	s, err := d.GetDefaultScreen()
	if err != nil {
		return errors.Wrap(err, "Failed to get default screen")
	}

	gtk.AddProviderForScreen(s, css,
		uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION))

	return nil
}
