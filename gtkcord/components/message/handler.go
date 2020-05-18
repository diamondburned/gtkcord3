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
	m.c.AddHandler(m.react)
	m.c.AddHandler(m.unreact)
	m.c.Members.OnMember(m.onGuildMember)
}

func (m *Messages) find(id discord.Snowflake, found func(m *Message)) {
	semaphore.IdleMust(func() {
		for _, message := range m.messages {
			if message.ID != id {
				continue
			}
			found(message)
			return
		}
	})
}

func (m *Messages) onTypingStart(t *gateway.TypingStartEvent) {
	if m.channelID.Get() != t.ChannelID {
		return
	}
	semaphore.IdleMust(func() {
		m.Input.Typing.Add(t)
	})
}

func (m *Messages) onMessageCreate(c *gateway.MessageCreateEvent) {
	if m.channelID.Get() != c.ChannelID {
		return
	}

	semaphore.IdleMust(func() {
		m.UpsertUnsafe(&c.Message)

		// Check typing
		m.Input.Typing.Remove(c.Author.ID)
	})
}

func (m *Messages) onMessageUpdate(u *gateway.MessageUpdateEvent) {
	if m.channelID.Get() != u.ChannelID {
		return
	}

	semaphore.IdleMust(func() {
		m.UpdateUnsafe(&u.Message)
	})
}

func (m *Messages) onMessageDelete(d *gateway.MessageDeleteEvent) {
	if m.channelID.Get() != d.ChannelID {
		return
	}

	semaphore.IdleMust(func() {
		m.deleteUnsafe(d.ID)
	})
}

func (m *Messages) onMessageDeleteBulk(d *gateway.MessageDeleteBulkEvent) {
	if m.channelID.Get() != d.ChannelID {
		return
	}

	semaphore.IdleMust(func() {
		m.deleteUnsafe(d.IDs...)
	})
}

func (m *Messages) onGuildMember(guildID discord.Snowflake, member discord.Member) {
	if m.guildID.Get() != guildID {
		return
	}

	semaphore.IdleMust(func() {
		for _, message := range m.messages {
			if message.AuthorID != member.User.ID {
				continue
			}
			message.updateMember(m.c, guildID, member)
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
