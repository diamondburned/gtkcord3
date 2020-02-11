package main

import (
	"context"
	"log"
	"os"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord"
)

func main() {
	var token = os.Getenv("TOKEN")
	if token == "" {
		log.Fatalln("No tokens given!")
	}

	a, err := gtkcord.New()
	if err != nil {
		log.Fatalln("Can't create a Gtk3 window:", err)
	}

	s, err := state.New(token)
	if err != nil {
		log.Fatalln("Can't create a Discord state:", err)
	}

	s.ErrorLog = func(err error) {
		log.Println("State error:", err)
	}

	if err := s.Open(); err != nil {
		log.Fatalln("Can't connect to Discord:", err)
	}

	s.WaitFor(context.Background(), func(v interface{}) bool {
		_, ok := v.(*gateway.ReadyEvent)
		return ok
	})

	if err := a.UseState(s); err != nil {
		log.Fatalln("Can't initiate the Gtk3 window:", err)
	}
}
