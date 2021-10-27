package completer

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/internal/log"
)

func (c *State) completeRecentMentions() {
	guildID := c.container.GuildID()
	if !guildID.IsValid() {
		return
	}

	ids := c.container.RecentAuthors(MaxCompletionEntries)

	for _, id := range ids {
		m, err := c.state.MemberStore.Member(guildID, id)
		if err != nil {
			continue
		}

		c.members = append(c.members, *m)
	}

	c._completeMembers()
}

func (c *State) completeMentions(word string) {
	if word == "" {
		c.completeRecentMentions()
		return
	}

	guildID := c.container.GuildID()
	if !guildID.IsValid() {
		c.completeMentionsDM(word)
		return
	}

	members, err := c.state.MemberStore.Members(guildID)
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
		c.state.MemberState.SearchMember(guildID, word)
		return
	}

	c._completeMembers()
}

func (c *State) _completeMembers() {
	for _, m := range c.members {
		b := gtk.NewBox(gtk.OrientationHorizontal, 0)

		var name = m.Nick
		if m.Nick == "" {
			name = m.User.Username
		}

		var url = m.User.AvatarURL()
		if url != "" {
			url += "?size=32"
		}

		b.Add(completerImage(url))
		b.Add(completerLeftLabel(name))
		b.Add(completerRightLabel(m.User.Username + "#" + m.User.Discriminator))
		c.addCompletionEntry(b, m.User.Mention())
	}
}

func (c *State) completeMentionsDM(word string) {
	ch, err := c.state.Channel(c.container.ChannelID())
	if err != nil {
		log.Errorln("failed to get DM channel:", err)
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

	for _, u := range c.users {
		b := gtk.NewBox(gtk.OrientationHorizontal, 0)

		var url = u.AvatarURL()
		if url != "" {
			url += "?size=64"
		}

		b.Add(completerImage(url))
		b.Add(completerLeftLabel(u.Username))
		b.Add(completerRightLabel(u.Username + "#" + u.Discriminator))
		c.addCompletionEntry(b, u.Mention())
	}
}
