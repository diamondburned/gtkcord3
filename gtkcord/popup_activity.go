package gtkcord

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const SectionPadding = 7

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

	header, _ := gtk.LabelNew("")
	header.SetHAlign(gtk.ALIGN_START)
	header.SetSingleLineMode(true)
	header.SetEllipsize(pango.ELLIPSIZE_END)
	header.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	margin4(header, SectionPadding, SectionPadding-3, SectionPadding, SectionPadding)

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.Add(header)
	gtkutils.InjectCSSUnsafe(main, "activity", `
		.activity {
			background-color: rgba(0, 0, 0, 0.12)
		}
	`)

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
		must(func() {
			a.image(ac.ApplicationID, ac.Assets.LargeImage, ac.Assets.LargeText)
			a.header("Playing " + ac.Name)
		})

	case discord.ListeningActivity:
		a.Custom = false
		must(func() {
			a.image(ac.ApplicationID, ac.Assets.LargeImage, ac.Assets.LargeText)
			a.header("Listening to " + ac.Name)
		})

	case discord.StreamingActivity:
		a.Custom = false
		must(func() {
			a.image(ac.ApplicationID, ac.Assets.LargeImage, ac.Assets.LargeText)
			a.header("Streaming " + ac.Details)
		})

	case discord.CustomActivity:
		a.Custom = true
		must(func() {
			a.image(0, "", "")
			a.header(ac.State)
		})

		return
	}

	if a.Info == nil {
		l := must(gtk.LabelNew, "?").(*gtk.Label)
		must(func() {
			l.SetMarginStart(SectionPadding)
			l.SetMarginEnd(SectionPadding)
			l.SetEllipsize(pango.ELLIPSIZE_END)
			l.SetLineWrapMode(pango.WRAP_WORD_CHAR)
			l.SetHAlign(gtk.ALIGN_START)

			a.Details.Add(l)
		})

		a.Info = l
	}

	must(a.Info.SetTooltipText, ac.Details+"\n"+ac.State)
	must(a.Info.SetMarkup, fmt.Sprintf(
		"<span weight=\"bold\">%s</span>\n<span size=\"smaller\">%s</span>",
		escape(ac.Details), escape(ac.State),
	))
}

func (a *UserPopupActivity) header(name string) {
	if a.Custom {
		log.Println("label:", name)
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

func (a *UserPopupActivity) image(id discord.Snowflake, asset, text string) {
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
		a.Image.SetSizeRequest(HeaderStatusSize, HeaderStatusSize)
		a.Image.SetMarginStart(SectionPadding)
		a.Image.SetMarginBottom(SectionPadding)
		a.Details.PackStart(a.Image, false, false, 0)
	}

	a.Image.SetTooltipText(text)
	go asyncFetch(asset, a.Image, HeaderStatusSize, HeaderStatusSize)
}
