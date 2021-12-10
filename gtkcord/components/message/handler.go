package message

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

func (m *Messages) injectHandlers() {
	if m.Realized() {
		panic("injectHandler call on realized")
	}

	var cancel func()
	m.ConnectRealize(func() {
		cancel = m.c.AddHandler(func(v interface{}) {
			glib.IdleAdd(func() {
				switch v := v.(type) {
				case *gateway.TypingStartEvent:
					m.onTypingStart(v)
				case *gateway.MessageCreateEvent:
					m.onMessageCreate(v)
				case *gateway.MessageUpdateEvent:
					m.onMessageUpdate(v)
				case *gateway.MessageDeleteEvent:
					m.onMessageDelete(v)
				case *gateway.MessageDeleteBulkEvent:
					m.onMessageDeleteBulk(v)
				case *gateway.GuildMemberAddEvent:
					m.onGuildMemberAdd(v)
				case *gateway.GuildMembersChunkEvent:
					m.onGuildMembersChunk(v)
				case *gateway.MessageReactionAddEvent:
					m.react(v)
				case *gateway.MessageReactionRemoveEvent:
					m.unreact(v)
				case *gateway.MessageReactionRemoveAllEvent:
					m.unreactAll(v)
				}
			})
		})
	})
	m.ConnectUnrealize(func() {
		cancel()
		cancel = nil
	})
}

func (m *Messages) Find(id discord.MessageID) *Message { return m.find(id) }

func (m *Messages) find(id discord.MessageID) *Message {
	for _, message := range m.messages {
		if message.ID != id {
			continue
		}
		return message
	}
	return nil
}

func (m *Messages) onTypingStart(t *gateway.TypingStartEvent) {
	if m.channelID != t.ChannelID {
		return
	}

	m.Input.Typing.Add(t)
}

func (m *Messages) onMessageCreate(c *gateway.MessageCreateEvent) {
	if m.channelID != c.ChannelID {
		return
	}

	m.Upsert(&c.Message)

	// Check typing
	m.Input.Typing.Remove(c.Author.ID)
}

func (m *Messages) onMessageUpdate(u *gateway.MessageUpdateEvent) {
	if m.channelID != u.ChannelID {
		return
	}

	m.Update(&u.Message)
}

func (m *Messages) onMessageDelete(d *gateway.MessageDeleteEvent) {
	if m.channelID != d.ChannelID {
		return
	}

	m.Delete(d.ID)
}

func (m *Messages) onMessageDeleteBulk(d *gateway.MessageDeleteBulkEvent) {
	if m.channelID != d.ChannelID {
		return
	}

	m.Delete(d.IDs...)
}

func (m *Messages) onGuildMemberAdd(a *gateway.GuildMemberAddEvent) {
	if m.guildID != a.GuildID {
		return
	}

	for _, message := range m.messages {
		if message.AuthorID != a.User.ID {
			continue
		}
		message.UpdateMember(m.c, m.guildID, a.Member)
	}
}

func (m *Messages) onGuildMembersChunk(c *gateway.GuildMembersChunkEvent) {
	if m.guildID != c.GuildID {
		return
	}

	for _, n := range c.Members {
		for _, message := range m.messages {
			if message.AuthorID != n.User.ID {
				continue
			}
			message.UpdateMember(m.c, m.guildID, n)
		}
	}
}

func (m *Messages) react(r *gateway.MessageReactionAddEvent) {
	if m.channelID != r.ChannelID {
		return
	}

	if msg := m.find(r.MessageID); msg != nil {
		msg.reactions.ReactAdd(r)
	}
}

func (m *Messages) unreact(r *gateway.MessageReactionRemoveEvent) {
	if m.channelID != r.ChannelID {
		return
	}

	if msg := m.find(r.MessageID); msg != nil {
		msg.reactions.ReactRemove(r)
	}
}

func (m *Messages) unreactEmoji(r *gateway.MessageReactionRemoveEmojiEvent) {
	if m.channelID != r.ChannelID {
		return
	}

	if msg := m.find(r.MessageID); msg != nil {
		msg.reactions.RemoveEmoji(r.Emoji)
	}
}

func (m *Messages) unreactAll(r *gateway.MessageReactionRemoveAllEvent) {
	if m.channelID != r.ChannelID {
		return
	}

	if msg := m.find(r.MessageID); msg != nil {
		msg.reactions.RemoveAll()
	}
}
