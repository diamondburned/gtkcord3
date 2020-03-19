package guild

import (
	"html"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

type Avatar struct {
	*gtk.ListBoxRow

	Image  *gtk.Image
	IStyle *gtk.StyleContext
	iclass string

	state *ningen.State
}

func NewAvatar(s *ningen.State) *Avatar {
	r, _ := gtk.ListBoxRowNew()
	r.Show()
	r.SetSelectable(false) // still activatable
	r.SetSizeRequest(IconSize+IconPadding*2, IconSize+IconPadding*2)
	gtkutils.InjectCSSUnsafe(r, "avatar", "")

	i, _ := gtk.ImageNew()
	i.Show()
	i.SetHAlign(gtk.ALIGN_CENTER)
	i.SetVAlign(gtk.ALIGN_CENTER)
	gtkutils.ImageSetIcon(i, "user-available-symbolic", IconSize)
	r.Add(i)

	is, _ := i.GetStyleContext()
	is.AddClass("status")

	a := &Avatar{ListBoxRow: r, Image: i, IStyle: is, state: s}

	s.AddHandler(func(v interface{}) {
		switch v.(type) {
		case *gateway.SessionsReplaceEvent, *gateway.UserUpdateEvent:
			a.CheckUpdate()
		}
	})

	return a
}

func (a *Avatar) OnClick() {
	go a.CheckPresence()
	popup.SpawnHamburger(a.ListBoxRow, a.state)
}

func (a *Avatar) UpdateStatus(status discord.Status) {
	switch status {
	case discord.OnlineStatus:
		gtkutils.DiffClassUnsafe(&a.iclass, "online", a.IStyle)
	case discord.DoNotDisturbStatus:
		gtkutils.DiffClassUnsafe(&a.iclass, "busy", a.IStyle)
	case discord.IdleStatus:
		gtkutils.DiffClassUnsafe(&a.iclass, "idle", a.IStyle)
	case discord.InvisibleStatus, discord.OfflineStatus:
		gtkutils.DiffClassUnsafe(&a.iclass, "offline", a.IStyle)
	case discord.UnknownStatus:
		gtkutils.DiffClassUnsafe(&a.iclass, "unknown", a.IStyle)
	}
}

// thread-safe
func (a *Avatar) CheckUpdate() {
	a.CheckPresence()
	a.CheckUser()
}

// thread-safe
func (a *Avatar) CheckPresence() {
	if p, _ := a.state.Presence(0, a.state.Ready.User.ID); p != nil {
		semaphore.IdleMust(a.UpdateStatus, p.Status)
	}
}

// thread-safe
func (a *Avatar) CheckUser() {
	if u, _ := a.state.Me(); u != nil {
		semaphore.IdleMust(a.UpdateUser, *u)
	}
}

func (a *Avatar) UpdateUser(u discord.User) {
	a.SetTooltipMarkup(`<span weight="bold">` + html.EscapeString(u.Username) + `</span>` +
		"#" + u.Discriminator)

	if u.Avatar == "" {
		return
	}

	var url = u.AvatarURL() + "?size=64"

	go func() {
		err := cache.SetImageScaled(url, a.Image, IconSize, IconSize, cache.Round)
		if err != nil {
			log.Errorln("Failed to update the pixbuf hamburger icon:", err)
			return
		}
	}()
}
