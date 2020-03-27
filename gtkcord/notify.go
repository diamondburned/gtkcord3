package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/gotk3/gotk3/glib"
)

func (a *Application) bindNotifier() {
	a.State.AddHandler(a.onMessageCreate)
}

func (a *Application) onMessageCreate(create *gateway.MessageCreateEvent) {
	var msg = discord.Message(*create)

	if !a.State.MessageMentions(msg) {
		return
	}

	name := a.State.AuthorDisplayName(msg)

	notification := glib.NotificationNew(name + " mentioned you.")
	notification.SetBody(humanize.TrimString(msg.Content, 256))
	notification.SetPriority(glib.NOTIFICATION_PRIORITY_HIGH)

	// Set the click action to open the client:
	gtkutils.NSetDefaultActionAndTargetValue(
		notification,
		"app.load-channel", // actions.go
		glib.VariantFromInt64(int64(create.ChannelID)),
	)

	// Spawn:
	a.Application.SendNotification("mentioned", notification)
}
