package gtkcord

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	ChannelsWidth = 240
	BannerHeight  = 135
	LabelHeight   = 48

	ChannelHash = "# "
)

type Channels struct {
	gtk.IWidget

	Scroll *gtk.ScrolledWindow
	Main   *gtk.Box

	// Headers
	BannerImage *gtk.Image

	// Channel list
	ChList   *gtk.ListBox
	Channels []*Channel

	GuildID discord.Snowflake
}

type Channel struct {
	gtk.IWidget

	Row   *gtk.ListBoxRow
	Label *gtk.Label

	ID       discord.Snowflake
	Name     string
	Category bool
}

func (g *Guild) loadChannels(s *state.State, guild discord.Guild) error {
	if g.Channels != nil {
		return nil
	}

	chs, err := s.Channels(guild.ID)
	if err != nil {
		return errors.Wrap(err, "Failed to get channels")
	}
	chs = filterChannels(s, chs)

	cs, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create channel scroller")
	}
	cs.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)

	main, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return errors.Wrap(err, "Failed to create main box")
	}
	main.SetSizeRequest(ChannelsWidth, -1)
	must(cs.Add, main)

	g.Channels = &Channels{
		IWidget: cs,
		Scroll:  cs,
		Main:    main,
	}

	/*
	 * === Header ===
	 */

	if guild.Banner != "" {
		banner, err := gtk.ImageNew()
		if err != nil {
			return errors.Wrap(err, "Failed to create banner image")
		}
		banner.SetSizeRequest(ChannelsWidth, BannerHeight)

		must(main.Add, banner)
		g.Channels.BannerImage = banner

		go g.UpdateBanner(guild.BannerURL())
	}

	/*
	 * === Channels list ===
	 */

	cl, err := gtk.ListBoxNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create channel list")
	}
	cl.SetActivateOnSingleClick(true)
	must(main.Add, cl)

	if err := transformChannels(g.Channels, chs); err != nil {
		return errors.Wrap(err, "Failed to transform channels")
	}

	for _, ch := range g.Channels.Channels {
		must(cl.Add, ch.IWidget)
	}

	cl.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		row := g.Channels.Channels[r.GetIndex()]
		log.Println("Channel", row.Name, "selected")
	})

	return nil
}

func newChannel(ch discord.Channel) (*Channel, error) {
	switch ch.Type {
	case discord.GuildText, discord.GuildNews, discord.GuildStore:
		return newChannelRow(ch)
	case discord.GuildCategory:
		return newCategory(ch)
	case discord.DirectMessage, discord.GroupDM:
		return newDMChannel(ch)
	case discord.GuildVoice:
		// TODO
		return newChannelRow(ch)
	}

	panic("Unknown channel type " + strconv.Itoa(int(ch.Type)))
}

func newCategory(ch discord.Channel) (*Channel, error) {
	r, err := gtk.ListBoxRowNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create channel row")
	}
	r.SetSelectable(false)
	r.SetSensitive(false)

	l, err := gtk.LabelNew(
		`<span font_variant="smallcaps" font_size="smaller">` + escape(ch.Name) + "</span>")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create label")
	}
	l.SetUseMarkup(true)
	l.SetXAlign(0)
	l.SetMarginStart(15)
	l.SetMarginTop(15)

	must(r.Add, l)
	return &Channel{
		IWidget:  r,
		Row:      r,
		Label:    l,
		ID:       ch.ID,
		Name:     ch.Name,
		Category: true,
	}, nil
}
func newChannelRow(ch discord.Channel) (*Channel, error) {
	r, err := gtk.ListBoxRowNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create channel row")
	}

	l, err := gtk.LabelNew(ChannelHash + bold(ch.Name))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create label")
	}
	l.SetXAlign(0)
	l.SetMarginStart(8)
	l.SetUseMarkup(true)

	must(r.Add, l)
	return &Channel{
		IWidget:  r,
		Row:      r,
		Label:    l,
		ID:       ch.ID,
		Name:     ch.Name,
		Category: false,
	}, nil
}
func newDMChannel(ch discord.Channel) (*Channel, error) {
	panic("Implement me")
}

func (g *Guild) UpdateBanner(url string) {
	r, err := HTTPClient.Get(url + "?size=512")
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
