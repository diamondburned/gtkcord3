package channel

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/members/user"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

const (
	DMAvatarSize = 38
	IconPadding  = 8
)

type PrivateChannel struct {
	*gtk.ListBoxRow
	Style      *gtk.StyleContext
	stateClass string // row style

	Body *user.Container

	ID   discord.Snowflake
	Name string

	Group bool
}

// NOT thread-safe
func newPrivateChannel(ch discord.Channel) (pc *PrivateChannel) {
	var name = ch.Name
	if name == "" {
		var names = make([]string, len(ch.DMRecipients))
		for i, p := range ch.DMRecipients {
			names[i] = p.Username
		}

		name = humanize.Strings(names)
	}

	body := user.New()
	body.Show()
	body.Name.SetText(name)

	r, _ := gtk.ListBoxRowNew()
	r.Add(body)
	r.Show()
	// set the channel ID to searches
	r.SetProperty("name", ch.ID.String())

	rs, _ := r.GetStyleContext()
	rs.AddClass("dmchannel")

	return &PrivateChannel{
		ListBoxRow: r,
		Style:      rs,
		Body:       body,

		ID:   ch.ID,
		Name: name,
		// Group: ch.Type == discord.GroupDM,
	}
}

func _ChIDFromRow(row *gtk.ListBoxRow) string {
	v, err := row.GetProperty("name")
	if err != nil {
		log.Errorln("Failed to get channel ID:", err)
		return ""
	}
	return v.(string)
}

func (pc *PrivateChannel) GuildID() discord.Snowflake { return 0 }

func (pc *PrivateChannel) ChannelID() discord.Snowflake {
	return pc.ID
}

func (pc *PrivateChannel) ChannelInfo() (name, topic string) {
	return pc.Name, ""
}

func (pc *PrivateChannel) setClass(class string) {
	gtkutils.DiffClassUnsafe(&pc.stateClass, class, pc.Style)
}

func (pc *PrivateChannel) updateActivity(ac *discord.Activity) {
	pc.Body.UpdateActivity(ac)
}

func (pc *PrivateChannel) updateStatus(status discord.Status) {
	pc.Body.UpdateStatus(status)
}

func (pc *PrivateChannel) updateAvatar(url string) {
	pc.Body.UpdateAvatar(url)
}

func (pc *PrivateChannel) setUnread(unread bool) {
	if unread {
		pc.setClass("pinged")
	} else {
		pc.setClass("")
	}
}
