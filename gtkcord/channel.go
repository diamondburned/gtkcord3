package gtkcord

import (
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/httpcache"
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
	Category bool

	Messages *Messages
}

func (ch *Channel) loadMessages(s *state.State, parser *md.Parser) error {
	if ch.Messages == nil {
		ch.Messages = &Messages{
			ChannelID: ch.ID,
		}
	}

	if err := ch.Messages.Reset(s, parser); err != nil {
		return errors.Wrap(err, "Failed to reset messages in channel")
	}

	return nil
}

func (g *Guild) loadChannels(
	s *state.State,
	guild discord.Guild,
	onChannel func(*Guild, *Channel)) error {

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
	cl.SetVExpand(true)
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
		onChannel(g, row)
	})

	/*
	 * === Messages ===
	 */

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
		`<span font_size="smaller">` + escape(strings.ToUpper(ch.Name)) + "</span>")
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
		Category: false,
	}, nil
}
func newDMChannel(ch discord.Channel) (*Channel, error) {
	panic("Implement me")
}

func (g *Guild) UpdateBanner(url string) {
	b, err := httpcache.HTTPGet(url + "?size=512")
	if err != nil {
		logWrap(err, "Failed to GET URL "+url)
		return
	}

	p, err := NewPixbuf(b, PbSize(ChannelsWidth, BannerHeight))
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
