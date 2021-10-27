package overview

import (
	"html"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
)

type ChannelInfo struct {
	*gtk.Box

	// Box for the hash and name
	Header *gtk.Box
	Hash   *gtk.Label
	Name   *gtk.Label

	Description *gtk.TextView
}

func NewChannelInfo(ch *discord.Channel) *ChannelInfo {
	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Show()

	header := gtk.NewBox(gtk.OrientationHorizontal, 0)
	header.Show()
	header.SetMarginTop(CommonMargin)
	header.SetMarginEnd(CommonMargin)
	header.SetMarginStart(CommonMargin)
	header.SetMarginBottom(8)

	hash := gtk.NewLabel(`<span size="xx-large" weight="bold">#</span>`)
	hash.Show()
	hash.SetUseMarkup(true)
	hash.SetMarginEnd(8)
	hash.SetVAlign(gtk.AlignStart)

	name := gtk.NewLabel(
		`<span size="x-large" weight="bold">` + html.EscapeString(ch.Name) + `</span>`)
	name.Show()
	name.SetUseMarkup(true)
	name.SetVAlign(gtk.AlignBaseline)
	name.SetLineWrap(true)
	name.SetLineWrapMode(pango.WrapWordChar)

	desc := gtk.NewTextView()
	desc.Show()
	desc.SetEditable(false)
	desc.SetCursorVisible(false)
	desc.SetHExpand(true)
	desc.SetWrapMode(gtk.WrapWordChar)
	gtkutils.Margin4(desc, 0, CommonMargin, CommonMargin, CommonMargin)

	// Parse the topic into markup/tags:
	md.Parse([]byte(ch.Topic), desc)

	box.Add(header)
	box.Add(desc)

	header.Add(hash)
	header.Add(name)

	return &ChannelInfo{
		box,
		header,
		hash,
		name,
		desc,
	}
}
