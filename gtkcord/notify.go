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
	a.MPRIS.OnMetadata = a.onMPRISEvent
	a.MPRIS.OnPlaybackStatus = a.onMPRISEvent
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

func (a *Application) onMPRISEvent(m gdbus.Metadata, playing bool) {
	if a.State == nil {
		return
	}

	// Are we playing?
	if !playing || m.Title == "" {
		a.updateStatus(nil)
		return // no.
	}

	// Yes. Update.
	a.updateMetadata(m)
}

func (a *Application) updateMetadata(m gdbus.Metadata) {
	var artist = humanize.Strings(m.Artists)
	var state = artist
	if m.Album != "" {
		state += " (" + m.Album + ")"
	}

	a.updateStatus(&discord.Activity{
		Name:    artist,
		Type:    discord.ListeningActivity,
		Details: m.Title,
		State:   state,
	})
}

func (a *Application) updateStatus(activity *discord.Activity) {
	var status = discord.OnlineStatus

	p, err := a.State.Presence(0, a.State.Ready.User.ID)
	if err == nil {
		status = p.Status
	}

	data := gateway.UpdateStatusData{
		Status:     status,
		AFK:        false,
		Activities: new([]discord.Activity),
	}
	*data.Activities = []discord.Activity{} // This needs to be a [] in the JSON.

	// Because Discord is garbage.
	if activity != nil {
		*data.Activities = append(*data.Activities, *activity)
	}

	if err = a.State.Gateway.UpdateStatus(data); err != nil {
		log.Errorln("Failed to update status:", err)
	}
}
