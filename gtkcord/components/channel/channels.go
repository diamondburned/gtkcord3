package channel

import (
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Channels struct {
	gtkutils.ExtendedWidget
	GuildID discord.Snowflake

	Scroll *gtk.ScrolledWindow
	Main   *gtk.Box

	// Headers
	BannerImage *gtk.Image

	// Channel list
	ChList   *gtk.ListBox
	Channels []*Channel

	busy sync.Mutex

	state    *ningen.State
	OnSelect func(ch *Channel)
}

func NewChannels(s *ningen.State) (chs *Channels) {
	semaphore.IdleMust(func() {
		main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		main.SetSizeRequest(ChannelsWidth, -1)

		cs, _ := gtk.ScrolledWindowNew(nil, nil)
		cs.SetSizeRequest(ChannelsWidth, -1)
		cs.Add(main)

		cl, _ := gtk.ListBoxNew()
		cl.SetVExpand(true)
		cl.SetActivateOnSingleClick(true)
		gtkutils.InjectCSSUnsafe(cl, "channels", "")

		main.Add(cl)

		chs = &Channels{
			ExtendedWidget: cs,
			Scroll:         cs,
			Main:           main,
			ChList:         cl,
		}

		cl.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
			if chs.OnSelect == nil {
				return
			}

			var row = chs.Channels[r.GetIndex()]
			go func() {
				row.setUnread(false, false)
				chs.OnSelect(row)
			}()
		})
	})

	return nil
}

func (chs *Channels) LoadGuild(guildID discord.Snowflake) error {
	channels, err := chs.state.Channels(guildID)
	if err != nil {
		return errors.Wrap(err, "Failed to get guild channels")
	}
	channels = filterChannels(channels)

	chs.busy.Lock()
	defer chs.busy.Unlock()

	// Remove old channels
	semaphore.IdleMust(func() {
		for _, ch := range chs.Channels {
			chs.ChList.Remove(ch)
		}
	})

	go func() {
		guild, err := chs.state.Guild(guildID)
		if err == nil && guild.Banner != "" {
			go chs.UpdateBanner(guild.BannerURL())
		}
	}()

	chws := transformChannels(chs.state, channels)

	semaphore.IdleMust(func() {
		for _, ch := range chws {
			chs.ChList.Insert(ch, -1)
		}
	})

	return nil
}

func (chs *Channels) UpdateBanner(url string) {
	if chs.BannerImage == nil {
		semaphore.IdleMust(func() {
			chs.BannerImage, _ = gtk.ImageNew()
			chs.BannerImage.SetSizeRequest(ChannelsWidth, BannerHeight)
			chs.Main.PackStart(chs.BannerImage, false, false, 0)
		})
	}

	if err := cache.SetImageScaled(
		url+"?size=512",
		chs.BannerImage,
		ChannelsWidth, BannerHeight); err != nil {

		log.Errorln("Failed to get the pixbuf guild icon:", err)
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

func (chs *Channels) TraverseReadState(rs *gateway.ReadState, ack bool) {
	for _, ch := range chs.Channels {
		if ch.ID != rs.ChannelID {
			continue
		}

		// ack == read
		ch.setUnread(!ack, rs.MentionCount > 0)
		break
	}
}
