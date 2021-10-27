package main

import (
	"log"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord"
	
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

var (
	Name   = "Example plugin"
	Author = "Bluskript"
)

func Ready(a *gtkcord.Application) {
	a.State.AddHandler(onTypingStart)
	BAD_SEMAPHORE_CALL(func() {
		a.Channels.Main.Add(pluginButton())
	})
}

func pluginButton() *gtk.Button {
	// Gtk errors can be ignored, things will panic on their own anyway.
	pb := gtk.NewButton()
	pb.SetLabel("BOTTOM TEXT")
	pb.SetSizeRequest(128, 128)
	pb.Show()
	return pb
}

func onTypingStart(t *gateway.TypingStartEvent) {
	log.Println("Typing start from plugin")
}
