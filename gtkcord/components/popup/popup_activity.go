package popup

import (
	"fmt"
	"html"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
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
	details, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	gtkutils.InjectCSSUnsafe(details, "popup-activity", "")

	header, _ := gtk.LabelNew("")
	header.SetHAlign(gtk.ALIGN_FILL)
	header.SetXAlign(0.0)
	header.SetSingleLineMode(true)
	header.SetEllipsize(pango.ELLIPSIZE_END)
	gtkutils.Margin4(header, SectionPadding, SectionPadding-3, SectionPadding, SectionPadding)

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.Add(header)
	gtkutils.InjectCSSUnsafe(main, "activity", "")

	return &UserPopupActivity{
		Box:     main,
		Header:  header,
		Details: details,
	}
}

func (a *UserPopupActivity) Update(ac discord.Activity) {
	semaphore.IdleMust(a.UpdateUnsafe, ac)
}

func (a *UserPopupActivity) UpdateUnsafe(ac discord.Activity) {
	switch ac.Type {
	case discord.GameActivity:
		a.Custom = false
		a.image(ac.ApplicationID, ac.Assets)
		a.header("Playing " + ac.Name)

	case discord.ListeningActivity:
		a.Custom = false
		a.image(ac.ApplicationID, ac.Assets)
		a.header("Listening to " + ac.Name)

	case discord.StreamingActivity:
		a.Custom = false
		a.image(ac.ApplicationID, ac.Assets)
		a.header("Streaming " + ac.Details)

	case discord.CustomActivity:
		a.Custom = true
		a.image(0, nil)
		a.header(ningen.EmojiString(ac.Emoji) + " " + ac.State)

		return
	}

	if a.Info == nil {
		l, _ := gtk.LabelNew("?")
		l.SetMarginStart(SectionPadding)
		l.SetMarginEnd(SectionPadding)
		l.SetMarginBottom(SectionPadding / 2)

		l.SetEllipsize(pango.ELLIPSIZE_END)
		l.SetHAlign(gtk.ALIGN_FILL)
		l.SetVAlign(gtk.ALIGN_CENTER)
		l.SetXAlign(0.0)

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

func (a *UserPopupActivity) image(id discord.Snowflake, assets *discord.ActivityAssets) {
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
		a.Image, _ = gtk.ImageNew()
		a.Image.SetSizeRequest(PopupImageSize, PopupImageSize)
		a.Image.SetMarginStart(SectionPadding)
		a.Image.SetMarginBottom(SectionPadding)
		a.Details.PackStart(a.Image, false, false, 0)
	}

	a.Image.SetTooltipText(text)
	go cache.AsyncFetch(asset, a.Image, PopupImageSize, PopupImageSize)
}
