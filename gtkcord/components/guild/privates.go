package guild

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
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
	semaphore.IdleMust(func() {
		r, _ := gtk.ListBoxRowNew()
		r.Show()
		r.SetSizeRequest(IconSize+IconPadding*2, IconSize+IconPadding*2)
		r.SetHAlign(gtk.ALIGN_FILL)
		r.SetVAlign(gtk.ALIGN_CENTER)
		r.SetTooltipMarkup("<b>Private Messages</b>")
		r.SetActivatable(true)

		s, _ := r.GetStyleContext()
		s.AddClass("dmbutton")

		i, _ := gtk.ImageNew()
		i.Show()
		i.SetHAlign(gtk.ALIGN_CENTER)
		i.SetVAlign(gtk.ALIGN_CENTER)
		gtkutils.ImageSetIcon(i, "system-users-symbolic", IconSize/3*2)
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
	gtkutils.DiffClass(&dm.class, class, dm.Style)
}
