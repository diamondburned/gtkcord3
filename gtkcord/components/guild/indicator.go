package guild

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
)

// (
// 	StripHeight       = 8
// 	StripHoverHeight  = 16
// 	StripActiveHeight = 40
// )

type Hoverable interface {
	SetHovered(hovered bool)
}

var _ Hoverable = (*UnreadStrip)(nil)

type UnreadStrip struct {
	*gtk.Overlay // contains child

	Revealer *gtk.Revealer
	Strip    *gtk.Box // width IconPadding
	Style    *gtk.StyleContext

	// interact states
	suppress  bool
	hover     bool
	active    bool
	intrclass string

	// read states
	unread    bool
	pinged    bool
	readclass string
}

func NewUnreadStrip(child gtk.IWidget) *UnreadStrip {
	overlay, _ := gtk.OverlayNew()
	overlay.SetVAlign(gtk.ALIGN_START)
	overlay.Show()
	overlay.Add(child)

	revealer, _ := gtk.RevealerNew()
	revealer.Show()
	revealer.SetVExpand(true)
	revealer.SetVAlign(gtk.ALIGN_CENTER)
	revealer.SetHAlign(gtk.ALIGN_START)
	revealer.SetRevealChild(false)
	revealer.SetTransitionDuration(70)
	revealer.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)

	strip, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	strip.Show()

	style, _ := strip.GetStyleContext()
	style.AddClass("read-indicator")

	gtkutils.AddCSSUnsafe(style, `
		@define-color pinged rgb(240, 71, 71);

		.read-indicator {
			padding: 6px 3px; /* Always show, use revealer to hide */
			transition: 70ms linear;
			border-radius: 0 99px 99px 0;
			background-color: @theme_fg_color;
		}
		.read-indicator.pinged {
			background-color: @pinged;
		}
		.read-indicator.hover {
			padding: 10px 3px;
		}
		.read-indicator.active {
			padding: 20px 3px;
		}
	`)

	revealer.Add(strip)
	overlay.AddOverlay(revealer)

	return &UnreadStrip{
		Overlay:  overlay,
		Revealer: revealer,
		Strip:    strip,
		Style:    style,
	}
}

func (r *UnreadStrip) State() (unread, pinged bool) {
	return r.unread, r.pinged
}

func (r *UnreadStrip) updateState() {
	// Change the interaction state:
	switch {
	case r.hover:
		gtkutils.DiffClassUnsafe(&r.intrclass, "hover", r.Style)
	case r.active:
		gtkutils.DiffClassUnsafe(&r.intrclass, "active", r.Style)
	default:
		gtkutils.DiffClassUnsafe(&r.intrclass, "", r.Style)
	}

	switch {
	case r.pinged:
		gtkutils.DiffClassUnsafe(&r.readclass, "pinged", r.Style)
	case r.unread:
		// Technically this doesn't do anything, but it removes all other
		// classes. It also gives extra flexibility.
		gtkutils.DiffClassUnsafe(&r.readclass, "unread", r.Style)
	default:
		gtkutils.DiffClassUnsafe(&r.readclass, "", r.Style)
	}

	// Hide the strip if we're not displaying anything. Suppress must be false.
	r.Revealer.SetRevealChild(!r.suppress && (r.intrclass != "" || r.readclass != ""))
}

func (r *UnreadStrip) SetPinged() {
	r.pinged = true
	r.updateState()
}

func (r *UnreadStrip) SetRead() {
	r.pinged = false
	r.unread = false
	r.updateState()
}

func (r *UnreadStrip) SetUnread() {
	r.pinged = false
	r.unread = true
	r.updateState()
}

func (r *UnreadStrip) SetHovered(hovered bool) {
	r.hover = hovered
	r.updateState()
}

func (r *UnreadStrip) SetActive(active bool) {
	r.active = active
	r.updateState()
}

func (r *UnreadStrip) SetSuppress(suppressed bool) {
	r.suppress = suppressed
	r.updateState()
}

func (r *UnreadStrip) FromRow(row *gtk.ListBoxRow) {
	r.active = row.IsSelected()
	r.updateState()
}
