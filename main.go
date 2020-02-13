package main

import (
	"context"
	"os"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/log"
)

func main() {
	var token = os.Getenv("TOKEN")
	if token == "" {
		log.Fatalln("No tokens given!")
	}

	if err := gtkcord.Init(); err != nil {
		log.Fatalln("Can't create a Gtk3 window:", err)
	}

	s, err := state.New(token)
	if err != nil {
		log.Fatalln("Can't create a Discord state:", err)
	}

	s.ErrorLog = func(err error) {
		log.Errorln("Discord error:", err)
	}

	if err := s.Open(); err != nil {
		log.Fatalln("Can't connect to Discord:", err)
	}

	s.WaitFor(context.Background(), func(v interface{}) bool {
		_, ok := v.(*gateway.ReadyEvent)
		return ok
	})

	if err := gtkcord.UseState(s); err != nil {
		log.Fatalln("Can't initiate the Gtk3 window:", err)
	}
}
