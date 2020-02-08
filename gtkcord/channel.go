package gtkcord

import (
	"fmt"
	"io/ioutil"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	ChannelsWidth = 240
	BannerHeight  = 135
	LabelHeight   = 48
)

type Channels struct {
	gtk.IWidget

	Main *gtk.Box

	// Headers
	Header *gtk.Box
	Name   *gtk.Label
	// nullable
	BannerImage *gtk.Image

	// Channel list
	ChList   *gtk.ListBox
	Channels []*Channel

	GuildID discord.Snowflake
}

type Channel struct {
	ID   discord.Snowflake
	Name string
}

func (g *Guild) loadChannels(s *state.State, guild discord.Guild) error {
	if g.Channels != nil {
		return nil
	}

	/*
		discordChannels, err := s.Channels(guild.ID)
		if err != nil {
			return errors.Wrap(err, "Failed to get channels")
		}
	*/

	main, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return errors.Wrap(err, "Failed to create main box")
	}

	g.Channels = &Channels{
		IWidget: main,
		Main:    main,
	}

	{ // Header

		overlay, err := gtk.OverlayNew()
		if err != nil {
			return errors.Wrap(err, "Failed to make header overlay")
		}

		header, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		if err != nil {
			return errors.Wrap(err, "Failed to create header box")
		}
		header.SetSizeRequest(ChannelsWidth, LabelHeight)
		g.Channels.Header = header

		banner, err := gtk.ImageNew()
		if err != nil {
			return errors.Wrap(err, "Failed to create banner image")
		}
		banner.SetSizeRequest(ChannelsWidth, LabelHeight)
		g.Channels.BannerImage = banner

		if guild.Banner != "" {
			banner.SetSizeRequest(ChannelsWidth, BannerHeight)
			header.SetSizeRequest(ChannelsWidth, BannerHeight)

			go g.UpdateBanner(guild.BannerURL())
		}

		must(overlay.Add, banner)

		// Add label here
		{
			var labelMain gtk.IWidget

			label, err := gtk.LabelNew("")
			if err != nil {
				return errors.Wrap(err, "Failed to create guild name label")
			}
			label.SetXAlign(0.0)
			label.SetMarginStart(20)
			label.SetSizeRequest(ChannelsWidth, LabelHeight)
			g.Channels.Name = label

			if guild.Banner != "" {
				label.SetYAlign(0.1)

				ov, err := gtk.OverlayNew()
				if err != nil {
					return errors.Wrap(err, "Failed to create guild name overlay")
				}

				p, err := icons.PixbufSolid(ChannelsWidth, LabelHeight, 0, 0, 0, 85)
				if err != nil {
					return errors.Wrap(err, "Failed to create pixbuf solid")
				}

				i, err := gtk.ImageNewFromPixbuf(p)
				if err != nil {
					return errors.Wrap(err, "Failed to create solid image")
				}
				i.SetProperty("xalign", 0.0)
				i.SetProperty("yalign", 0.0)

				must(ov.Add, i)
				must(ov.AddOverlay, label)
				labelMain = ov
			} else {
				labelMain = label
			}
			must(label.SetMarkup, bold(guild.Name))
			must(overlay.AddOverlay, labelMain)
		}

		header.Add(overlay)
		main.Add(header)

	}

	return nil
}

func (g *Guild) UpdateBanner(url string) {
	r, err := HTTPClient.Get(url)
	if err != nil {
		logWrap(err, "Failed to GET URL "+url)
		return
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		logError(fmt.Errorf("Bad status code %d for %s", r.StatusCode, url))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logWrap(err, "Failed to download image")
		return
	}

	p, err := NewPixbuf(b, PbSize(ChannelsWidth, BannerHeight))
	if err != nil {
		logWrap(err, "Failed to get the pixbuf guild icon")
		return
	}

	must(g.Channels.BannerImage.SetFromPixbuf, p)
}
