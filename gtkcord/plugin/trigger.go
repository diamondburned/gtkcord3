package plugin

import "github.com/diamondburned/arikawa/gateway"

func TriggerTypingStart(t *gateway.TypingStartEvent) {
	for _,pl := range Plugins {
		pl.TypingStart(t)
	}
}