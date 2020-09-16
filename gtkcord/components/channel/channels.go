package channel

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/ningen/states/read"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	ChannelsWidth = 240
	BannerHeight  = 135
	LabelHeight   = 48
)

type Channels struct {
	*gtk.ScrolledWindow
	Main *gtk.Box

	GuildID discord.GuildID

	// Headers
	BannerImage *gtk.Image

	// Channel list
	ChList   *gtk.ListBox
	Channels []*Channel
	Selected *Channel

	state *ningen.State

	OnSelect func(ch *Channel)
}

func NewChannels(state *ningen.State, onSelect func(ch *Channel)) (chs *Channels) {
	semaphore.IdleMust(func() {
		main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		main.Show()

		cs, _ := gtk.ScrolledWindowNew(nil, nil)
		cs.Show()
		cs.SetSizeRequest(variables.ChannelWidth, -1)
		cs.Add(main)

		cl, _ := gtk.ListBoxNew()
		cl.Show()
		cl.SetVExpand(true)
		cl.SetActivateOnSingleClick(true)
		gtkutils.InjectCSSUnsafe(cl, "channels", "")

		main.Add(cl)

		chs = &Channels{
			ScrolledWindow: cs,
			Main:           main,
			ChList:         cl,
			state:          state,
			OnSelect:       onSelect,
		}

		cl.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
			if chs.OnSelect == nil || len(chs.Channels) == 0 || r == nil {
				return
			}

			chs.Selected = chs.Channels[r.GetIndex()]
			chs.OnSelect(chs.Selected)
		})
	})

	state.ReadState.OnUpdate(chs.TraverseReadState)
	return
}

func (chs *Channels) Cleanup() {
	if chs.Channels == nil {
		return
	}

	// Remove old channels
	for _, ch := range chs.Channels {
		chs.ChList.Remove(ch)
	}

	chs.Selected = nil
	chs.Channels = nil
}

func (chs *Channels) LoadGuild(guildID discord.GuildID) error {
	chs.GuildID = guildID

	channels, err := chs.state.Channels(chs.GuildID)
	if err != nil {
		return errors.Wrap(err, "Failed to get guild channels")
	}
	channels = filterChannels(chs.state, channels)

	guild, err := chs.state.Store.Guild(chs.GuildID)
	if err == nil && guild.Banner != "" {
		chs.UpdateBanner(guild.BannerURL())
	}

	chs.Channels = transformChannels(chs.state, channels)

	for i, ch := range chs.Channels {
		chs.ChList.Insert(ch, i)
	}

	return nil
}

func (chs *Channels) UpdateBanner(url string) {
	if chs.BannerImage == nil {
		chs.BannerImage, _ = gtk.ImageNew()
		chs.BannerImage.SetSizeRequest(ChannelsWidth, BannerHeight)
		chs.Main.PackStart(chs.BannerImage, false, false, 0)
	}

	const w, h = ChannelsWidth, BannerHeight

	go func() {
		if err := cache.SetImageScaled(url+"?size=512", chs.BannerImage, w, h); err != nil {
			log.Errorln("Failed to get the pixbuf guild icon:", err)
			return
		}
	}()
}

func (chs *Channels) FindByID(id discord.ChannelID) *Channel {
	for _, ch := range chs.Channels {
		if ch.ID == id {
			return ch
		}
	}
	return nil
}

func (chs *Channels) First() *Channel {
	for _, ch := range chs.Channels {
		if ch.Category {
			continue
		}
		return ch
	}
	return nil
}

func (chs *Channels) TraverseReadState(e *read.UpdateEvent) { 
	rs, unread := e.ReadState, e.Unread
	semaphore.Async(func() {
		for _, ch := range chs.Channels {
			if ch.ID != rs.ChannelID {
				continue
			}

			ch.setUnread(unread, rs.MentionCount > 0)
			break
		}
	})
}
