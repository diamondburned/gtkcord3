package guild

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

type DMButton struct {
	*gtk.ListBoxRow
	Unread *UnreadStrip

	OnClick func()

	class string
}

// thread-safe
func NewPMButton(s *ningen.State) (dm *DMButton) {
	r, _ := gtk.ListBoxRowNew()
	r.Show()
	r.SetHAlign(gtk.ALIGN_FILL)
	r.SetVAlign(gtk.ALIGN_CENTER)
	r.SetActivatable(true)

	r.SetSizeRequest(IconSize+IconPadding*2, IconSize+IconPadding*2)
	gtkutils.Margin2(r, IconPadding/2, 0)
	gtkutils.InjectCSSUnsafe(r, "dmbutton", "")

	i, _ := gtk.ImageNew()
	i.Show()
	i.SetSizeRequest(IconSize, IconSize)
	i.SetHAlign(gtk.ALIGN_CENTER)
	i.SetVAlign(gtk.ALIGN_CENTER)
	gtkutils.ImageSetIcon(i, "system-users-symbolic", IconSize/3*2)

	ov := NewUnreadStrip(i)
	r.Add(ov)

	dm = &DMButton{
		ListBoxRow: r,
		Unread:     ov,
	}

	name := "Private Messages"
	BindName(r, ov, &name)

	// Initialize the read state.
	go dm.resetRead(s)

	return
}

func (dm *DMButton) onClick() {
	dm.Unread.SetActive(true)
	dm.OnClick()
}

func (dm *DMButton) setUnread(unread bool) {
	if unread {
		dm.Unread.SetPinged()
	} else {
		dm.Unread.SetRead()
	}
}

func (dm *DMButton) inactive() {
	dm.Unread.SetActive(false)
}

func (dm *DMButton) resetRead(s *ningen.State) {
	// Find and detect any unread channels:
	chs, err := s.PrivateChannels()
	if err != nil {
		log.Errorln("Failed to get private channels for DMButton:", err)
		return
	}

	for _, ch := range chs {
		rs := s.FindLastRead(ch.ID)
		if rs == nil {
			continue
		}

		// Snowflakes have timestamps, which allow us to do this:
		if ch.LastMessageID.Time().After(rs.LastMessageID.Time()) {
			dm.setUnread(true)
			break
		}
	}
}
