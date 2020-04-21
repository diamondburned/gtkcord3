package overview

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const CommonMargin = 15

type Container struct {
	*gtk.ScrolledWindow
	Column *handy.Column

	Guild   *GuildInfo
	Channel *ChannelInfo
	Members *Members
}

func NewContainer(state *ningen.State, gID, chID discord.Snowflake) (*Container, error) {
	guild, err := state.Store.Guild(gID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get guild")
	}

	ch, err := state.Store.Channel(chID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get channel")
	}

	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.Show()

	column := handy.ColumnNew()
	column.Show()
	column.SetMaximumWidth(500) // hard codeeee
	scroll.Add(column)

	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	b.Show()
	column.Add(b)

	ginfo := NewGuildInfo(*guild)
	cinfo := NewChannelInfo(*ch)
	membs := NewMembers(state, *guild)

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
	}, nil
}

func newSeparator() *gtk.Separator {
	s, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	s.Show()
	// s.SetMarginStart(CommonMargin)
	// s.SetMarginEnd(CommonMargin)
	return s
}
