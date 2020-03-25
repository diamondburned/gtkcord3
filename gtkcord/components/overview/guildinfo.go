package overview

import (
	"fmt"
	"html"
	"strconv"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
)

const GuildIconSize = 96

type GuildInfo struct {
	*gtk.Box

	// rounded
	Image *gtk.Image

	// top guild name, bottom nitro info
	Info  *gtk.Box
	Name  *gtk.Label
	Extra *gtk.Label // nillable
}

func NewGuildInfo(guild discord.Guild) *GuildInfo {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.Show()

	img, _ := gtk.ImageNew()
	img.Show()
	img.SetSizeRequest(GuildIconSize, GuildIconSize)
	gtkutils.Margin(img, CommonMargin)
	gtkutils.ImageSetIcon(img, "network-server-symbolic", GuildIconSize)

	info, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	info.Show()

	name, _ := gtk.LabelNew(
		`<span weight="bold" size="xx-large">` + html.EscapeString(guild.Name) + `</span>`)
	name.Show()
	name.SetUseMarkup(true)
	name.SetVExpand(true)
	name.SetVAlign(gtk.ALIGN_END)
	name.SetHAlign(gtk.ALIGN_START)

	var lvl string

	switch guild.NitroBoost {
	case discord.NitroLevel1:
		lvl = "Nitro Level 1"
	case discord.NitroLevel2:
		lvl = "Nitro Level 2"
	case discord.NitroLevel3:
		lvl = "Nitro Level 3"
	default:
		lvl = "-"
	}

	extra, _ := gtk.LabelNew(`<span color="#ff73fa">` + lvl + `</span>`)
	extra.Show()
	extra.SetUseMarkup(true)
	extra.SetVExpand(true)
	extra.SetVAlign(gtk.ALIGN_START)
	extra.SetHAlign(gtk.ALIGN_START)

	box.Add(img)
	box.Add(info)

	info.Add(name)
	info.Add(extra)

	if guild.Icon != "" {
		cache.AsyncFetchUnsafe(
			fmt.Sprintf("%s?size=%d", guild.IconURL(), 256),
			img, GuildIconSize, GuildIconSize, cache.Round,
		)
	}

	return &GuildInfo{
		box,
		img,
		info,
		name,
		extra,
	}
}

func sizeToURL(url string, w, h int) string {
	return url + "?width=" + strconv.Itoa(w) + "&height=" + strconv.Itoa(h)
}
