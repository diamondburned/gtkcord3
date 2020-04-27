package ningen

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
)

func messageMentions(msg discord.Message, uID discord.Snowflake) bool {
	for _, user := range msg.Mentions {
		if user.ID == uID {
			return true
		}
	}
	return false
}

func (s *State) MessageMentions(msg discord.Message) bool {
	var mutedGuild *Mute

	// If there's guild:
	if msg.GuildID.Valid() {
		if mutedGuild = s.GetGuildMuted(msg.GuildID); mutedGuild != nil {
			// We're only checking mutes and suppressions, as channels don't
			// have these. Whatever channels have will override guilds.

			// @everyone mentions still work if the guild is muted and @everyone
			// is not suppressed.
			if msg.MentionEveryone && !mutedGuild.Everyone {
				return true
			}

			// TODO: roles

			// If the guild is muted of all messages:
			if mutedGuild.All {
				return false
			}
		}
	}

	// Boolean on whether the message contains a self mention or not:
	var mentioned = messageMentions(msg, s.Ready.User.ID)

	// Check channel settings. Channel settings override guilds.
	if mutedCh := s.GetChannelMuted(msg.ChannelID); mutedCh != nil {
		switch mutedCh.Notifications {
		case gateway.AllNotifications:
			// If the channel is muted.
			if mutedCh.All {
				return false
			}

		case gateway.NoNotifications:
			// If no notifications are allowed, not even mentions.
			return false

		case gateway.OnlyMentions:
			// If mentions are allowed. We return early because this overrides
			// the guild settings, even if Guild wants all messages.
			return mentioned
		}
	}

	if mutedGuild != nil {
		switch mutedGuild.Notifications {
		case gateway.AllNotifications:
			// If the guild is muted, but we can return early here. If we allow
			// all notifications, we can return the opposite of muted.
			//   - If we're muted, we don't want a mention.
			//   - If we're not muted, we want a mention.
			return !mutedGuild.All

		case gateway.NoNotifications:
			// If no notifications are allowed whatsoever.
			return false

		case gateway.OnlyMentions:
			// We can return early here.
			return mentioned
		}
	}

	// Is this from a DM? TODO: get a better check.
	if ch, err := s.Channel(msg.ChannelID); err == nil {
		// True if the message is from DM or group.
		return ch.Type == discord.DirectMessage || ch.Type == discord.GroupDM
	}

	return false
}
