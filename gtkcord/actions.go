package gtkcord

import (
	"reflect"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/glib"
)

func (a *Application) bindActions() {
	semaphore.IdleMust(func() {
		a.Application.AddAction(newAction("load-channel", a.actionLoadChannel))
	})
}

func (a *Application) actionLoadChannel(_ *glib.SimpleAction, id int64) {
	ch, err := a.State.Store.Channel(discord.Snowflake(id))
	if err != nil {
		log.Errorln("Can't find channel")
		return
	}

	a.SwitchToID(ch.ID, ch.GuildID)
}

func newAction(name string, fn interface{}) *glib.SimpleAction {
	// Reflection magic:
	v := reflect.ValueOf(fn)
	t := v.Type()
	argT := t.In(1) // grab the second argument

	var varType *glib.VariantType

	switch argT.Kind() {
	case reflect.Bool:
		varType = glib.VARIANT_TYPE_BOOLEAN
	case reflect.Uint8:
		varType = glib.VARIANT_TYPE_BYTE
	case reflect.Int16:
		varType = glib.VARIANT_TYPE_INT16
	case reflect.Int32:
		varType = glib.VARIANT_TYPE_INT32
	case reflect.Int64:
		varType = glib.VARIANT_TYPE_INT64
	case reflect.String:
		varType = glib.VARIANT_TYPE_STRING
	case reflect.Uint16:
		varType = glib.VARIANT_TYPE_UINT16
	case reflect.Uint32:
		varType = glib.VARIANT_TYPE_UINT32
	case reflect.Uint64:
		varType = glib.VARIANT_TYPE_UINT64
	default:
		log.Panicln("Unknown type:", argT)
	}

	simple := glib.SimpleActionNew(name, varType)
	simpleV := reflect.ValueOf(simple)

	simple.Connect("activate", func(_ *glib.SimpleAction, gv *glib.Variant) {
		// Invalid.
		if !gv.IsType(varType) {
			return
		}

		var value interface{}
		switch gv.TypeString() {
		case "s":
			value = gv.GetString()
		case "b":
			value = gv.GetBoolean()
		case "n", "i", "x":
			value, _ = gv.GetInt()
		case "y", "q", "u", "t":
			value, _ = gv.GetUint()
		default:
			return // ???
		}

		go v.Call([]reflect.Value{simpleV, reflect.ValueOf(value)})
	})

	return simple
}
