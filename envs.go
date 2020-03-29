package main

import (
	"os"

	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/internal/log"
)

func LoadEnvs() {
	if css := os.Getenv("GTKCORD_CUSTOM_CSS"); css != "" {
		window.CustomCSS = css
	}

	if os.Getenv("GTKCORD_QUIET") != "" {
		log.Quiet = true
	}
}
