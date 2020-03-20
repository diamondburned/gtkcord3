// +build gtk

package md

import (
	"testing"

	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

const _md = `Discord sucks.
Seriously.
> be discord
> die
> tfw

   asdasdasdasd

yup.

https://google.com/joe_mama
`

func TestGtk(t *testing.T) {
	gtk.Init(nil)

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatalln("Unable to create window:", err)
	}
	win.SetTitle("Simple Example")
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	tv, _ := gtk.TextViewNew()
	tv.SetHExpand(true)
	tv.SetVExpand(true)
	tv.SetEditable(false)
	tv.SetWrapMode(gtk.WRAP_WORD_CHAR)

	tb, _ := tv.GetBuffer()

	if err := Parse([]byte(_md), tb); err != nil {
		t.Fatal("Failed to parse:", err)
	}

	s, _ := gtk.ScrolledWindowNew(nil, nil)
	s.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	s.Add(tv)

	win.Add(s)
	win.SetDefaultSize(800, 600)
	win.ShowAll()

	gtk.Main()
}
