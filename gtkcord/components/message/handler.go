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
	if m.channelID.Get() != t.ChannelID {
		return
	}

	m.guard.Lock()
	defer m.guard.Unlock()

	go m.Input.Typing.Add(t)
}

func (m *Messages) onMessageCreate(c *gateway.MessageCreateEvent) {
	if m.channelID.Get() != c.ChannelID {
		return
	}

	m.guard.Lock()
	defer m.guard.Unlock()

	m.insert((*discord.Message)(c))

	// Check typing
	go m.Input.Typing.Remove(c.Author.ID)
}

func (m *Messages) onMessageUpdate(u *gateway.MessageUpdateEvent) {
	if m.channelID.Get() != u.ChannelID {
		return
	}

	m.guard.RLock()
	defer m.guard.RUnlock()

	m.update((*discord.Message)(u))
}

func (m *Messages) onMessageDelete(d *gateway.MessageDeleteEvent) {
	if m.channelID.Get() != d.ChannelID {
		return
	}

	m.guard.Lock()
	defer m.guard.Unlock()

	m.delete(d.ID)
}

func (m *Messages) onMessageDeleteBulk(d *gateway.MessageDeleteBulkEvent) {
	if m.channelID.Get() != d.ChannelID {
		return
	}

	m.guard.Lock()
	defer m.guard.Unlock()

	m.delete(d.IDs...)
}

func (m *Messages) onGuildMembersChunk(c *gateway.GuildMembersChunkEvent) {
	if m.channelID.Get() != c.GuildID {
		return
	}

	m.guard.RLock()
	defer m.guard.RUnlock()

	m.updateMessageAuthor(c.Members...)
}
