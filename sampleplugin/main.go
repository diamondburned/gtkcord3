package main

import (
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord"
	"log"
)

type PluginHook struct {
	Application *gtkcord.Application
}

func (h PluginHook) Init(a *gtkcord.Application) {
	if a == nil {
		log.Println("bruh gtkcord is NIL!")
	}
	h.Application = a
	h.Application.State.AddHandler(onTypingStart)
}

func onTypingStart(t *gateway.TypingStartEvent) {
	Hook.Application.Header.Destroy()
}

var Hook = PluginHook{
	Application: nil,
}