package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
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
		case *gateway.GuildUpdateEvent:
			onGuildUpdate(v)
		case *gateway.MessageDeleteBulkEvent:
			onMessageDeleteBulk(v)
		case *gateway.GuildMembersChunkEvent:
			onGuildMembersChunk(v)
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

func onGuildUpdate(g *gateway.GuildUpdateEvent) {
}
