package channel

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/loadstatus"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/ningen/v2"
	"github.com/diamondburned/ningen/v2/states/read"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/pkg/errors"
)

const (
	ChannelsWidth = 240
	BannerHeight  = 135
	LabelHeight   = 48
)

type Channels struct {
	*loadstatus.Page

	Scroll *gtk.ScrolledWindow
	Main   *gtk.Box

	// Headers
	BannerImage *gtk.Image

	// Channel list
	ChList   *gtk.ListBox
	Channels []*Channel
	Selected *Channel

	state *ningen.State

	OnSelect func(ch *Channel)
	GuildID  discord.GuildID
}

func NewChannels(state *ningen.State, onSelect func(ch *Channel)) (chs *Channels) {
	main := gtk.NewBox(gtk.OrientationVertical, 0)
	main.Show()

	cs := gtk.NewScrolledWindow(nil, nil)
	cs.Show()
	cs.Add(main)

	cl := gtk.NewListBox()
	cl.Show()
	cl.SetVExpand(true)
	cl.SetActivateOnSingleClick(true)
	gtkutils.InjectCSS(cl, "channels", "")

	main.Add(cl)

	page := loadstatus.NewPage()
	page.SetChild(cs)

	chs = &Channels{
		Page:     page,
		Scroll:   cs,
		Main:     main,
		ChList:   cl,
		state:    state,
		OnSelect: onSelect,
	}

	cl.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		if chs.OnSelect == nil || len(chs.Channels) == 0 || r == nil {
			return
		}

		chs.Selected = chs.Channels[r.Index()]
		chs.OnSelect(chs.Selected)
	})

	state.ReadState.OnUpdate(func(rs *read.UpdateEvent) {
		glib.IdleAdd(func() { chs.TraverseReadState(rs) })
	})

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
	chs.GuildID = 0
}

func (chs *Channels) onError(err error) {
	chs.Page.SetError("Error", err)
}

func (chs *Channels) LoadGuild(guildID discord.GuildID) { // async
	chs.SetLoading()
	chs.GuildID = guildID

	go func() {
		onErr := func(err error, wrap string) {
			glib.IdleAdd(func() { chs.onError(errors.Wrap(err, wrap)) })
		}

		channels, err := chs.state.Channels(guildID)
		if err != nil {
			onErr(err, "failed to get guild channels")
			return
		}
		channels = FilterChannels(chs.state, channels)

		var bannerURL string

		guild, err := chs.state.Guild(chs.GuildID)
		if err == nil && guild.Banner != "" {
			bannerURL = guild.BannerURL()
		}

		glib.IdleAdd(func() {
			chs.SetDone()
			chs.Channels = transformChannels(chs.state, channels)

			for _, ch := range chs.Channels {
				chs.ChList.Insert(ch, -1)
			}

			if bannerURL != "" {
				chs.UpdateBanner(guild.BannerURL())
			}
		})
	}()
}

func (chs *Channels) UpdateBanner(url string) {
	if chs.BannerImage == nil {
		chs.BannerImage = gtk.NewImage()
		chs.BannerImage.SetSizeRequest(ChannelsWidth, BannerHeight)
		chs.Main.PackStart(chs.BannerImage, false, false, 0)
	}

	const w, h = ChannelsWidth, BannerHeight
	cache.SetImageURLScaled(chs.BannerImage, url+"?size=512", w, h)
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

func (chs *Channels) TraverseReadState(rs *read.UpdateEvent) {
	for _, ch := range chs.Channels {
		if ch.ID != rs.ChannelID {
			continue
		}

		ch.setUnread(rs.Unread, rs.MentionCount > 0)
		break
	}
}
