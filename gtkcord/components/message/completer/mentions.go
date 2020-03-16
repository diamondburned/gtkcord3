package completer

import (
	"strings"

	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

func (c *State) completeRecentMentions() {
	ids := c.container.GetRecentAuthors(MaxCompletionEntries)
	guildID := c.container.GetGuildID()

	for _, id := range ids {
		m, err := c.state.Store.Member(guildID, id)
		if err != nil {
			continue
		}

		c.members = append(c.members, *m)
	}

	semaphore.IdleMust(c._completeMembers)
}

func (c *State) completeMentions(word string) {
	if word == "" {
		c.completeRecentMentions()
		return
	}

	guildID := c.container.GetGuildID()
	if !guildID.Valid() {
		c.completeMentionsDM(word)
		return
	}

	members, err := c.state.Store.Members(guildID)
	if err != nil {
		log.Errorln("Failed to get members:", err)
		return
	}

	for i, m := range members {
		if contains(m.User.Username, word) || contains(m.Nick, word) {
			c.members = append(c.members, members[i])

			if len(c.members) > MaxCompletionEntries {
				break
			}
		}
	}

	if len(c.members) == 0 {
		// Request the member in a background goroutine
		c.state.SearchMember(guildID, word)
		return
	}

	semaphore.IdleMust(c._completeMembers)
}

func (c *State) _completeMembers() {
	for _, m := range c.members {
		b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

		var name = m.Nick
		if m.Nick == "" {
			name = m.User.Username
		}

		var url = m.User.AvatarURL()
		if url != "" {
			url += "?size=64"
		}

		b.Add(completerImage(url, cache.Round))
		b.Add(completerLeftLabel(name))
		b.Add(completerRightLabel(m.User.Username + "#" + m.User.Discriminator))
		c.addCompletionEntry(b, m.User.Mention())
	}
}

func (c *State) completeMentionsDM(word string) {
	ch, err := c.state.Channel(c.container.GetChannelID())
	if err != nil {
		log.Errorln("Failed to get DM channel:", err)
		return
	}

	for i, u := range ch.DMRecipients {
		var name = strings.ToLower(u.Username)
		if strings.Contains(name, word) {
			c.users = append(c.users, ch.DMRecipients[i])

			if len(c.users) > MaxCompletionEntries {
				break
			}
		}
	}

	if len(c.users) == 0 {
		return
	}

	semaphore.IdleMust(func() {
		for _, u := range c.users {
			b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

			var url = u.AvatarURL()
			if url != "" {
				url += "?size=64"
			}

			b.Add(completerImage(url))
			b.Add(completerLeftLabel(u.Username))
			b.Add(completerRightLabel(u.Username + "#" + u.Discriminator))
			c.addCompletionEntry(b, u.Mention())
		}
	})
}
