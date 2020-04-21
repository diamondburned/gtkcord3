package window

import (
	"bytes"

	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/markbates/pkger"
)

var (
	CustomCSS string // raw CSS, once
	FileCSS   string // path
)

// I don't like this:
// list row:selected { box-shadow: inset 2px 0 0 0 white; }

func initCSS() {
	s := Window.Screen

	CSS := string(mustReadCSS())

	stock, _ := gtk.CssProviderNew()
	if err := stock.LoadFromData(CSS); err != nil {
		log.Fatalln("Failed to parse stock CSS:", err)
	}

	gtk.AddProviderForScreen(
		s, stock,
		uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
	)

	// Add env var CSS:
	env, _ := gtk.CssProviderNew()
	if err := env.LoadFromData(CustomCSS); err != nil {
		log.Errorln("Failed to parse env var custom CSS:", err)
	}

	gtk.AddProviderForScreen(
		s, env,
		uint(gtk.STYLE_PROVIDER_PRIORITY_USER),
	)
}

func ReloadCSS() {
	s := Window.Screen

	// Replace file CSS:
	if Window.fileCSS != nil {
		gtk.RemoveProviderForScreen(s, Window.fileCSS)
	}

	file, _ := gtk.CssProviderNew()
	if err := file.LoadFromPath(FileCSS); err != nil {
		log.Errorln("Failed to parse file in "+FileCSS+":", err)
		return
	}

	Window.fileCSS = file

	gtk.AddProviderForScreen(
		s, file,
		uint(gtk.STYLE_PROVIDER_PRIORITY_USER),
	)
}

func mustReadCSS() []byte {
	f, err := pkger.Open("/gtkcord/style.css")
	if err != nil {
		log.Panicln("Failed to open file:", err)
	}
	s, err := f.Stat()
	if err != nil {
		log.Panicln("Failed to stat file:", err)
	}
	var buf bytes.Buffer
	buf.Grow(int(s.Size()))

	if _, err := buf.ReadFrom(f); err != nil {
		log.Panicln("Failed to read file:", err)
	}

	return buf.Bytes()
}
