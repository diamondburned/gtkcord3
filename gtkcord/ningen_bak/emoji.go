package ningen

import "github.com/diamondburned/arikawa/v2/discord"

func EmojiString(e *discord.Emoji) string {
	if e == nil {
		return ""
	}

	var emoji = e.Name
	if e.ID.Valid() { // if the emoji is custom:
		emoji = ":" + emoji + ":"
	}

	return emoji
}
