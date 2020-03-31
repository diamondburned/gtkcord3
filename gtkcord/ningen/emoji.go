package ningen

import "github.com/diamondburned/arikawa/discord"

func EmojiString(e discord.Emoji) string {
	var emoji = e.Name
	if e.ID.Valid() { // if the emoji is custom:
		emoji = ":" + emoji + ":"
	}

	return emoji
}
