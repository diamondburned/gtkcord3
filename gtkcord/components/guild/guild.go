package guild

import (
	"html"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/roundimage"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
)

const (
	FolderSize  = 32
	IconSize    = 48
	IconPadding = 8
	TotalWidth  = IconSize + IconPadding*3
)

type Guild struct {
	*gtk.ListBoxRow
	Parent *GuildFolder
	Unread *UnreadStrip

	Event *gtk.EventBox
	Image *roundimage.Image

	IconURL   string
	BannerURL string

	ID   discord.GuildID
	Name string

	unreadChs map[discord.ChannelID]bool
	muted     bool
}

func marginate(r *gtk.ListBoxRow, i *roundimage.Image) {
	// Set paddings (height is less, width is WIDE):
	r.SetSizeRequest(TotalWidth, IconSize+IconPadding)

	if i != nil {
		i.SetSizeRequest(IconSize, IconSize)
	}
}

func newGuildRow(s *ningen.State, guildID discord.GuildID, parent *GuildFolder) *Guild {
	g, err := s.Offline().Guild(guildID)
	if err != nil {
		log.Errorln("failed to get guild ID " + guildID.String() + ", using a placeholder...")
		g = &discord.Guild{
			ID:   guildID,
			Name: "Unavailable",
		}
	}

	var guild *Guild

	r := gtk.NewListBoxRow()
	i := roundimage.NewImage(0)
	marginate(r, i)

	r.SetHAlign(gtk.AlignCenter)
	r.SetVAlign(gtk.AlignCenter)
	r.SetSensitive(err == nil)
	r.SetActivatable(true)
	gtkutils.InjectCSS(r, "guild", "")

	i.SetInitials(g.Name)
	i.SetHAlign(gtk.AlignCenter)
	i.SetVAlign(gtk.AlignCenter)

	// gtkutils.Margin2(i, IconPadding, IconPadding) // extra padding

	guild = &Guild{
		ListBoxRow: r,
		Parent:     parent,
		Unread:     NewUnreadStrip(i),
		ID:         guildID,
		Name:       g.Name,
		Image:      i,
		IconURL:    g.IconURL(),
		BannerURL:  g.BannerURL(),
		unreadChs:  map[discord.ChannelID]bool{},
	}

	// Bind the name popup.
	guild.Event = BindName(guild.ListBoxRow, guild.Unread, &guild.Name)

	// Check if the guild is unavailable:
	// TODO: retry mechanism
	if err != nil {
		guild.SetUnavailable(true)
		return guild
	}

	// Update the guild icon in the background.
	guild.UpdateImage()

	if s.MutedState.Guild(guildID, false) {
		guild.muted = true
		return guild
	}

	if rs := guild.containsUnreadChannel(s); rs != nil {
		log.Printf("for guild %s found unread %#v", g.Name, rs)
		pinged := rs.MentionCount > 0
		guild.setUnread(true, pinged)
	}

	return guild
}

func (g *Guild) SetUnavailable(unavailable bool) {
	g.ListBoxRow.SetSensitive(!unavailable)
}

func (g *Guild) UpdateImage() {
	if g.IconURL == "" {
		g.Image.Clear()
		return
	}

	cache.SetImageURLScaled(g.Image, g.IconURL+"?size=64", IconSize, IconSize)
}

// nil == none
func (guild *Guild) containsUnreadChannel(s *ningen.State) *gateway.ReadState {
	channels, err := s.Offline().Channels(guild.ID)
	if err != nil {
		log.Errorln("failed to get channels:", err)
		return nil
	}

	// A bit slow is ok.
	channels = channel.FilterChannels(s, channels)

	guild.unreadChs = map[discord.ChannelID]bool{}
	var found *gateway.ReadState

	for _, ch := range channels {
		// in a guild, only text channels matter:
		if ch.Type != discord.GuildText {
			continue
		}

		if s.MutedState.Channel(ch.ID) || s.MutedState.Category(ch.ID) {
			continue
		}

		if rs := s.ReadState.FindLast(ch.ID); rs != nil {
			unread := true &&
				rs.LastMessageID.IsValid() &&
				ch.LastMessageID.IsValid() &&
				ch.LastMessageID > rs.LastMessageID

			if !unread {
				continue
			}

			pinged := rs.MentionCount > 0
			guild.unreadChs[ch.ID] = pinged

			if found == nil || pinged {
				found = rs
			}
		}
	}

	return found
}

func (guild *Guild) setUnread(unread, pinged bool) {
	switch {
	case pinged:
		guild.Unread.SetPinged()
	case unread:
		guild.Unread.SetUnread()
	default:
		guild.Unread.SetRead()
	}

	if guild.Parent != nil {
		guild.Parent.setUnread(unread, pinged)
	}
}

func escape(str string) string {
	return html.EscapeString(str)
}

func bold(str string) string {
	return "<b>" + escape(str) + "</b>"
}
