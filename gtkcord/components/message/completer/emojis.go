package completer

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/md"

	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/internal/log"
)

func (c *State) completeEmojis(word string) {
	if word == "" {
		return
	}

	guildID := c.container.GuildID()
	if !guildID.IsValid() {
		return
	}

	guildEmojis, err := c.state.EmojiState.Get(guildID)
	if err != nil {
		log.Errorln("failed to get emojis:", err)
		return
	}

	filtered := guildEmojis[:0]
	filteredLen := 0

	for _, guild := range guildEmojis {
		filteredEmojis := make([]discord.Emoji, 0, MaxCompletionEntries)

		for _, e := range guild.Emojis {
			if contains(e.Name, word) {
				filteredEmojis = append(filteredEmojis, e)
				filteredLen++

				if filteredLen > MaxCompletionEntries {
					break
				}
			}
		}

		guild.Emojis = filteredEmojis
		filtered = append(filtered, guild)

		if filteredLen > MaxCompletionEntries {
			break
		}
	}

	if len(filtered) == 0 {
		return
	}

	for _, guild := range filtered {
		for _, e := range guild.Emojis {
			b := gtk.NewBox(gtk.OrientationHorizontal, 0)

			b.Add(completerImage(md.EmojiURL(e.ID.String(), e.Animated)))
			b.Add(completerLeftLabel(e.Name))
			b.Add(completerRightLabel(guild.Name))
			c.addCompletionEntry(b, e.String())
		}
	}
}
