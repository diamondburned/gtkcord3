package channel

import (
	"strconv"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/user"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/humanize"
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

	Name string
	ID   discord.ChannelID

	Group bool
}

// NOT thread-safe
func newPrivateChannel(ch discord.Channel) (pc *PrivateChannel) {
	name := ch.Name
	if name == "" {
		names := make([]string, len(ch.DMRecipients))
		for i := range ch.DMRecipients {
			names[i] = ch.DMRecipients[i].Username
		}
		name = humanize.Strings(names)
	}

	if name == "" {
		name = "Unnamed"
	}

	body := user.New()
	body.Show()
	body.Name.SetText(name)

	r := gtk.NewListBoxRow()
	r.SetName(ch.ID.String())
	r.Add(body)
	r.Show()

	rs := r.StyleContext()
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

func chIDFromRow(row *gtk.ListBoxRow) discord.ChannelID {
	v, _ := strconv.ParseInt(row.Name(), 10, 64)
	return discord.ChannelID(v)
}

func (pc *PrivateChannel) GuildID() discord.GuildID { return 0 }

func (pc *PrivateChannel) ChannelID() discord.ChannelID {
	return pc.ID
}

func (pc *PrivateChannel) ChannelInfo() (name, topic string) {
	return pc.Name, ""
}

func (pc *PrivateChannel) setClass(class string) {
	gtkutils.DiffClass(&pc.stateClass, class, pc.Style)
}

func (pc *PrivateChannel) updateActivity(ac *discord.Activity) {
	pc.Body.UpdateActivity(ac)
}

func (pc *PrivateChannel) updateStatus(status gateway.Status) {
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
