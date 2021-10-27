package guild

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
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
	intrclass string
	suppress  bool
	hover     bool
	active    bool

	// read states
	readclass string
	unread    bool
	pinged    bool
}

var stripCSS = gtkutils.CSSAdder(`
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

func NewUnreadStrip(child gtk.Widgetter) *UnreadStrip {
	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetHAlign(gtk.AlignCenter)
	box.StyleContext().AddClass("child-box")
	box.Add(child)

	overlay := gtk.NewOverlay()
	overlay.SetVAlign(gtk.AlignCenter)
	overlay.Add(box)
	overlay.Show()

	revealer := gtk.NewRevealer()
	revealer.SetVExpand(true)
	revealer.SetVAlign(gtk.AlignCenter)
	revealer.SetHAlign(gtk.AlignStart)
	revealer.SetRevealChild(false)
	revealer.SetTransitionDuration(70)
	revealer.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	revealer.Show()

	strip := gtk.NewBox(gtk.OrientationHorizontal, 0)
	strip.Show()

	style := strip.StyleContext()
	style.AddClass("read-indicator")
	stripCSS(style)

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
		gtkutils.DiffClass(&r.intrclass, "hover", r.Style)
	case r.active:
		gtkutils.DiffClass(&r.intrclass, "active", r.Style)
	default:
		gtkutils.DiffClass(&r.intrclass, "", r.Style)
	}

	switch {
	case r.pinged:
		gtkutils.DiffClass(&r.readclass, "pinged", r.Style)
	case r.unread:
		// Technically this doesn't do anything, but it removes all other
		// classes. It also gives extra flexibility.
		gtkutils.DiffClass(&r.readclass, "unread", r.Style)
	default:
		gtkutils.DiffClass(&r.readclass, "", r.Style)
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
