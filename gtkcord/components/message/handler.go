package message

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
)

func (m *Messages) injectHandlers() {
	m.c.AddHandler(m.onTypingStart)
	m.c.AddHandler(m.onMessageCreate)
	m.c.AddHandler(m.onMessageUpdate)
	m.c.AddHandler(m.onMessageDelete)
	m.c.AddHandler(m.onMessageDeleteBulk)
	m.c.AddHandler(m.onGuildMembersChunk)
	m.c.AddHandler(m.react)
	m.c.AddHandler(m.unreact)
}

func (m *Messages) find(id discord.Snowflake, found func(m *Message)) {
	m.guard.Lock()
	defer m.guard.Unlock()

	for _, message := range m.messages {
		if message.ID != id {
			continue
		}
		found(message)
		return
	}
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

	m.Upsert(&c.Message)

	// Check typing
	m.Input.Typing.Remove(c.Author.ID)
}

func (m *Messages) onMessageUpdate(u *gateway.MessageUpdateEvent) {
	if m.channelID.Get() != u.ChannelID {
		return
	}

	m.Update(&u.Message)
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
	if m.guildID.Get() != c.GuildID {
		return
	}

	guildID := m.guildID.Get()

	semaphore.IdleMust(func() {
		m.guard.RLock()
		defer m.guard.RUnlock()

		for _, n := range c.Members {
			for _, message := range m.messages {
				if message.AuthorID != n.User.ID {
					continue
				}
				message.updateMember(m.c, guildID, n)
			}
		}
	})
}

func (m *Messages) react(r *gateway.MessageReactionAddEvent) {
	if m.channelID.Get() != r.ChannelID {
		return
	}

	m.find(r.MessageID, func(m *Message) {
		if m.reactions == nil {
			return
		}
		m.reactions.ReactAdd(r)
	})
}

func (m *Messages) unreact(r *gateway.MessageReactionRemoveEvent) {
	if m.channelID.Get() != r.ChannelID {
		return
	}

	m.find(r.MessageID, func(m *Message) {
		m.reactions.ReactRemove(r)
	})
}

func (m *Messages) unreactEmoji(r *gateway.MessageReactionRemoveEmoji) {
	if m.channelID.Get() != r.ChannelID {
		return
	}

	m.find(r.MessageID, func(m *Message) {
		m.reactions.RemoveEmoji(r.Emoji)
	})
}

func (m *Messages) unreactAll(r *gateway.MessageReactionRemoveAllEvent) {
	if m.channelID.Get() != r.ChannelID {
		return
	}

	m.find(r.MessageID, func(m *Message) {
		m.reactions.RemoveAll()
	})
}
