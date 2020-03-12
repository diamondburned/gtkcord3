package message

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/log"
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
	if m.GetChannelID() != t.ChannelID {
		return
	}
	m.Typing.Add(t)
}

func (m *Messages) onMessageCreate(c *gateway.MessageCreateEvent) {
	if m.GetChannelID() != c.ChannelID {
		return
	}

	m.Insert((*discord.Message)(c))

	// Check typing
	m.Typing.Remove(c.Author.ID)
}

func (m *Messages) onMessageUpdate(u *gateway.MessageUpdateEvent) {
	if m.GetChannelID() != u.ChannelID {
		return
	}

	m.Update((*discord.Message)(u))
}

func (m *Messages) onMessageDelete(d *gateway.MessageDeleteEvent) {
	if m.GetChannelID() != d.ChannelID {
		return
	}

	m.Delete(d.ID)
}
func (m *Messages) onMessageDeleteBulk(d *gateway.MessageDeleteBulkEvent) {
	if m.GetChannelID() != d.ChannelID {
		return
	}

	m.Delete(d.IDs...)
}

func (m *Messages) onGuildMembersChunk(c *gateway.GuildMembersChunkEvent) {
	if m.GetGuildID() != c.GuildID {
		log.Println("GuildMembersChunk not from our guild", m.GuildID)
		return
	}

	m.UpdateMessageAuthor(c.Members...)
}
