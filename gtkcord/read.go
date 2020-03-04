package gtkcord

import (
	"github.com/diamondburned/arikawa/gateway"
)

func (a *application) hookReads() {
	a.State.OnReadChange = a.Guilds.traverseReadState
}

func (guilds *Guilds) traverseReadState(rs *gateway.ReadState, ack bool) {
	var guild *Guild

	ch, err := App.State.Channel(rs.ChannelID)
	// maybe DM?
	if err == nil && !ch.GuildID.Valid() {
		App.Privates.traverseReadState(rs, ack)
		return
	}

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

	guild.setUnread(!ack, rs.MentionCount > 0)

	if guild.Channels == nil {
		return
	}

	guild.Channels.traverseReadState(rs, ack)
}

func (guild *Guild) setUnread(unread, pinged bool) {
	if App.State.GuildMuted(guild.ID, false) {
		return
	}

	if guild.Channels != nil && !unread {
		for _, ch := range guild.Channels.Channels {
			// Category mute is very special. It doesn't count towards guild
			// unread, but it should still be highlighted.
			if ch.unread && !App.State.CategoryMuted(ch.ID) {
				unread = true
				break
			}
		}
	}

	switch {
	case pinged:
		guild.setClass("pinged")
	case unread:
		guild.setClass("unread")
	default:
		guild.setClass("")
	}

	if guild.Parent != nil {
		for _, guild := range guild.Parent.Folder.Guilds {
			unread := guild.stateClass == "unread"
			pinged := guild.stateClass == "pinged"

			if unread || pinged {
				guild.Parent.setUnread(true, pinged)
				return
			}
		}

		guild.Parent.setUnread(false, false)
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
		ch.setUnread(!ack, rs.MentionCount > 0)
		break
	}
}

func (channel *Channel) updateReadState(rs *gateway.ReadState) {
	if rs == nil {
		channel.setUnread(false, false)
		return
	}

	unread := channel.LastMsg != rs.LastMessageID
	pinged := unread && rs.MentionCount > 0

	channel.setUnread(unread, pinged)

	if channel.Channels != nil && App.Guild != channel.Channels.Guild {
		channel.Channels.Guild.setUnread(unread, pinged)
	}
}

func (channel *Channel) setUnread(unread, pinged bool) {
	if App.State.ChannelMuted(channel.ID) {
		channel.setClass("muted")
		channel.unread = false
		return
	}

	channel.unread = unread

	if !unread && pinged {
		pinged = false
	}

	switch {
	case pinged:
		channel.setClass("pinged")
	case unread:
		channel.setClass("unread")
	default:
		channel.setClass("")
	}
}

func (pcs *PrivateChannels) traverseReadState(rs *gateway.ReadState, ack bool) {
	if App.ChannelID() == rs.ChannelID {
		ack = true
	}

	ch, ok := pcs.Channels[rs.ChannelID.String()]
	if !ok {
		return
	}

	ch.LastMsg = rs.LastMessageID
	ch.setUnread(!ack)
}

func (pc *PrivateChannel) updateReadState(rs *gateway.ReadState) {
	if rs == nil {
		pc.setUnread(false)
		return
	}

	unread := pc.LastMsg != rs.LastMessageID
	pc.setUnread(unread)

	if pc.Parent != nil {
		must(func() {
			if pc.ListBoxRow.GetIndex() != 0 {
				pc.Parent.List.InvalidateSort()
			}
		})
	}
}

func (pcs *PrivateChannels) setUnread(unread bool) {
	if !unread {
		for _, ch := range pcs.Channels {
			if ch.stateClass == "pinged" {
				unread = true
				break
			}
		}
	}

	if unread {
		pcs.setButtonClass("pinged")
	} else {
		pcs.setButtonClass("")
	}
}
func (pc *PrivateChannel) setUnread(unread bool) {
	if unread {
		pc.setClass("pinged")
	} else {
		pc.setClass("")
	}

	if pc.Parent != nil {
		pc.Parent.setUnread(unread)

		must(func() {
			if pc.ListBoxRow.GetIndex() == 0 {
				return
			}
			pc.Parent.List.InvalidateSort()
		})
	}
}
