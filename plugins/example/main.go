package main

import (
	"log"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
)

var application *gtkcord.Application

func Ready(a *gtkcord.Application) {
	application = a
	a.State.AddHandler(onTypingStart)
	semaphore.IdleMust(func() {
		a.Channels.Main.Add(pluginButton())
	})
}

func pluginButton() *gtk.Button {
	// Gtk errors can be ignored, things will SEGFAULT on their own anyway.
	pb, _ := gtk.ButtonNew()
	pb.SetLabel("BOTTOM TEXT")
	pb.SetSizeRequest(128, 128)
	pb.Show()
	return pb
}

func onTypingStart(t *gateway.TypingStartEvent) {
	log.Println("Typing start from plugin")
}
