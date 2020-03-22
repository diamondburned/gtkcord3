package main

import (
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

type PluginHook struct {
	window *gtk.Window
}

func (h PluginHook) SetWindow(window *gtk.Window) {
	h.window = window
}

func (h PluginHook) TypingStart(t *gateway.TypingStartEvent) {
	log.Printf("some dude with id %v is typing in channel %v", t.UserID, t.ChannelID)
	h.window.Fullscreen()
}

var Hook = PluginHook{
	window:nil,
}