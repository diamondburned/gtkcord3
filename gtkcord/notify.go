package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils/gdbus"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/gtkcord3/internal/log"
)

func (a *Application) bindNotifier() {
	a.State.AddHandler(a.onMessageCreate)
}

func (a *Application) onMessageCreate(create *gateway.MessageCreateEvent) {
	var msg = (*discord.Message)(create)

	if !a.State.MessageMentions(*msg) {
		return
	}

	var (
		name    = a.State.AuthorDisplayName(*msg)
		content = humanize.TrimString(msg.Content, 256)
		markup  = md.ParseToSimpleMarkupWithMessage([]byte(content), a.State.Store, msg)
	)

	notification := gdbus.Notification{
		AppName:   "gtkcord3",
		ReplaceID: gdbus.GetUUID("mentioned"),
		AppIcon:   "user-available",
		Title:     name + " mentioned you.",
		Message:   string(markup),
		Actions: [][2]string{
			{"default", "load-channel:" + create.ChannelID.String()},
		},
	}

	if err := a.DBusConn.Notify(notification); err != nil {
		log.Errorln("Failed to notify:", err)
	}
}
