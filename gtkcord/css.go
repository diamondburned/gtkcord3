package gtkcord

import (
	"os"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const CSS = `
headerbar { padding: 0; }
headerbar button { box-shadow: none; }
textview, textview > text { background-color: transparent; }

.message:not(.condensed), .message-input {
	border-top: 1px solid rgba(0, 0, 0, 0.12);
}

.message-input {
	background-image: linear-gradient(transparent, rgba(10, 10, 10, 0.3));
}
.message-input button {
	background: none;
	box-shadow: none;
	border: none;
	opacity: 0.65;
}
.message-input button:hover {
	opacity: 1;
}

.guild image {
    box-shadow: 0px 0px 4px -1px rgba(0,0,0,0.5);
    border-radius: 50%;
	background-color: grey;
}
`

var CustomCSS = os.Getenv("GTKCORD_CUSTOM_CSS")

// I don't like this:
// list row:selected { box-shadow: inset 2px 0 0 0 white; }

func (a *application) loadCSS() error {
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
		uint(gtk.STYLE_PROVIDER_PRIORITY_USER))

	return nil
}

type StyleContextGetter interface {
	GetStyleContext() (*gtk.StyleContext, error)
}

func InjectCSS(g StyleContextGetter, class, CSS string) {
	css := must(gtk.CssProviderNew).(*gtk.CssProvider)
	must(css.LoadFromData, CSS)

	style := must(g.GetStyleContext).(*gtk.StyleContext)
	must(style.AddClass, class)
	must(style.AddProvider, css, uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION))
}
