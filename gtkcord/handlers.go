package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
)

func (a *application) hookEvents() {
	a.State.AddHandler(func(v interface{}) {
		a.busy.Lock()
		defer a.busy.Unlock()

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

	if err := mw.Insert(discord.Message(*m)); err != nil {
		logWrap(err, "Failed to insert message from "+m.Author.Username)
	}
}

func onMessageUpdate(m *gateway.MessageUpdateEvent) {
	mw, ok := App.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	mw.Update(discord.Message(*m))
}

func onMessageDelete(m *gateway.MessageDeleteEvent) {
	mw, ok := App.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	mw.Delete(m.ID)
}

func onMessageDeleteBulk(m *gateway.MessageDeleteBulkEvent) {
	mw, ok := App.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	for _, id := range m.IDs {
		mw.Delete(id)
	}
}

func onGuildUpdate(g *gateway.GuildUpdateEvent) {
}
