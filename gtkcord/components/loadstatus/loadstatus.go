package loadstatus

import (
	"strings"

	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

type Page struct {
	gtk.Stack
	spin    *spinner
	err     *errorPage
	main    gtk.Widgetter
	pholder gtk.Widgetter
}

func NewPage() *Page {
	page := &Page{Stack: *gtk.NewStack()}
	page.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	page.SetLoading()
	page.ShowAll()
	return page
}

func (p *Page) SetError(title string, err error) {
	p.spin.spinner.Stop()

	if p.err == nil {
		p.err = newErrorPage()
		p.Stack.AddNamed(p.err, "err")
	}

	p.SetVisibleChild(p.err)
	p.err.SetInfo(title, err.Error())
}

func (p *Page) SetPlaceholder(placeholder gtk.Widgetter) {
	if p.pholder == placeholder {
		p.SetVisibleChild(placeholder)
		return
	}

	if p.pholder != nil {
		p.Remove(p.pholder)
	}

	p.pholder = placeholder
	p.AddNamed(placeholder, "pholder")
	p.spin.spinner.Stop()
}

func (p *Page) SetLoading() {
	if p.spin == nil {
		p.spin = newSpinner()
		p.AddNamed(p.spin, "spin")
	}

	p.SetVisibleChild(p.spin)
	p.spin.spinner.Start()
}

func (p *Page) SetDone() {
	p.SetChild(p.main)
}

func (p *Page) SetChild(main gtk.Widgetter) {
	if p.main == main {
		p.SetVisibleChild(main)
		return
	}

	if p.main != nil {
		p.Remove(p.main)
	}

	p.main = main
	p.AddNamed(p.main, "main")
	p.spin.spinner.Stop()
}

type errorPage struct {
	handy.StatusPage
}

func newErrorPage() *errorPage {
	page := &errorPage{StatusPage: *handy.NewStatusPage()}
	page.SetIconName("action-unavailable-symbolic")
	return page
}

func (err *errorPage) SetInfo(title, error string) {
	err.SetTitle(title)

	var desc strings.Builder

	parts := strings.Split(error, ": ")
	for i, part := range parts {
		desc.WriteString(strings.Repeat("\t", i))
		desc.WriteString(part)
		if i != len(parts)-1 {
			desc.WriteString(": ")
			desc.WriteString("\n")
		}
	}

	err.SetDescription(desc.String())
}

type spinner struct {
	gtk.Box
	spinner *gtk.Spinner
}

func newSpinner() *spinner {
	spin := gtk.NewSpinner()
	spin.SetVAlign(gtk.AlignCenter)
	spin.SetHAlign(gtk.AlignCenter)
	spin.SetSizeRequest(48, 48)
	spin.Start()

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.SetVAlign(gtk.AlignCenter)
	box.SetHAlign(gtk.AlignCenter)
	box.SetVExpand(true)
	box.SetHExpand(true)
	box.Add(spin)
	box.ShowAll()

	return &spinner{
		Box:     *box,
		spinner: spin,
	}
}
