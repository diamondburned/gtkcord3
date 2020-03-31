package gtkcord

import (
	"fmt"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils/gdbus"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/gtkcord3/internal/log"
)

func (a *Application) bindNotifier() {
	a.State.AddHandler(a.onMessageCreate)
	a.MPRIS.OnMetadata = a.onMetadataChange
}

func (a *Application) onMessageCreate(create *gateway.MessageCreateEvent) {
	var msg = (*discord.Message)(create)

	if !a.State.MessageMentions(*msg) {
		return
	}

	var (
		title   = a.State.AuthorDisplayName(*msg) + " mentioned you"
		content = humanize.TrimString(msg.Content, 256)
		markup  = md.ParseToSimpleMarkupWithMessage([]byte(content), a.State.Store, msg)
	)

	if ch, _ := a.State.Store.Channel(msg.ChannelID); ch != nil {
		var suffix = " (#" + ch.Name + ")"

		if msg.GuildID.Valid() {
			if g, _ := a.State.Store.Guild(msg.GuildID); g != nil {
				suffix = " (#" + ch.Name + ", " + g.Name + ")"
			}
		}

		title += suffix
	}

	notification := gdbus.Notification{
		AppName: "gtkcord3",
		AppIcon: "user-available",
		Title:   title + ".",
		Message: string(markup),
		Actions: []*gdbus.Action{
			{
				ID:    "default",
				Label: "Open",
				Callback: func() {
					a.actionLoadChannel(nil, int64(create.ChannelID))
				},
			},
		},
	}

	if _, err := a.Notifier.Notify(notification); err != nil {
		log.Errorln("Failed to notify:", err)
	}
}

func (a *Application) onMetadataChange(m *gdbus.Metadata) {
	if a.State == nil {
		return
	}

	var status = discord.OnlineStatus

	p, err := a.State.Presence(0, a.State.Ready.User.ID)
	if err == nil {
		status = p.Status
	}

	var artist = humanize.Strings(m.Artists)

	data := gateway.UpdateStatusData{
		Status: status,
		AFK:    false,
		Activities: []discord.Activity{{
			Name:    artist,
			Type:    discord.ListeningActivity,
			Details: m.Title,
			State:   fmt.Sprintf("%s (%s)", artist, m.Album),
		}},
	}

	if err = a.State.Gateway.UpdateStatus(data); err != nil {
		log.Errorln("Failed to update status:", err)
	}
}
