package window

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/internal/log"
)

var (
	ApplicationCSS string
	CustomCSS      string // raw CSS, once
	FileCSS        string // path
)

// I don't like this:
// list row:selected { box-shadow: inset 2px 0 0 0 white; }

func initCSS() {
	s := Window.Screen

	stock := gtk.NewCSSProvider()
	if err := stock.LoadFromData(ApplicationCSS); err != nil {
		log.Fatalln("failed to parse stock CSS:", err)
	}

	gtk.StyleContextAddProviderForScreen(
		s, stock,
		uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
	)

	// Add env var CSS:
	env := gtk.NewCSSProvider()
	if err := env.LoadFromData(CustomCSS); err != nil {
		log.Errorln("failed to parse env var custom CSS:", err)
	}

	gtk.StyleContextAddProviderForScreen(
		s, env,
		uint(gtk.STYLE_PROVIDER_PRIORITY_USER),
	)
}

func ReloadCSS() {
	s := Window.Screen

	// Replace file CSS:
	if Window.fileCSS != nil {
		gtk.StyleContextRemoveProviderForScreen(s, Window.fileCSS)
	}

	if FileCSS == "" {
		return
	}

	file := gtk.NewCSSProvider()
	if err := file.LoadFromPath(FileCSS); err != nil {
		log.Errorf("failed to parse file in %q: %v", FileCSS, err)
		return
	}

	Window.fileCSS = file

	gtk.StyleContextAddProviderForScreen(
		s, file,
		uint(gtk.STYLE_PROVIDER_PRIORITY_USER),
	)
}
