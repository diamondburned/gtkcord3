package popup

import (
	"fmt"
	"html"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"

	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
)

const SectionPadding = 10

type UserPopupActivity struct {
	*gtk.Box

	Header *gtk.Label

	Custom  bool
	details bool

	Details *gtk.Box
	Image   *gtk.Image
	Info    *gtk.Label
}

func NewUserPopupActivity() *UserPopupActivity {
	details := gtk.NewBox(gtk.OrientationHorizontal, 0)
	gtkutils.InjectCSS(details, "popup-activity", "")

	header := gtk.NewLabel("")
	header.SetXAlign(0.0)
	header.SetHAlign(gtk.AlignFill)
	header.SetHExpand(true)
	header.SetMaxWidthChars(0)
	header.SetSingleLineMode(true)
	header.SetEllipsize(pango.EllipsizeEnd)
	gtkutils.Margin4(header, SectionPadding, SectionPadding-3, SectionPadding, SectionPadding)

	main := gtk.NewBox(gtk.OrientationVertical, 0)
	main.Add(header)
	gtkutils.InjectCSS(main, "activity", "")

	return &UserPopupActivity{
		Box:     main,
		Header:  header,
		Details: details,
	}
}

func (a *UserPopupActivity) Update(ac discord.Activity) {
	switch ac.Type {
	case discord.GameActivity:
		a.Custom = false
		a.image(ac.AppID, ac.Assets)
		a.header("Playing " + ac.Name)

	case discord.ListeningActivity:
		a.Custom = false
		a.image(ac.AppID, ac.Assets)
		a.header("Listening to " + ac.Name)

	case discord.StreamingActivity:
		a.Custom = false
		a.image(ac.AppID, ac.Assets)
		a.header("Streaming " + ac.Details)

	case discord.CustomActivity:
		a.Custom = true
		a.image(0, nil)

		switch {
		case ac.Emoji == nil:
			a.header(ac.State)
		case ac.Emoji.ID.IsValid():
			a.header(":" + ac.Emoji.Name + ": " + ac.State)
		default:
			a.header(ac.Emoji.Name + " " + ac.State)
		}

		return
	}

	if a.Info == nil {
		l := gtk.NewLabel("?")
		l.SetMarginStart(SectionPadding)
		l.SetMarginEnd(SectionPadding)
		l.SetMarginBottom(SectionPadding / 2)

		l.SetEllipsize(pango.EllipsizeEnd)
		l.SetHAlign(gtk.AlignFill)
		l.SetVAlign(gtk.AlignCenter)
		l.SetXAlign(0.0)
		l.SetHExpand(true)
		l.SetMaxWidthChars(0) // max of parent

		a.Details.Add(l)

		a.Info = l
	}

	a.Info.SetTooltipText(ac.Details + "\n" + ac.State)
	a.Info.SetMarkup(fmt.Sprintf(
		"<span weight=\"bold\">%s</span>\n<span size=\"smaller\">%s</span>",
		html.EscapeString(ac.Details), html.EscapeString(ac.State),
	))
}

func (a *UserPopupActivity) header(name string) {
	if a.Custom {
		a.Header.SetLabel(name)

		if a.details {
			a.Remove(a.Details)
			a.details = false
		}

		return
	}

	if !a.details {
		a.Add(a.Details)
		a.details = true
	}

	a.Header.SetMarkup(`<span size="smaller" weight="bold">` + name + `</span>`)
}

func (a *UserPopupActivity) image(id discord.AppID, assets *discord.ActivityAssets) {
	var asset, text string
	if assets != nil {
		asset = assets.LargeImage
		text = assets.LargeText
	}

	if asset == "" {
		if a.Image != nil {
			a.Remove(a.Image)
			a.Image.Destroy()
			a.Image = nil
		}
		return
	}

	if strings.HasPrefix(asset, "spotify:") {
		asset = "https://i.scdn.co/image/" + strings.TrimPrefix(asset, "spotify:")
	} else {
		asset = "https://cdn.discordapp.com/app-assets/" + id.String() + "/" + asset + ".png"
	}

	if a.Image == nil {
		a.Image = gtk.NewImage()
		a.Image.SetSizeRequest(PopupImageSize, PopupImageSize)
		a.Image.SetMarginStart(SectionPadding)
		a.Image.SetMarginBottom(SectionPadding)
		a.Details.PackStart(a.Image, false, false, 0)
	}

	a.Image.SetTooltipText(text)
	cache.SetImageURLScaled(a.Image, asset, PopupImageSize, PopupImageSize)
}
