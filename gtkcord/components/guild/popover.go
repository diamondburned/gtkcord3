package guild

import (
	"html"

	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

func newNamePopover(name string, relative gtk.IWidget) *gtk.Popover {
	popover, _ := gtk.PopoverNew(relative)
	label, _ := gtk.LabelNew("<b>" + html.EscapeString(name) + "</b>")
	label.SetUseMarkup(true)
	label.SetMarginStart(5)
	label.SetMarginEnd(5)
	label.Show()

	popover.Add(label)
	popover.SetPosition(gtk.POS_RIGHT)
	popover.SetModal(false)
	popover.Popup()
	popover.Connect("closed", popover.Destroy)

	return popover
}

type wrapper interface {
	gtk.IWidget
	gtkutils.Marginator
}

func BindName(container gtkutils.Container, w wrapper, name *string) *gtk.EventBox {
	// Wrap the image inside this event box.
	evb, _ := gtk.EventBoxNew()
	evb.Show()
	evb.SetEvents(int(gdk.ENTER_NOTIFY_MASK | gdk.LEAVE_NOTIFY_MASK))

	// shared state
	var popover *gtk.Popover

	// shitty hack to not pass anything down further
	hoverer, ok := w.(Hoverable)

	evb.Connect("enter-notify-event", func() bool {
		if text := *name; text != "" {
			popover = newNamePopover(text, evb)
		}
		if ok {
			hoverer.SetHovered(true)
		}
		return false
	})
	evb.Connect("leave-notify-event", func() bool {
		if popover != nil {
			popover.Popdown()
			popover = nil
		}
		if ok {
			hoverer.SetHovered(false)
		}
		return false
	})

	// Wrap.
	container.Remove(w)
	evb.Add(w)
	container.Add(evb)

	// Transfer margin.
	gtkutils.TransferMargin(evb, w)

	return evb
}

type binder interface {
	gtk.IWidget
	gtkutils.Connector
	SetEvents(int)
}

func BindNameDirect(conn binder, hoverer Hoverable, name *string) {
	// shared state
	var popover *gtk.Popover

	conn.SetEvents(int(gdk.ENTER_NOTIFY_MASK | gdk.LEAVE_NOTIFY_MASK))

	conn.Connect("enter-notify-event", func() bool {
		if text := *name; text != "" {
			popover = newNamePopover(text, conn)
		}
		hoverer.SetHovered(true)
		return false
	})
	conn.Connect("leave-notify-event", func() bool {
		if popover != nil {
			popover.Hide()
			popover = nil
		}
		hoverer.SetHovered(false)
		return false
	})
}
