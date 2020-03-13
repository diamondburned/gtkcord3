package guild

import (
	"html"
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

const (
	FolderSize  = 42
	IconSize    = 52
	IconPadding = 8
)

type Guild struct {
	gtkutils.ExtendedWidget
	Parent *GuildFolder

	Row   *gtk.ListBoxRow
	Style *gtk.StyleContext

	Image *gtk.Image
	IURL  string

	BannerURL string

	ID   discord.Snowflake
	Name string

	busy       sync.Mutex
	stateClass string
	unreadChs  map[discord.Snowflake]bool
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

	var name = bold(g.Name)
	var guild *Guild

	semaphore.IdleMust(func() {
		r, _ := gtk.ListBoxRowNew()
		// Set paddings:
		r.SetSizeRequest(IconSize+IconPadding*2, IconSize+IconPadding*2)
		r.SetHAlign(gtk.ALIGN_FILL)
		r.SetVAlign(gtk.ALIGN_CENTER)
		r.SetTooltipMarkup(name)
		r.SetActivatable(true)

		style, _ := r.GetStyleContext()
		style.AddClass("guild")

		i, _ := gtk.ImageNew()
		gtkutils.ImageSetIcon(i, "system-users-symbolic", IconSize/3*2)
		i.SetHAlign(gtk.ALIGN_CENTER)
		i.SetVAlign(gtk.ALIGN_CENTER)
		r.Add(i)

		guild = &Guild{
			ExtendedWidget: r,
			Parent:         parent,

			Row:       r,
			Style:     style,
			ID:        guildID,
			Name:      g.Name,
			IURL:      g.IconURL(),
			Image:     i,
			BannerURL: g.BannerURL(),

			unreadChs: map[discord.Snowflake]bool{},
		}

		// Check if the guild is unavailable:
		if fetcherr != nil {
			guild.SetUnavailable(true)
		}
	})

	if fetcherr != nil {
		return guild, nil
	}

	// Prefetch unread state:
	go func() {
		if s.GuildMuted(guildID, false) {
			guild.setClass("muted")
			return
		}

		if rs := guild.containsUnreadChannel(s); rs != nil {
			unread := true
			pinged := rs.MentionCount > 0

			guild.setUnread(unread, pinged)
		}
	}()

	return guild, nil
}

// thread safe
func (g *Guild) setClass(class string) {
	// gtkutils.DiffClass(&g.stateClass, class, g.Style)
}

func (g *Guild) SetUnavailable(unavailable bool) {
	g.Row.SetSensitive(!unavailable)
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
	guild.busy.Lock()
	defer guild.busy.Unlock()

	if guild.stateClass == "muted" {
		return
	}

	switch {
	case pinged:
		guild.setClass("pinged")
	case unread:
		guild.setClass("unread")
	default:
		guild.setClass("")
	}

	if guild.Parent != nil {
		guild.Parent.setUnread(unread, pinged)
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
