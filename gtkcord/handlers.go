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
			a.onMessageCreate(v)
		case *gateway.MessageUpdateEvent:
			a.onMessageUpdate(v)
		case *gateway.MessageDeleteEvent:
			a.onMessageDelete(v)
		case *gateway.MessageDeleteBulkEvent:
			a.onMessageDeleteBulk(v)
		}
	})
}

func (a *application) onMessageCreate(m *gateway.MessageCreateEvent) {
	mw, ok := a.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	if err := mw.Insert(a.State, a.parser, discord.Message(*m)); err != nil {
		logWrap(err, "Failed to insert message from "+m.Author.Username)
	}
}

func (a *application) onMessageUpdate(m *gateway.MessageUpdateEvent) {
	mw, ok := a.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	mw.Update(a.State, a.parser, discord.Message(*m))
}

func (a *application) onMessageDelete(m *gateway.MessageDeleteEvent) {
	mw, ok := a.Messages.(*Messages)
	if !ok {
		return
	}

	if m.ChannelID != mw.Channel.ID {
		return
	}

	mw.Delete(m.ID)
}

func (a *application) onMessageDeleteBulk(m *gateway.MessageDeleteBulkEvent) {
	mw, ok := a.Messages.(*Messages)
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
