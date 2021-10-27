package extras

import (
	"fmt"
	"html"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/roundimage"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/ningen/v2"
)

func newNormalEmbed(
	s *ningen.State, msg *discord.Message, embed discord.Embed) gtk.Widgetter {

	content := gtk.NewBox(gtk.OrientationVertical, 0)
	content.SetHAlign(gtk.AlignFill)

	widthHint := 0 // used for calculating requested embed width

	if embed.Author != nil {
		box := gtk.NewBox(gtk.OrientationHorizontal, 0)
		embedSetMargin(box)

		if embed.Author.ProxyIcon != "" {
			img := roundimage.NewImage(0)
			img.SetMarginEnd(variables.EmbedMargin)
			cache.SetImageStreamed(img, embed.Author.ProxyIcon, 24, 24)
			box.Add(img)
		}

		if embed.Author.Name != "" {
			author := gtk.NewLabel(embed.Author.Name)
			author.SetUseMarkup(true)
			author.SetLineWrap(true)
			author.SetLineWrapMode(pango.WrapWordChar)
			author.SetXAlign(0.0)

			if embed.Author.URL != "" {
				author.SetMarkup(fmt.Sprintf(
					`<a href="%s">%s</a>`,
					html.EscapeString(embed.Author.URL), html.EscapeString(embed.Author.Name),
				))
			}

			box.Add(author)
		}

		content.Add(box)
	}

	if embed.Title != "" {
		var title = `<span weight="heavy">` + html.EscapeString(embed.Title) + `</span>`
		if embed.URL != "" {
			title = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(embed.URL), title)
		}

		label := gtk.NewLabel("")
		label.SetMarkup(title)
		label.SetLineWrap(true)
		label.SetLineWrapMode(pango.WrapWordChar)
		label.SetXAlign(0.0)
		embedSetMargin(label)

		content.Add(label)
	}

	if embed.Description != "" {
		txtv := gtk.NewTextView()
		txtv.SetCursorVisible(false)
		txtv.SetEditable(false)
		txtv.SetWrapMode(gtk.WrapWordChar)

		embedSetMargin(txtv)
		md.ParseWithMessage([]byte(embed.Description), txtv, s, msg)

		// Make text smaller:
		md.WrapTag(txtv, map[string]interface{}{
			"scale":     0.9,
			"scale-set": true,
		})

		content.Add(txtv)
	}

	if len(embed.Fields) > 0 {
		fields := gtk.NewGrid()
		fields.SetRowSpacing(uint(7))
		fields.SetColumnSpacing(uint(14))
		embedSetMargin(fields)

		content.Add(fields)

		col, row := 0, 0

		for _, field := range embed.Fields {
			text := gtk.NewLabel("")
			text.SetEllipsize(pango.EllipsizeEnd)
			text.SetXAlign(0.0)
			text.SetMarkup(fmt.Sprintf(
				`<span weight="heavy">%s</span>`+"\n"+`<span weight="light">%s</span>`,
				field.Name, field.Value,
			))
			text.SetTooltipText(field.Name + "\n" + field.Value)

			// I have no idea what this does. It')s just improvised.
			if field.Inline && col < 3 {
				fields.Attach(text, col, row, 1, 1)
				col++
			} else {
				if col > 0 {
					row++
				}

				col = 0
				fields.Attach(text, col, row, 1, 1)

				if !field.Inline {
					row++
				} else {
					col++
				}
			}
		}
	}

	if embed.Footer != nil || embed.Timestamp.IsValid() {
		footer := gtk.NewBox(gtk.OrientationHorizontal, 0)
		embedSetMargin(footer)

		if embed.Footer != nil {
			if embed.Footer.ProxyIcon != "" {
				img := roundimage.NewImage(0)
				img.SetMarginEnd(variables.EmbedMargin)
				cache.SetImageStreamed(img, embed.Footer.ProxyIcon, 24, 24)
				footer.Add(img)
			}

			if embed.Footer.Text != "" {
				text := gtk.NewLabel(embed.Footer.Text)
				text.SetVAlign(gtk.AlignStart)
				text.SetOpacity(0.65)
				text.SetLineWrap(true)
				text.SetLineWrapMode(pango.WrapWordChar)
				text.SetXAlign(0.0)

				footer.Add(text)
			}
		}

		if embed.Timestamp.IsValid() {
			time := humanize.TimeAgo(embed.Timestamp.Time())

			text := gtk.NewLabel(time)
			text.SetAttributes(gtkutils.PangoAttrs(
				pango.NewAttrScale(0.85),
				pango.NewAttrForegroundAlpha(0xDDDD),
			))
			if embed.Footer != nil {
				text.SetText(" - " + time)
			}

			footer.Add(text)
		}

		content.Add(footer)
	}

	if embed.Thumbnail != nil {
		wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
		wrapper.Add(content)

		// Do a shitty hack:
		content = wrapper
		content.SetHAlign(gtk.AlignStart)

		w, h := int(embed.Thumbnail.Width), int(embed.Thumbnail.Height)
		w, h = maxSize(w, h, 80, 80)

		wrapper.Add(newExtraImage(
			sizeToURL(embed.Thumbnail.Proxy, w, h),
			embed.Thumbnail.URL, 0, 0,
		))
	}

	if embed.Image != nil {
		wrapper := gtk.NewBox(gtk.OrientationVertical, 0)
		wrapper.Add(content)

		// Do a shitty hack again:
		content = wrapper
		content.SetHAlign(gtk.AlignStart)

		w, h := int(embed.Image.Width), int(embed.Image.Height)
		w, h = maxSize(w, h, variables.EmbedMaxWidth, variables.EmbedImgHeight)

		// set width hint to resize embeds accordingly
		widthHint = w

		wrapper.Add(newExtraImage(
			sizeToURL(embed.Image.Proxy, w, h),
			embed.Image.URL, w, h,
		))
	}

	// Calculate the embed width without padding:
	var w = clampWidth(variables.EmbedMaxWidth)
	if widthHint > 0 && w > widthHint {
		w = widthHint
	}

	// Wrap the content inside another box:
	main := gtk.NewBox(gtk.OrientationVertical, 0)
	main.SetHAlign(gtk.AlignStart)
	main.SetSizeRequest(w+(variables.EmbedMargin*2), 0)
	main.Add(content)

	// Apply margin to content:
	content.SetMarginTop(variables.EmbedMargin) // account for children not having margin top
	content.SetMarginBottom(variables.EmbedMargin / 2)

	// Add a frame around the main embed:
	gtkutils.InjectCSS(main, "embed", fmt.Sprintf(EmbedMainCSS, embed.Color))

	return main
}
