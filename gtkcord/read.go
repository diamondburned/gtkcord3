package gtkcord

import (
	"github.com/diamondburned/arikawa/gateway"
)

func (a *application) hookReads() {
	a.State.OnReadUpdate = func(rs *gateway.ReadState) {
		a.Guilds.traverseReadState(rs)
	}
}

func (guilds *Guilds) traverseReadState(rs *gateway.ReadState) {
	var guild *Guild

	ch, err := App.State.Store.Channel(rs.ChannelID)
	if err == nil && ch.GuildID.Valid() {
		for _, g := range guilds.Guilds {
			if g.ID == ch.GuildID {
				guild = g
				break
			}
		}
	}

	if guild == nil {
	Main:
		for _, g := range guilds.Guilds {
			// We can skip this one, as channel constructors check for read
			// states.
			if g.Channels == nil {
				continue
			}

			for _, ch := range g.Channels.Channels {
				if ch.ID == rs.ChannelID {
					guild = g
					break Main
				}
			}
		}
	}

	if guild != nil {
		guild.updateReadState(rs)

		if guild.Channels != nil {
			guild.Channels.traverseReadState(rs)
		}
	}
}
func (guild *Guild) updateReadState(rs *gateway.ReadState) {
	must(guild.setUnread, true)
}

func (guild *Guild) setUnread(unread bool) {
	if unread {
		guild.SetOpacity(1)
	} else {
		guild.SetOpacity(0.5)
	}
}

func (channels *Channels) traverseReadState(rs *gateway.ReadState) {
	if App.ChannelID() == rs.ChannelID {
		return
	}

	for _, ch := range channels.Channels {
		if ch.ID != rs.ChannelID {
			continue
		}

		ch.updateReadState(rs)
		return
	}
}

func (channel *Channel) updateReadState(rs *gateway.ReadState) {
	if rs == nil {
		must(channel.setUnread, false)
		return
	}

	if channel.LastMsg != rs.LastMessageID {
		must(channel.setUnread, true)
	} else {
		must(channel.setUnread, false)
	}
}

func (channel *Channel) setUnread(unread bool) {
	if unread {
		channel.SetOpacity(1)
	} else {
		channel.SetOpacity(0.5)
	}
}
