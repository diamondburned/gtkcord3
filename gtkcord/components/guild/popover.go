package guild

import (
	"html"

	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
)

type namePopover struct {
	*gtk.Popover
	label *gtk.Label
}

func newNamePopover(name string, relative gtk.Widgetter) *namePopover {
	label := gtk.NewLabel("<b>" + html.EscapeString(name) + "</b>")
	label.SetUseMarkup(true)
	label.SetMarginStart(5)
	label.SetMarginEnd(5)
	label.SetHExpand(true)
	label.SetLineWrapMode(pango.WrapWordChar)
	label.SetMaxWidthChars(50)
	label.Show()

	popover := gtk.NewPopover(relative)
	popover.SetPosition(gtk.PosRight)
	popover.Add(label)
	popover.SetModal(false)

	return &namePopover{popover, label}
}

func BindName(c gtk.Containerer, w gtk.Widgetter, name *string) *gtk.EventBox {
	// Wrap the image inside this event box.
	evb := gtk.NewEventBox()
	evb.AddEvents(int(gdk.EnterNotifyMask | gdk.LeaveNotifyMask))
	evb.Show()

	// shared state
	var popover *namePopover

	// shitty hack to not pass anything down further
	hoverer, ok := w.(Hoverable)

	evb.Connect("enter-notify-event", func() bool {
		if text := *name; text != "" {
			if popover == nil {
				popover = newNamePopover(text, evb)
			}
			popover.label.SetText(text)
			popover.Popup()
		}
		if ok {
			hoverer.SetHovered(true)
		}
		return false
	})
	evb.Connect("leave-notify-event", func() bool {
		if popover != nil {
			popover.Popdown()
		}
		if ok {
			hoverer.SetHovered(false)
		}
		return false
	})

	// Wrap.
	container := gtk.BaseContainer(c)
	container.Remove(w)
	evb.Add(w)
	container.Add(evb)

	// Transfer margin.
	gtkutils.TransferMargin(evb, w)

	return evb
}

func BindNameDirect(w gtk.Widgetter, hoverer Hoverable, name *string) {
	// shared state
	var popover *namePopover

	conn := gtk.BaseWidget(w)
	conn.SetEvents(int(gdk.EnterNotifyMask | gdk.LeaveNotifyMask))

	conn.Connect("enter-notify-event", func() bool {
		if text := *name; text != "" {
			if popover == nil {
				popover = newNamePopover(text, conn)
			}
			popover.label.SetText(text)
			popover.Popup()
		}
		hoverer.SetHovered(true)
		return false
	})
	conn.Connect("leave-notify-event", func() bool {
		if popover != nil {
			popover.Popdown()
		}
		hoverer.SetHovered(false)
		return false
	})
}
