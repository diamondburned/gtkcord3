package main

import (
	"os"
	"strconv"

	"github.com/diamondburned/gtkcord3/gtkcord/components/message"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
)

func LoadEnvs() {
	if css := os.Getenv("GTKCORD_CUSTOM_CSS"); css != "" {
		window.CustomCSS = css
	}

	if w, _ := strconv.Atoi(os.Getenv("GTKCORD_MSGWIDTH")); w > 100 { // min 100
		message.MaxMessageWidth = w
	}
}
