package gtkcord

import (
	"github.com/diamondburned/arikawa/v2/discord"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gtkcord3/internal/log"
)

func (a *Application) bindActions() {
	a.Application.AddAction(newAction("load-channel", "x", func(v *glib.Variant) {
		chID := discord.ChannelID(v.Int64())

		ch, err := a.State.Cabinet.Channel(chID)
		if err != nil {
			log.Errorln("can't find channel", chID)
			return
		}

		a.SwitchToID(ch.ID, ch.GuildID)
	}))
}

func newAction(name, typ string, fn func(v *glib.Variant)) *gio.SimpleAction {
	action := gio.NewSimpleAction(name, glib.NewVariantType(typ))
	action.ConnectActivate(fn)
	return action
}
