package gtkcord

import (
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/pbpool"
	"github.com/diamondburned/gtkcord3/log"
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
	ExtendedWidget

	Scroll *gtk.ScrolledWindow
	Main   *gtk.Box

	// Headers
	BannerImage *gtk.Image

	// Channel list
	ChList   *gtk.ListBox
	Channels []*Channel
}

type Channel struct {
	ExtendedWidget

	Row   *gtk.ListBoxRow
	Label *gtk.Label

	ID       discord.Snowflake
	Guild    discord.Snowflake
	Name     string
	Topic    string
	Category bool

	Messages *Messages
}

func (g *Guild) loadChannels() error {
	if g.Channels != nil {
		return nil
	}

	guild, err := App.State.Guild(g.ID)
	if err != nil {
		return errors.Wrap(err, "Failed to get guild "+g.ID.String())
	}

	chs, err := App.State.Channels(guild.ID)
	if err != nil {
		return errors.Wrap(err, "Failed to get channels")
	}
	chs = filterChannels(App.State, chs)

	cs, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create channel scroller")
	}

	main, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return errors.Wrap(err, "Failed to create main box")
	}

	g.Channels = &Channels{
		ExtendedWidget: cs,
		Scroll:         cs,
		Main:           main,
	}

	/*
	 * === Header ===
	 */

	if guild.Banner != "" {
		banner, err := gtk.ImageNew()
		if err != nil {
			return errors.Wrap(err, "Failed to create banner image")
		}

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

	if err := transformChannels(g.Channels, chs); err != nil {
		return errors.Wrap(err, "Failed to transform channels")
	}

	cl.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		row := g.Channels.Channels[r.GetIndex()]
		App.loadChannel(g, row)
	})

	must(func() {
		cs.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
		main.SetSizeRequest(ChannelsWidth, -1)
		cs.Add(main)

		if banner := g.Channels.BannerImage; banner != nil {
			banner.SetSizeRequest(ChannelsWidth, BannerHeight)
			main.Add(banner)
		}

		cl.SetVExpand(true)
		cl.SetActivateOnSingleClick(true)
		main.Add(cl)

		for _, ch := range g.Channels.Channels {
			cl.Add(ch)
		}
	})

	return nil
}

func newChannel(ch discord.Channel) (*Channel, error) {
	switch ch.Type {
	case discord.GuildText:
		return newChannelRow(ch)
	case discord.GuildCategory:
		return newCategory(ch)
	case discord.DirectMessage, discord.GroupDM:
		return newDMChannel(ch)
	}

	log.Panicln("Unknown channel type " + strconv.Itoa(int(ch.Type)))
	return nil, nil
}

func newCategory(ch discord.Channel) (*Channel, error) {
	r, err := gtk.ListBoxRowNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create channel row")
	}
	r.SetSelectable(false)
	r.SetSensitive(false)

	l, err := gtk.LabelNew(
		`<span font_size="smaller">` + escape(strings.ToUpper(ch.Name)) + "</span>")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create label")
	}
	l.SetUseMarkup(true)
	l.SetXAlign(0)
	l.SetMarginStart(15)
	l.SetMarginTop(15)

	r.Add(l)

	return &Channel{
		ExtendedWidget: r,

		Row:      r,
		Label:    l,
		ID:       ch.ID,
		Guild:    ch.GuildID,
		Name:     ch.Name,
		Topic:    ch.Topic,
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
	l.SetOpacity(0.75) // TODO: read state

	r.Add(l)

	return &Channel{
		ExtendedWidget: r,

		Row:      r,
		Label:    l,
		ID:       ch.ID,
		Guild:    ch.GuildID,
		Name:     ch.Name,
		Topic:    ch.Topic,
		Category: false,
	}, nil
}
func newDMChannel(ch discord.Channel) (*Channel, error) {
	panic("Implement me")
}

func (g *Guild) UpdateBanner(url string) {
	p, err := pbpool.DownloadScaled(url+"?size=512", ChannelsWidth, BannerHeight)
	if err != nil {
		logWrap(err, "Failed to get the pixbuf guild icon")
		return
	}

	must(g.Channels.BannerImage.SetFromPixbuf, p)
}

func (chs *Channels) First() int {
	for i, ch := range chs.Channels {
		if ch.Category {
			continue
		}

		return i
	}
	return -1
}
