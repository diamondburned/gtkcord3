package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/log"
)

func (a *application) hookEvents() {
	a.State.AddHandler(func(v interface{}) {
		a.busy.RLock()
		defer a.busy.RUnlock()

		switch v := v.(type) {
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
		}
	})
}

func onMessageCreate(m *gateway.MessageCreateEvent) {
	mw, ok := App.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	go func() {
		if err := mw.Insert(discord.Message(*m)); err != nil {
			logWrap(err, "Failed to insert message from "+m.Author.Username)
		}
	}()
}

func onMessageUpdate(m *gateway.MessageUpdateEvent) {
	mw, ok := App.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	go mw.Update(discord.Message(*m))
}

func onMessageDelete(m *gateway.MessageDeleteEvent) {
	mw, ok := App.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	go mw.Delete(m.ID)
}

func onMessageDeleteBulk(m *gateway.MessageDeleteBulkEvent) {
	mw, ok := App.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	go mw.Delete(m.IDs...)
}

func onGuildMembersChunk(c *gateway.GuildMembersChunkEvent) {
	mw, ok := App.Messages.(*Messages)
	if !ok {
		return
	}

	if c.GuildID != mw.Channel.Guild {
		return
	}

	if guild := mw.Channel.Channels.Guild; guild != nil {
		go func() {
			for _, m := range c.Members {
				guild.requestedMember(m.User.ID)
			}
		}()
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
