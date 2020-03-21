// +build gtk

package md

import (
	"testing"

	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

const _md = `Discord sucks.
Seriously.
> be __discord__
> die
> tfw

	asdasdasdasdasdas

yup. ***lolmao*** ` + "`" + `echo "yeet $HOME"` + "`" + `

https://google.com/joe_mama

` + "```" + `gO
package main

func main() {
	fmt.Println("Hello, 世界!")
}
` + "```" + `
meh.

<:Thonk:456835728559702052>

joe mama <a:Thonk:456835728559702052> lol!!

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
	Parse([]byte(_md), tb)

	// md := goldmark.New(
	// 	goldmark.WithParser(parser.NewParser(
	// 		parser.WithBlockParsers(BlockParsers()...),
	// 		parser.WithInlineParsers(InlineParsers()...),
	// 	)),
	// )
	// var buf bytes.Buffer
	// err = md.Convert([]byte(_md), &buf)

	// log.Println(err, buf.String())

	s, _ := gtk.ScrolledWindowNew(nil, nil)
	s.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	s.Add(tv)

	win.Add(s)
	win.SetDefaultSize(800, 600)
	win.ShowAll()

	gtk.Main()
}
