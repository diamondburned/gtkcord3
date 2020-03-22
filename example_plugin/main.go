package main

import (
	"log"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
)

var application *gtkcord.Application

func Ready(a *gtkcord.Application) {
	application = a
	a.State.AddHandler(onTypingStart)
}

func onTypingStart(t *gateway.TypingStartEvent) {
	log.Println("Typing start from plugin")
	semaphore.IdleMust(application.Window.ApplicationWindow.Close)
}
