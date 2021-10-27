package overview

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/loadstatus"
	"github.com/diamondburned/ningen/v2"
)

const CommonMargin = 15

type Container struct {
	*gtk.ScrolledWindow
	Column *handy.Clamp

	Guild   *GuildInfo
	Channel *ChannelInfo
	Members *Members
}

func overviewErrorPage(err error) gtk.Widgetter {
	page := loadstatus.NewPage()
	page.SetError("Guild Error", err)
	return page
}

func NewContainer(state *ningen.State, gID discord.GuildID, chID discord.ChannelID) gtk.Widgetter {
	state = state.Offline()

	guild, err := state.Guild(gID)
	if err != nil {
		return overviewErrorPage(err)
	}

	ch, err := state.Channel(chID)
	if err != nil {
		return overviewErrorPage(err)
	}

	scroll := gtk.NewScrolledWindow(nil, nil)
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scroll.Show()

	column := handy.NewClamp()
	column.SetMaximumSize(500) // hard codeeee
	column.Show()
	scroll.Add(column)

	b := gtk.NewBox(gtk.OrientationVertical, 0)
	b.Show()
	column.Add(b)

	ginfo := NewGuildInfo(guild)
	cinfo := NewChannelInfo(ch)
	membs := NewMembers(state, gID, chID)

	b.Add(ginfo)
	b.Add(newSeparator())
	b.Add(cinfo)
	b.Add(newSeparator())
	b.Add(membs)

	return &Container{
		scroll,
		column,
		ginfo,
		cinfo,
		membs,
	}
}

func newSeparator() *gtk.Separator {
	s := gtk.NewSeparator(gtk.OrientationHorizontal)
	s.Show()
	// s.SetMarginStart(CommonMargin)
	// s.SetMarginEnd(CommonMargin)
	return s
}
