package overview

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/overview/members"
	"github.com/diamondburned/ningen/v2"
)

type Members struct {
	*gtk.Box

	Header  *gtk.Label
	Members *members.Container
}

func NewMembers(s *ningen.State, g discord.GuildID, ch discord.ChannelID) *Members {
	b := gtk.NewBox(gtk.OrientationVertical, 0)
	b.Show()

	h := gtk.NewLabel(`<span weight="bold">Members</span>`)
	h.Show()
	h.SetUseMarkup(true)
	h.SetMarginTop(CommonMargin)
	h.SetMarginBottom(8)

	m := members.New(s)
	m.Load(g, ch)
	m.SetMarginStart(CommonMargin)
	m.SetMarginEnd(CommonMargin)

	b.Add(h)
	b.Add(m)

	return &Members{
		Box:     b,
		Header:  h,
		Members: m,
	}
}
