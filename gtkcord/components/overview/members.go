package overview

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/overview/members"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/gotk3/gotk3/gtk"
)

type Members struct {
	*gtk.Box

	Header  *gtk.Label
	Members *members.Container
}

func NewMembers(s *ningen.State, g discord.Guild) *Members {
	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	b.Show()

	h, _ := gtk.LabelNew(`<span weight="bold">Members</span>`)
	h.Show()
	h.SetUseMarkup(true)
	h.SetMarginTop(CommonMargin)
	h.SetMarginBottom(8)

	m := members.New(s)
	m.LoadGuild(g)
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
