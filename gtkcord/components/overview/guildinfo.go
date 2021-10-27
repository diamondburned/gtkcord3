package overview

import (
	"fmt"
	"html"
	"strconv"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/roundimage"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
)

const GuildIconSize = 96

type GuildInfo struct {
	*gtk.Box

	// rounded
	Image *roundimage.Image

	// top guild name, bottom nitro info
	Info  *gtk.Box
	Name  *gtk.Label
	Extra *gtk.Label // nillable
}

func NewGuildInfo(guild *discord.Guild) *GuildInfo {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.Show()

	img := roundimage.NewImage(0)
	img.SetSizeRequest(GuildIconSize, GuildIconSize)
	img.SetFromIconName("network-server-symbolic", 0)
	img.SetPixelSize(GuildIconSize)
	gtkutils.Margin(img, CommonMargin)
	img.Show()

	info := gtk.NewBox(gtk.OrientationVertical, 0)
	info.Show()

	name := gtk.NewLabel(
		`<span weight="bold" size="xx-large">` + html.EscapeString(guild.Name) + `</span>`)
	name.Show()
	name.SetUseMarkup(true)
	name.SetVExpand(true)
	name.SetVAlign(gtk.AlignEnd)
	name.SetHAlign(gtk.AlignStart)

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

	extra := gtk.NewLabel(`<span color="#ff73fa">` + lvl + `</span>`)
	extra.Show()
	extra.SetUseMarkup(true)
	extra.SetVExpand(true)
	extra.SetVAlign(gtk.AlignStart)
	extra.SetHAlign(gtk.AlignStart)

	box.Add(img)
	box.Add(info)

	info.Add(name)
	info.Add(extra)

	if guild.Icon != "" {
		url := fmt.Sprintf("%s?size=%d", guild.IconURL(), 256)
		cache.SetImageURLScaled(img, url, GuildIconSize, GuildIconSize)
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
