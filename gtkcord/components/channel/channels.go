package channel

import (
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
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
	*gtk.ScrolledWindow
	Main *gtk.Box

	GuildID discord.Snowflake

	// Headers
	BannerImage *gtk.Image

	// Channel list
	ChList   *gtk.ListBox
	Channels []*Channel
	Selected *Channel

	busy  sync.RWMutex
	state *ningen.State

	OnSelect func(ch *Channel)
}

func NewChannels(state *ningen.State) (chs *Channels) {
	semaphore.IdleMust(func() {
		main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		main.Show()

		cs, _ := gtk.ScrolledWindowNew(nil, nil)
		cs.Show()
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
		}

		cl.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
			if chs.OnSelect == nil || len(chs.Channels) == 0 || r == nil {
				return
			}

			chs.Selected = chs.Channels[r.GetIndex()]
			go func() {
				chs.Selected.setUnread(false, false)
				chs.OnSelect(chs.Selected)
			}()
		})
	})

	state.AddReadChange(chs.TraverseReadState)
	return
}

// // messageCreate handler for unreads
// func (chs *Channels) messageCreate(c *gateway.MessageCreateEvent) {
// 	// If the guild ID doesn't match:
// 	if c.GuildID != chs.GuildID {
// 		return
// 	}
// 	// If the message is the user's:
// 	if c.Author.ID == chs.state.Ready.User.ID {
// 		return
// 	}

// 	chs.busy.Lock()
// 	defer chs.busy.Unlock()

// 	// If the current channel is selected:
// 	if chs.Selected != nil && chs.Selected.ID == c.ChannelID {
// 		if !chs.Selected.unread {

// 		}
// 		return
// 	}

// 	// Find the channel:
// 	ch := chs.FindByID(c.ChannelID)
// 	// If no channel is found:
// 	if ch == nil {
// 		return
// 	}

// }

func (chs *Channels) Cleanup() {
	chs.busy.Lock()
	defer chs.busy.Unlock()

	if chs.Channels == nil {
		return
	}

	// Remove old channels
	semaphore.IdleMust(func() {
		for _, ch := range chs.Channels {
			chs.ChList.Remove(ch)
		}
	})
	chs.Selected = nil
	chs.Channels = nil
}

func (chs *Channels) LoadGuild(guildID discord.Snowflake) error {
	chs.GuildID = guildID

	channels, err := chs.state.Channels(chs.GuildID)
	if err != nil {
		return errors.Wrap(err, "Failed to get guild channels")
	}
	channels = filterChannels(chs.state.State, channels)

	chs.busy.Lock()
	defer chs.busy.Unlock()

	go func() {
		guild, err := chs.state.Guild(chs.GuildID)
		if err == nil && guild.Banner != "" {
			chs.UpdateBanner(guild.BannerURL())
		}
	}()

	chs.Channels = transformChannels(chs.state, channels)

	semaphore.IdleMust(func() {
		for _, ch := range chs.Channels {
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

	const w, h = ChannelsWidth, BannerHeight

	if err := cache.SetImageScaled(url+"?size=512", chs.BannerImage, w, h); err != nil {
		log.Errorln("Failed to get the pixbuf guild icon:", err)
		return
	}
}

func (chs *Channels) FindByID(id discord.Snowflake) *Channel {
	chs.busy.RLock()
	defer chs.busy.RUnlock()

	for _, ch := range chs.Channels {
		if ch.ID == id {
			return ch
		}
	}
	return nil
}

func (chs *Channels) First() *Channel {
	chs.busy.RLock()
	defer chs.busy.RUnlock()

	for _, ch := range chs.Channels {
		if ch.Category {
			continue
		}
		return ch
	}
	return nil
}

func (chs *Channels) TraverseReadState(s *ningen.State, rs *gateway.ReadState, unread bool) {
	chs.busy.RLock()
	defer chs.busy.RUnlock()

	for _, ch := range chs.Channels {
		if ch.ID != rs.ChannelID {
			continue
		}

		ch.setUnread(unread, rs.MentionCount > 0)
		break
	}
}
