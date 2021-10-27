package gtkcord

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils/gdbus"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/gtkcord3/internal/log"
)

func (a *Application) bindNotifier() {
	a.MPRIS.OnPlayback = a.onMPRISEvent
	a.State.AddHandler(func(create *gateway.MessageCreateEvent) {
		// Check if the message should trigger a mention.
		if !a.State.MessageMentions(create.Message) {
			return
		}

		glib.IdleAdd(func() { a.onMessageCreate(create) })
	})
}

func (a *Application) onMessageCreate(create *gateway.MessageCreateEvent) {
	// Ignore mentions from the current channel.
	if a.Messages != nil && a.Messages.ChannelID() == create.ChannelID {
		return
	}

	go func() {
		title := a.State.AuthorDisplayName(create) + " mentioned you"
		content := humanize.TrimString(create.Content, 256)
		markup := md.ParseToSimpleMarkupWithMessage(
			[]byte(content), a.State.Cabinet, &create.Message,
		)

		if ch, _ := a.State.Channel(create.ChannelID); ch != nil && ch.Name != "" {
			suffix := " (#" + ch.Name + ")"

			if create.GuildID.IsValid() {
				if g, _ := a.State.Guild(create.GuildID); g != nil {
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
						a.SwitchToID(create.ChannelID, create.GuildID)
					},
				},
			},
		}

		if _, err := a.Notifier.Notify(notification); err != nil {
			log.Errorln("Failed to notify:", err)
		}
	}()
}

func (a *Application) onMPRISEvent(m gdbus.Metadata, playing bool) {
	if a.State == nil {
		return
	}

	if playing && m.Title == "" {
		// Incomplete. Wait.
		return
	}

	// Are we playing?
	if !playing {
		log.Infof("paused music")
		a.updateStatus(nil)
		return // no.
	}

	// Yes. Update.
	log.Infof("playing %q by %q", m.Title, m.Artists)
	a.updateMetadata(m)
}

func (a *Application) updateMetadata(m gdbus.Metadata) {
	artist := humanize.Strings(m.Artists)
	state := artist
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
	me, _ := a.State.Me()
	uID := me.ID

	go func() {
		status := gateway.OnlineStatus

		p, err := a.State.Presence(0, uID)
		if err == nil {
			status = p.Status
		}

		data := gateway.UpdateStatusData{
			Status:     status,
			AFK:        false,
			Activities: []discord.Activity{},
		}

		if activity != nil {
			data.Activities = append(data.Activities, *activity)
		}

		if err = a.State.Gateway.UpdateStatus(data); err != nil {
			log.Errorln("Failed to update status:", err)
		}
	}()
}
