package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/message"
	"github.com/diamondburned/gtkcord3/log"
)

func (a *application) hookEvents() {
	a.State.AddHandler(func(v interface{}) {
		a.busy.RLock()
		defer a.busy.RUnlock()

		// TODO: presence update

		switch v := v.(type) {
		case *gateway.TypingStartEvent:
			onTypingStart(v)
		case *gateway.MessageCreateEvent:
			onMessageCreate(v)
		case *gateway.MessageUpdateEvent:
			onMessageUpdate(v)
		case *gateway.MessageDeleteEvent:
			onMessageDelete(v)
		case *gateway.GuildCreateEvent:
			onGuildCreate(v)
		case *gateway.GuildUpdateEvent:
			onGuildUpdate(v)
		case *gateway.MessageDeleteBulkEvent:
			onMessageDeleteBulk(v)
		case *gateway.GuildMembersChunkEvent:
			onGuildMembersChunk(v)
		case *gateway.SessionsReplaceEvent:
			onSessionsReplace(*v)
		case *gateway.UserSettingsUpdateEvent:
			onUserSettingsUpdate(v)
		}
	})
}

func onTypingStart(t *gateway.TypingStartEvent) {
	mw, ok := App.Messages.(*message.Messages)
	if !ok {
		return
	}

	if t.ChannelID != mw.ChannelID {
		return
	}

	mw.Typing.Add(t)
}

func onMessageCreate(m *gateway.MessageCreateEvent) {
	mw, ok := App.Messages.(*message.Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.ChannelID {
		return
	}

	go func() {
		if err := mw.Insert(discord.Message(*m)); err != nil {
			logWrap(err, "Failed to insert message from "+m.Author.Username)
			return
		}

		// Check typing
		mw.Typing.Remove(m.Author.ID)
	}()
}

func onMessageUpdate(m *gateway.MessageUpdateEvent) {
	mw, ok := App.Messages.(*message.Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.ChannelID {
		return
	}

	go mw.Update(discord.Message(*m))
}

func onMessageDelete(m *gateway.MessageDeleteEvent) {
	mw, ok := App.Messages.(*message.Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.ChannelID {
		return
	}

	go mw.Delete(m.ID)
}

func onMessageDeleteBulk(m *gateway.MessageDeleteBulkEvent) {
	mw, ok := App.Messages.(*message.Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.ChannelID {
		return
	}

	go mw.Delete(m.IDs...)
}

func onGuildMembersChunk(c *gateway.GuildMembersChunkEvent) {
	mw, ok := App.Messages.(*message.Messages)
	if !ok {
		return
	}

	if c.GuildID != mw.GuildID {
		return
	}

	for _, guild := range App.Guilds.Guilds {
		if guild.ID != mw.GuildID {
			continue
		}

		go func() {
			for _, m := range c.Members {
				App.MessageNew.RequestedMember(guild.ID, m.User.ID)
			}
		}()

		break
	}

	go func() {
		for _, m := range c.Members {
			mw.UpdateMessageAuthor(m)
		}
	}()
}

func onSessionsReplace(replaces gateway.SessionsReplaceEvent) {
	if len(replaces) == 0 {
		return
	}
	replace := replaces[0]

	App.Header.Hamburger.User.UpdateStatus(replace.Status)
	App.Header.Hamburger.User.UpdateActivity(replace.Game)
}

func onUserSettingsUpdate(r *gateway.UserSettingsUpdateEvent) {
	App.Header.Hamburger.User.UpdateStatus(r.Status)
}

func onGuildCreate(g *gateway.GuildCreateEvent) {
	var target *Guild

Find:
	for _, guild := range App.Guilds.Guilds {
		if guild.ID == g.ID {
			target = guild
			break Find
		}

		if guild.Folder != nil {
			for _, guild := range guild.Folder.Guilds {
				if guild.ID == g.ID {
					target = guild
					break Find
				}
			}
		}
	}

	if target == nil {
		log.Errorln("Couldn't find guild with ID", g.ID, "for update.")
		return
	}

	must(target.SetUnavailable, g.Unavailable)

	if target.Name != g.Name {
		must(target.Row.SetTooltipMarkup, bold(g.Name))
	}
	if url := g.IconURL(); target.IURL != url {
		target.IURL = url
		target.UpdateImage()
	}
}

func onGuildUpdate(g *gateway.GuildUpdateEvent) {
	var target *Guild

Find:
	for _, guild := range App.Guilds.Guilds {
		if guild.ID == g.ID {
			target = guild
			break Find
		}

		if guild.Folder != nil {
			for _, guild := range guild.Folder.Guilds {
				if guild.ID == g.ID {
					target = guild
					break Find
				}
			}
		}
	}

	// TODO: presences

	if target == nil {
		log.Errorln("Couldn't find guild with ID", g.ID, "for update.")
		return
	}

	if target.Name != g.Name {
		must(target.Row.SetTooltipMarkup, bold(g.Name))
	}
	if url := discord.Guild(*g).IconURL(); target.IURL != url {
		target.IURL = url
		target.UpdateImage()
	}
}
