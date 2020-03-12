package completer

import (
	"strings"

	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
)

func (c *State) completeChannels(word string) {
	guildID := *c.guildID
	if !guildID.Valid() {
		return
	}

	chs, err := c.state.Channels(guildID)
	if err != nil {
		log.Errorln("Failed to get channels:", err)
		return
	}

	for _, ch := range chs {
		if strings.HasPrefix(ch.Name, word) {
			c.channels = append(c.channels, ch)

			if len(c.channels) > MaxCompletionEntries {
				break
			}
		}
	}

	if len(c.channels) == 0 {
		return
	}

	semaphore.IdleMust(func() {
		for _, ch := range c.channels {
			l := completerLeftLabel("#" + ch.Name)
			c.addCompletionEntry(l, "<#"+ch.ID.String()+">")
		}
	})
}
