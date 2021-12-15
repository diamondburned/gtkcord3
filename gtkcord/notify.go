package gtkcord

import (
	"context"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils/gdbus"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
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

	state := a.State
	go func() {
		if !shouldPing(state) {
			return
		}

		title := state.AuthorDisplayName(create)
		if !create.GuildID.IsValid() {
			title += " sent you a message"
		} else {
			title += " mentioned you"
		}

		markup := md.ParseToSimpleMarkupWithMessage(
			[]byte(humanize.TrimString(create.Content, 256)),
			state.Cabinet, &create.Message,
		)

		if ch, _ := state.Channel(create.ChannelID); ch != nil && ch.Name != "" {
			suffix := " (#" + ch.Name + ")"

			if create.GuildID.IsValid() {
				if g, _ := state.Guild(create.GuildID); g != nil {
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
					ID:       "default",
					Label:    "Open",
					Callback: func() { a.SwitchToID(create.ChannelID, create.GuildID) },
				},
			},
		}

		if _, err := a.Notifier.Notify(notification); err != nil {
			log.Errorln("Failed to notify:", err)
		}
	}()
}

func shouldPing(s *ningen.State) bool {
	u, err := s.Me()
	if err != nil {
		return false
	}

	p, err := s.Presence(0, u.ID)
	if err != nil {
		return true // assume online
	}

	return p.Status == gateway.OnlineStatus || p.Status == gateway.IdleStatus
}

type mprisState struct {
	cancel context.CancelFunc
}

func newMPRISState() *mprisState {
	return &mprisState{}
}

func (s *mprisState) newContext() context.Context {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	return ctx
}

func (a *Application) onMPRISEvent(m gdbus.Metadata, playing bool) {
	if a.State == nil || a.MPRIS == nil || a.mprisState == nil {
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
	ctx := a.mprisState.newContext()

	go func() {
		status := gateway.OnlineStatus

		p, err := a.State.WithContext(ctx).Presence(0, uID)
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

		if err = a.State.Gateway.UpdateStatusCtx(ctx, data); err != nil {
			log.Errorln("Failed to update status:", err)
		}
	}()
}
