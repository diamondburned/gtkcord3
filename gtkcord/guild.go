package gtkcord

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

func (a *Application) SwitchGuild(g *guild.Guild) {
	a.changeChannelCol(a.Channels, func() bool {
		err := a.Channels.LoadGuild(g.ID)
		if err != nil {
			log.Errorln("Failed to load guild:", err)
		}
		return err == nil
	})
}

func (a *Application) SwitchDM() {
	a.changeChannelCol(a.Privates, func() bool {
		a.Privates.LoadChannels(a.State, a.State.Gateway.Ready)
		return true
	})
}

func (a *Application) changeChannelCol(w gtk.IWidget, fn func() bool) {
	// Lock
	a.busy.Lock()
	defer a.busy.Unlock()

	// Clean up channels
	a.Channels.Cleanup()
	a.Privates.Cleanup()

	// Blur the grid
	async(a.Grid.SetSensitive, false)
	defer async(a.Grid.SetSensitive, true)

	// Add a spinner here
	var spinner gtk.IWidget
	must(func() {
		a.Grid.Remove(w)
		spinner, _ = animations.NewSpinner(SpinnerSize)
		a.setCol(spinner, 2)
	})

	if !fn() {
		a.Grid.Remove(spinner)
		return
	}

	// Replace the spinner with the actual channel:
	must(func() {
		a.Grid.Remove(spinner)
		a.setCol(w, 2)
	})
}

func (a *Application) SwitchChannel(ch *channel.Channel) {
	// Lock
	a.busy.Lock()
	defer a.busy.Unlock()

	// Clean up messages
	a.Messages.Cleanup()

	// Blur the grid
	async(a.Grid.SetSensitive, false)
	defer async(a.Grid.SetSensitive, true)

	// Add a spinner here
	var spinner gtk.IWidget
	must(func() {
		a.Grid.Remove(w)
		spinner, _ = animations.NewSpinner(SpinnerSize)
		a.setCol(spinner, 4)
	})

	if err := a.Messages.Load(ch.ID); err != nil {
		log.Errorln("Failed to load messages:", err)

		a.Grid.Remove(spinner)
		return
	}

	must(func() {
		a.Grid.Remove(spinner)
		a.setCol(w, 2)
	})
}
