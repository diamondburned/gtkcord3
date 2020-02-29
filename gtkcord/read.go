package gtkcord

import (
	"github.com/diamondburned/arikawa/gateway"
)

func (a *application) hookReads() {
	a.State.OnReadChange = func(rs *gateway.ReadState, ack bool) {
		a.Guilds.traverseReadState(rs, ack)
	}
}

func (guilds *Guilds) traverseReadState(rs *gateway.ReadState, ack bool) {
	var guild *Guild

	ch, err := App.State.Channel(rs.ChannelID)
	if err == nil && ch.GuildID.Valid() {
		guild, _ = guilds.findByID(ch.GuildID)
	}

	if guild == nil {
		guild, _ = guilds.find(func(g *Guild) bool {
			if g.Channels == nil {
				return false
			}

			for _, ch := range g.Channels.Channels {
				if ch.ID == rs.ChannelID {
					return true
				}
			}

			return false
		})
	}

	if guild == nil {
		return
	}

	guild.setUnread(!ack)

	if guild.Channels == nil {
		return
	}

	guild.Channels.traverseReadState(rs, ack)
}

func (guild *Guild) setUnread(unread bool) {
	if App.State.GuildMuted(guild.ID, false) {
		return
	}

	if guild.Channels != nil {
		for _, ch := range guild.Channels.Channels {
			// Category mute is very special. It doesn't count towards guild
			// unread, but it should still be highlighted.
			if ch.unread && !App.State.CategoryMuted(ch.ID) {
				unread = true
				break
			}
		}
	}

	if guild.unread == unread {
		return
	}
	guild.unread = unread

	if unread {
		must(guild.Style.AddClass, "unread")
	} else {
		must(guild.Style.RemoveClass, "unread")
	}

	if guild.Parent != nil {
		for _, guild := range guild.Parent.Folder.Guilds {
			if guild.unread {
				guild.Parent.setUnread(true)
				return
			}
		}

		guild.Parent.setUnread(false)
	}
}

func (channels *Channels) traverseReadState(rs *gateway.ReadState, ack bool) {
	if App.ChannelID() == rs.ChannelID {
		return
	}

	for _, ch := range channels.Channels {
		if ch.ID != rs.ChannelID {
			continue
		}

		if ch.Channels == nil {
			ch.Channels = channels
		}

		// ack == read
		ch.setUnread(!ack)
	}
}

func (channel *Channel) updateReadState(rs *gateway.ReadState) {
	if rs == nil {
		channel.setUnread(false)
		return
	}

	unread := channel.LastMsg != rs.LastMessageID
	channel.setUnread(unread)

	if channel.Channels != nil && App.Guild != channel.Channels.Guild {
		channel.Channels.Guild.setUnread(unread)
	}
}

func (channel *Channel) setUnread(unread bool) {
	if App.State.ChannelMuted(channel.ID) {
		channel.setOpacity(0.25)
		channel.unread = false
		return
	}

	if channel.unread == unread {
		return
	}
	channel.unread = unread

	if unread {
		channel.setOpacity(1)
	} else {
		channel.setOpacity(0.5)
	}
}
