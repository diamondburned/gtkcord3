package overview

import (
	"html"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type ChannelInfo struct {
	*gtk.Box

	// Box for the hash and name
	Header *gtk.Box
	Hash   *gtk.Label
	Name   *gtk.Label

	Description *gtk.Label
}

func NewChannelInfo(ch discord.Channel) *ChannelInfo {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()

	header, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	header.Show()
	header.SetMarginTop(CommonMargin)
	header.SetMarginEnd(CommonMargin)
	header.SetMarginStart(CommonMargin)
	header.SetMarginBottom(8)

	hash, _ := gtk.LabelNew(`<span size="xx-large" weight="bold">#</span>`)
	hash.Show()
	hash.SetUseMarkup(true)
	hash.SetMarginEnd(8)
	hash.SetVAlign(gtk.ALIGN_START)

	name, _ := gtk.LabelNew(
		`<span size="x-large" weight="bold">` + html.EscapeString(ch.Name) + `</span>`)
	name.Show()
	name.SetUseMarkup(true)
	name.SetVAlign(gtk.ALIGN_BASELINE)
	name.SetLineWrap(true)
	name.SetLineWrapMode(pango.WRAP_WORD_CHAR)

	desc, _ := gtk.LabelNew(string(md.ParseToMarkup([]byte(ch.Topic))))
	desc.Show()
	desc.SetUseMarkup(true)
	desc.SetMarginStart(CommonMargin)
	desc.SetMarginEnd(CommonMargin)
	desc.SetMarginBottom(CommonMargin)
	desc.SetXAlign(0.0)
	desc.SetHExpand(true)
	desc.SetLineWrap(true)
	desc.SetLineWrapMode(pango.WRAP_WORD_CHAR)

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
