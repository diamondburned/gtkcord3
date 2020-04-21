package guild

import (
	"html"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sasha-s/go-deadlock"
)

const (
	FolderSize  = 32
	IconSize    = 52
	IconPadding = 8
)

type Guild struct {
	*gtk.ListBoxRow
	Parent *GuildFolder
	Unread *UnreadStrip

	Event *gtk.EventBox
	Image *gtk.Image
	IURL  string

	BannerURL string

	ID   discord.Snowflake
	Name string

	busy      deadlock.Mutex
	muted     bool
	unreadChs map[discord.Snowflake]bool
}

func marginate(r *gtk.ListBoxRow, i *gtk.Image) {
	// Set paddings (height is less, width is WIDE):
	r.SetSizeRequest(IconSize+IconPadding*3, IconSize+IconPadding)

	if i != nil {
		i.SetSizeRequest(IconSize, IconSize)
	}
}

func newGuildRow(
	s *ningen.State,
	guildID discord.Snowflake,
	g *discord.Guild,
	parent *GuildFolder) (*Guild, error) {

	var fetcherr error

	if g == nil {
		g, fetcherr = s.Guild(guildID)
		if fetcherr != nil {
			log.Errorln("Failed to get guild ID " + guildID.String() + ", using a placeholder...")
			g = &discord.Guild{
				ID:   guildID,
				Name: "Unavailable",
			}
		}
	}

	var guild *Guild

	r, _ := gtk.ListBoxRowNew()
	r.SetHAlign(gtk.ALIGN_CENTER)
	r.SetVAlign(gtk.ALIGN_CENTER)
	r.SetActivatable(true)
	gtkutils.InjectCSSUnsafe(r, "guild", "")

	i, _ := gtk.ImageNew()
	gtkutils.ImageSetIcon(i, "system-users-symbolic", IconSize/3*2)
	i.SetHAlign(gtk.ALIGN_CENTER)
	i.SetVAlign(gtk.ALIGN_CENTER)

	marginate(r, i)

	// gtkutils.Margin2(i, IconPadding, IconPadding) // extra padding

	guild = &Guild{
		ListBoxRow: r,
		Parent:     parent,
		Unread:     NewUnreadStrip(i),

		ID:        guildID,
		Name:      g.Name,
		IURL:      g.IconURL(),
		Image:     i,
		BannerURL: g.BannerURL(),

		unreadChs: map[discord.Snowflake]bool{},
	}

	// Bind the name popup.
	guild.Event = BindName(guild.ListBoxRow, guild.Unread, &guild.Name)

	// Check if the guild is unavailable:
	if fetcherr != nil {
		guild.SetUnavailable(true)
		return guild, nil
	}

	// Prefetch unread state:
	go func() {
		// Update the guild icon in the background.
		guild.UpdateImage()

		if s.GuildMuted(guildID, false) {
			guild.muted = true
			return
		}

		if rs := guild.containsUnreadChannel(s); rs != nil {
			unread := true
			pinged := rs.MentionCount > 0

			semaphore.Async(func() {
				guild.busy.Lock()
				guild.setUnread(unread, pinged)
				guild.busy.Unlock()
			})
		}
	}()

	return guild, nil
}

func (g *Guild) SetUnavailable(unavailable bool) {
	g.ListBoxRow.SetSensitive(!unavailable)
}

func (g *Guild) UpdateImage() {
	if g.IURL == "" {
		return
	}

	err := cache.SetImageScaled(g.IURL+"?size=64", g.Image, IconSize, IconSize, cache.Round)
	if err != nil {
		log.Errorln("Failed to update the pixbuf guild icon:", err)
		return
	}
}

// nil == none
func (guild *Guild) containsUnreadChannel(s *ningen.State) *gateway.ReadState {
	channels, err := s.Channels(guild.ID)
	if err != nil {
		log.Errorln("Failed to get channels:", err)
		return nil
	}

	guild.busy.Lock()
	defer guild.busy.Unlock()

	guild.unreadChs = map[discord.Snowflake]bool{}
	var found *gateway.ReadState

	for _, ch := range channels {
		// in a guild, only text channels matter:
		if ch.Type != discord.GuildText {
			continue
		}

		if s.CategoryMuted(ch.ID) {
			continue
		}

		if rs := s.FindLastRead(ch.ID); rs != nil {
			if ch.LastMessageID == rs.LastMessageID {
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
		// TODO: migrate to UnreadStrip and get rid of the goroutine.
		go guild.Parent.setUnread(unread, pinged)
	}
}

// func (guild *Guild) setUnread(s *ningen.State, unread, pinged bool) {
// 	if s.GuildMuted(guild.ID, false) {
// 		return
// 	}

// 	if !unread && guild.Folder == nil {
// 		if rs := guild.containsUnreadChannel(s); rs != nil {
// 			unread = true
// 			pinged = rs.MentionCount > 0
// 		}
// 	}

// 	switch {
// 	case pinged:
// 		guild.setClass("pinged")
// 	case unread:
// 		guild.setClass("unread")
// 	default:
// 		guild.setClass("")
// 	}

// 	if guild.Parent != nil {
// 		for _, guild := range guild.Parent.Folder.Guilds {
// 			unread := guild.stateClass == "unread"
// 			pinged := guild.stateClass == "pinged"

// 			if unread || pinged {
// 				guild.Parent.setUnread(s, true, pinged)
// 				return
// 			}
// 		}

// 		guild.Parent.setUnread(s, false, false)
// 	}
// }

func escape(str string) string {
	return html.EscapeString(str)
}

func bold(str string) string {
	return "<b>" + escape(str) + "</b>"
}
