package main

import (
	"os"
	"runtime"
	"strconv"

	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/logo"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/keyring"
	"github.com/diamondburned/gtkcord3/internal/log"

	_ "embed"

	// Profiler
	"net/http"
	_ "net/http/pprof"
)

var profile bool

//go:embed logo.png
var logoPNG []byte

func init() {
	glib.LogUseDefaultLogger()

	// flag.BoolVar(&profile, "prof", false, "Enable the profiler")
	logo.PNG = logoPNG

	// Set the right envs:
	if css := os.Getenv("GTKCORD_CUSTOM_CSS"); css != "" {
		window.CustomCSS = css
	}

	if w, _ := strconv.Atoi(os.Getenv("GTKCORD_MSGWIDTH")); w > 100 { // min 100
		variables.MaxMessageWidth = w
	}

	if os.Getenv("GTKCORD_QUIET") == "0" {
		log.Quiet = false
		profile = true
	}
}

func LoadKeyring() string {
	if token := os.Getenv("TOKEN"); token != "" {
		return token
	}

	return keyring.Get()
}

func main() {
	a := gtk.NewApplication("com.github.diamondburned.gtkcord3", 0)
	g := gtkcord.New(a)

	a.ConnectStartup(func() {
		handy.Init()
	})
	a.ConnectActivate(func() {
		g.Activate()
		g.ShowLogin(LoadKeyring())
	})

	a.Connect("shutdown", func() { g.Close() })

	if profile {
		runtime.SetBlockProfileRate(5000000) // 5ms
		go http.ListenAndServe("localhost:6969", nil)
	}

	if sig := a.Run(os.Args); sig > 0 {
		os.Exit(sig)
	}
}
