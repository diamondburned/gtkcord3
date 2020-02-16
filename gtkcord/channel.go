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

	/*
	 * === Main box ===
	 */

	cs := must(gtk.ScrolledWindowNew,
		(*gtk.Adjustment)(nil), (*gtk.Adjustment)(nil)).(*gtk.ScrolledWindow)
	must(cs.SetPolicy, gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)

	main := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	must(main.SetSizeRequest, ChannelsWidth, -1)

	must(cs.Add, main)

	g.Channels = &Channels{
		ExtendedWidget: cs,
		Scroll:         cs,
		Main:           main,
	}

	/*
	 * === Header ===
	 */

	if guild.Banner != "" {
		go g.Channels.UpdateBanner(guild.BannerURL())
	}

	/*
	 * === Channels list ===
	 */

	cl := must(gtk.ListBoxNew).(*gtk.ListBox)
	must(cl.SetVExpand, true)
	must(cl.SetActivateOnSingleClick, true)
	must(cl.Connect, "row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		row := g.Channels.Channels[r.GetIndex()]
		go App.loadChannel(g, row)
	})

	must(main.Add, cl)

	if err := transformChannels(g.Channels, chs); err != nil {
		return errors.Wrap(err, "Failed to transform channels")
	}

	for _, ch := range g.Channels.Channels {
		must(cl.Add, ch)
	}

	return nil
}

func newChannel(ch discord.Channel) *Channel {
	switch ch.Type {
	case discord.GuildText:
		return newChannelRow(ch)
	case discord.GuildCategory:
		return newCategory(ch)
	case discord.DirectMessage, discord.GroupDM:
		return newDMChannel(ch)
	}

	log.Panicln("Unknown channel type " + strconv.Itoa(int(ch.Type)))
	return nil
}

func newCategory(ch discord.Channel) *Channel {
	name := `<span font_size="smaller">` + escape(strings.ToUpper(ch.Name)) + "</span>"

	r := must(gtk.ListBoxRowNew).(*gtk.ListBoxRow)
	must(r.SetSelectable, false)
	must(r.SetSensitive, false)

	l := must(gtk.LabelNew, name).(*gtk.Label)
	must(l.SetUseMarkup, true)
	must(l.SetXAlign, 0.0)
	must(l.SetMarginStart, 15)
	must(l.SetMarginTop, 15)

	must(r.Add, l)

	return &Channel{
		ExtendedWidget: r,

		Row:      r,
		Label:    l,
		ID:       ch.ID,
		Guild:    ch.GuildID,
		Name:     ch.Name,
		Topic:    ch.Topic,
		Category: true,
	}
}

func newChannelRow(ch discord.Channel) *Channel {
	r := must(gtk.ListBoxRowNew).(*gtk.ListBoxRow)

	l := must(gtk.LabelNew, ChannelHash+bold(escape(ch.Name))).(*gtk.Label)
	must(l.SetUseMarkup, true)
	must(l.SetXAlign, 0.0)
	must(l.SetMarginStart, 8)
	must(l.SetOpacity, 0.75) // TODO: read state

	must(r.Add, l)

	return &Channel{
		ExtendedWidget: r,

		Row:      r,
		Label:    l,
		ID:       ch.ID,
		Guild:    ch.GuildID,
		Name:     ch.Name,
		Topic:    ch.Topic,
		Category: false,
	}
}
func newDMChannel(ch discord.Channel) *Channel {
	panic("Implement me")
}

func (chs *Channels) UpdateBanner(url string) {
	if chs.BannerImage == nil {
		chs.BannerImage = must(gtk.ImageNew).(*gtk.Image)
		must(chs.BannerImage.SetSizeRequest, ChannelsWidth, BannerHeight)
		must(chs.Main.PackStart, chs.BannerImage, false, false, uint(0))
	}

	p, err := pbpool.DownloadScaled(url+"?size=512", ChannelsWidth, BannerHeight)
	if err != nil {
		logWrap(err, "Failed to get the pixbuf guild icon")
		return
	}

	must(chs.BannerImage.SetFromPixbuf, p)
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
