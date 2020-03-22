package main

import (
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
	"log"
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
	pb, err := gtk.ButtonNew()
	if err != nil {
		log.Println("Error spawning GTK button from plugin : ", err)
	}
	pb.SetLabel("plugin spawned this lol")
	pb.SetSizeRequest(128, 128)
	pb.ShowAll()
	return pb
}

func onTypingStart(t *gateway.TypingStartEvent) {
	log.Println("Typing start from plugin")
}
