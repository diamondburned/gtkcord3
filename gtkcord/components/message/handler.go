package message

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
)

func (m *Messages) injectHandlers() {
	m.c.AddHandler(m.onTypingStart)
	m.c.AddHandler(m.onMessageCreate)
	m.c.AddHandler(m.onMessageUpdate)
	m.c.AddHandler(m.onMessageDelete)
	m.c.AddHandler(m.onMessageDeleteBulk)
	m.c.AddHandler(m.onGuildMembersChunk)
}

func (m *Messages) onTypingStart(t *gateway.TypingStartEvent) {
	m.guard.Lock()
	defer m.guard.Unlock()

	if m.ChannelID != t.ChannelID {
		return
	}

	m.Typing.Add(t)
}

func (m *Messages) onMessageCreate(c *gateway.MessageCreateEvent) {
	m.guard.Lock()
	defer m.guard.Unlock()

	if m.ChannelID != c.ChannelID {
		return
	}

	m.insert((*discord.Message)(c))

	// Check typing
	m.Typing.Remove(c.Author.ID)
}

func (m *Messages) onMessageUpdate(u *gateway.MessageUpdateEvent) {
	m.guard.Lock()
	defer m.guard.Unlock()

	if m.ChannelID != u.ChannelID {
		return
	}

	m.update((*discord.Message)(u))
}

func (m *Messages) onMessageDelete(d *gateway.MessageDeleteEvent) {
	m.guard.Lock()
	defer m.guard.Unlock()

	if m.ChannelID != d.ChannelID {
		return
	}

	m.delete(d.ID)
}
func (m *Messages) onMessageDeleteBulk(d *gateway.MessageDeleteBulkEvent) {
	m.guard.Lock()
	defer m.guard.Unlock()

	if m.ChannelID != d.ChannelID {
		return
	}

	m.delete(d.IDs...)
}

func (m *Messages) onGuildMembersChunk(c *gateway.GuildMembersChunkEvent) {
	m.guard.Lock()
	defer m.guard.Unlock()

	if m.ChannelID != c.GuildID {
		return
	}

	m.updateMessageAuthor(c.Members...)
}
