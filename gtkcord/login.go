package gtkcord

import "github.com/gotk3/gotk3/gtk"

type Login struct {
	*gtk.Box
	Email    *gtk.Entry
	Password *gtk.Entry
}

type 2FADialog struct {
	*gtk.Dialog
	Input *gtk.Entry
}

func NewLogin() (*Login, error) {
	email, _ := gtk.EntryNew()
	email.SetInputPurpose(gtk.INPUT_PURPOSE_EMAIL)
	email.SetPlaceholderText("Email")

	password, _ := gtk.EntryNew()
	password.SetInputPurpose(gtk.INPUT_PURPOSE_PASSWORD)
	password.SetPlaceholderText("Password")
	password.SetVisibility(false)
	password.SetInvisibleChar('*')
}
