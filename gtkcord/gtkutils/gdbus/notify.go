package gdbus

import (
	"context"
	"errors"
	"sync"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// Notifier wraps around a GIO DBus Connection and allows sending notifications
// using the DBus method.
type Notifier struct {
	*gio.DBusConnection

	actionMu sync.Mutex
	actions  map[uint32][]*Action
}

// NewNotifier creates a new notifier.
func NewNotifier(c *gio.DBusConnection) *Notifier {
	if c == nil {
		return &Notifier{}
	}

	n := &Notifier{
		DBusConnection: c,
		actions:        map[uint32][]*Action{},
	}

	c.SignalSubscribe(
		"", "org.freedesktop.Notifications",
		"", "/org/freedesktop/Notifications", "",
		gio.DBusSignalFlagsNone,
		func(c *gio.DBusConnection, sender, object, iface, signal string, params *glib.Variant) {
			if signal != "ActionInvoked" && signal != "NotificationClosed" {
				return
			}

			switch signal {
			case "ActionInvoked":
				id := params.ChildValue(0).Uint32()
				action := params.ChildValue(1).String()
				n.onAction(id, action)
			case "NotificationClosed":
				id := params.ChildValue(0).Uint32()
				reason := params.ChildValue(1).Uint32()
				n.onClose(id, reason)
			}
		},
	)

	return n
}

func (n *Notifier) onClose(id, reason uint32) {
	n.actionMu.Lock()
	delete(n.actions, id)
	n.actionMu.Unlock()
}

func (n *Notifier) onAction(id uint32, actionKey string) {
	n.actionMu.Lock()
	defer n.actionMu.Unlock()

	actions, ok := n.actions[id]
	if !ok {
		return
	}

	for _, action := range actions {
		if action.ID == actionKey {
			glib.IdleAdd(func() { action.Callback() })
		}
	}
}

type Action struct {
	ID       string
	Label    string
	Callback func() // called in main event loop
}

type Notification struct {
	AppName string
	AppIcon string
	Title   string
	Message string
	Actions []*Action
	Expiry  int32
}

// Notify is blocking.
func (c *Notifier) Notify(n Notification) (uint32, error) {
	if c.DBusConnection == nil {
		return 0, errors.New("no dbus connection")
	}

	args := make([]*glib.Variant, 8)
	args[0] = glib.NewVariantString(n.AppName)
	args[1] = glib.NewVariantUint32(0)
	args[2] = glib.NewVariantString(n.AppIcon)
	args[3] = glib.NewVariantString(n.Title)
	args[4] = glib.NewVariantString(n.Message)

	var firstAction []*glib.Variant

	if len(n.Actions) > 0 {
		firstAction = make([]*glib.Variant, len(n.Actions)*2)

		for i := 0; i < len(firstAction); i += 2 {
			action := n.Actions[i/2]

			firstAction[i+0] = glib.NewVariantString(action.ID)
			firstAction[i+1] = glib.NewVariantString(action.Label)
		}
	}

	args[5] = glib.NewVariantArray(glib.NewVariantType("s"), firstAction)
	args[6] = glib.NewVariantDict(nil).End()
	args[7] = glib.NewVariantInt32(n.Expiry)

	argsTuple := glib.NewVariantTuple(args)

	v, err := c.CallSync(
		context.Background(),
		"org.freedesktop.Notifications",
		"/org/freedesktop/Notifications",
		"org.freedesktop.Notifications",
		"Notify",
		argsTuple,
		glib.NewVariantType("*"), // any
		gio.DBusCallFlagsNone,
		5000,
	)
	if err != nil {
		return 0, err
	}

	child := v.ChildValue(0)
	id := child.Uint32()

	c.actionMu.Lock()
	c.actions[id] = n.Actions
	c.actionMu.Unlock()

	return id, nil
}
