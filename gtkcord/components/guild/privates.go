package guild

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
)

type DMButton struct {
	gtkutils.ExtendedWidget
	Style *gtk.StyleContext

	OnClick func()

	class string
}

// thread-safe
func NewPMButton() (dm *DMButton) {
	icon := icons.GetIcon("system-users-symbolic", IconSize/3*2)

	semaphore.IdleMust(func() {
		r, _ := gtk.ListBoxRowNew()
		r.SetSizeRequest(IconSize+IconPadding*2, IconSize+IconPadding*2)
		r.SetHAlign(gtk.ALIGN_FILL)
		r.SetVAlign(gtk.ALIGN_CENTER)
		r.SetTooltipMarkup("<b>Private Messages</b>")
		r.SetActivatable(true)

		s, _ := r.GetStyleContext()
		s.AddClass("dmbutton")
		s.AddClass("guild")

		i, _ := gtk.ImageNewFromPixbuf(icon)
		i.SetHAlign(gtk.ALIGN_CENTER)
		i.SetVAlign(gtk.ALIGN_CENTER)
		r.Add(i)

		dm = &DMButton{
			ExtendedWidget: r,
			Style:          s,
		}
	})

	return
}

func (dm *DMButton) setUnread(unread bool) {
	var class string
	if unread {
		class = "pinged"
	}
	gtkutils.DiffClassUnsafe(&dm.class, class, dm.Style)
}
