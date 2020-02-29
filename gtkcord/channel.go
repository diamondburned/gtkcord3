package gtkcord

import (
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
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
	gtkutils.ExtendedWidget
	Guild *Guild

	Scroll *gtk.ScrolledWindow
	Main   *gtk.Box

	// Headers
	BannerImage *gtk.Image

	// Channel list
	ChList   *gtk.ListBox
	Channels []*Channel
}

type Channel struct {
	gtkutils.ExtendedWidget
	Channels *Channels

	Row   *gtk.ListBoxRow
	Label *gtk.Label

	ID       discord.Snowflake
	Guild    discord.Snowflake
	Name     string
	Topic    string
	Category bool
	LastMsg  discord.Snowflake

	Messages *Messages

	unread bool

	// we keep track of opacity changes, since we don't want thousands of
	// queued up functions only to change the opacity.
	opacity float64
}

func (g *Guild) prefetchChannels() error {
	chs, err := App.State.Channels(g.ID)
	if err != nil {
		return errors.Wrap(err, "Failed to get channels")
	}
	chs = filterChannels(chs)

	g.Channels = &Channels{
		Guild: g,
	}

	if err := transformChannels(g.Channels, chs); err != nil {
		return errors.Wrap(err, "Failed to transform channels")
	}

	return nil
}

func (g *Guild) loadChannels() error {
	if g.Channels != nil && g.Channels.Main != nil {
		return nil
	}

	if g.Channels == nil {
		if err := g.prefetchChannels(); err != nil {
			return errors.Wrap(err, "Failed to load prefetched channel")
		}
	}

	/*
	 * === Main box ===
	 */

	cs := must(gtk.ScrolledWindowNew,
		nilAdjustment(), nilAdjustment()).(*gtk.ScrolledWindow)
	must(cs.SetPolicy, gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	g.Channels.ExtendedWidget = cs
	g.Channels.Scroll = cs

	main := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	must(main.SetSizeRequest, ChannelsWidth, -1)
	gtkutils.InjectCSS(main, "channels", "")
	g.Channels.Main = main

	must(cs.Add, main)

	/*
	 * === Channels list ===
	 */

	cl := must(gtk.ListBoxNew).(*gtk.ListBox)
	must(cl.SetVExpand, true)
	must(cl.SetActivateOnSingleClick, true)
	must(cl.Connect, "row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		row := g.Channels.Channels[r.GetIndex()]
		go func() {
			row.setUnread(false)
			App.loadChannel(g, row)
		}()
	})

	gtkutils.InjectCSS(cl, "channels", "")
	must(main.Add, cl)

	for _, ch := range g.Channels.Channels {
		ch.Channels = g.Channels
		must(cl.Add, ch)

		if ch.Category || !ch.LastMsg.Valid() {
			continue
		}

		if rs := App.State.FindLastRead(ch.ID); rs != nil {
			ch.updateReadState(rs)
		}
	}

	/*
	 * === Header ===
	 */

	if g.BannerURL != "" {
		go g.Channels.UpdateBanner(g.BannerURL)
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

func newCategory(ch discord.Channel) (chw *Channel) {
	name := `<span font_size="smaller">` + escape(strings.ToUpper(ch.Name)) + "</span>"

	must(func() {
		l, _ := gtk.LabelNew(name)
		l.SetUseMarkup(true)
		l.SetXAlign(0.0)
		l.SetMarginStart(15)
		l.SetMarginTop(15)

		r, _ := gtk.ListBoxRowNew()
		r.SetSelectable(false)
		r.SetSensitive(false)
		r.Add(l)

		chw = &Channel{
			ExtendedWidget: r,

			Row:      r,
			Label:    l,
			ID:       ch.ID,
			Guild:    ch.GuildID,
			Name:     ch.Name,
			Topic:    ch.Topic,
			Category: true,
		}
	})

	if App.State.ChannelMuted(chw.ID) {
		chw.setOpacity(0.25)
	}

	return chw
}

func newChannelRow(ch discord.Channel) (chw *Channel) {
	must(func() {
		l, _ := gtk.LabelNew(ChannelHash + bold(escape(ch.Name)))
		l.SetUseMarkup(true)
		l.SetXAlign(0.0)
		l.SetMarginStart(8)

		r, _ := gtk.ListBoxRowNew()
		r.Add(l)

		chw = &Channel{
			ExtendedWidget: r,

			Row:      r,
			Label:    l,
			ID:       ch.ID,
			Guild:    ch.GuildID,
			Name:     ch.Name,
			Topic:    ch.Topic,
			Category: false,
			LastMsg:  ch.LastMessageID,
			unread:   true, // workaround to set opacity
		}
	})

	chw.setUnread(false)

	return chw
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

	if err := cache.SetImage(
		url+"?size=512",
		chs.BannerImage,
		cache.Resize(ChannelsWidth, BannerHeight)); err != nil {

		logWrap(err, "Failed to get the pixbuf guild icon")
		return
	}
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

func (ch *Channel) setOpacity(opacity float64) {
	if opacity == ch.opacity {
		return
	}

	ch.opacity = opacity
	must(ch.SetOpacity, opacity)
}
