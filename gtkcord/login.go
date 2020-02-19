package gtkcord

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

type Login struct {
	*gtk.Box
	Email    *gtk.Entry
	Password *gtk.Entry
	Submit   *gtk.Button
	Error    *gtk.Label
}

type TOTPDialog struct {
	*gtk.Dialog
	Input *gtk.Entry
}

func NewLogin() (*Login, error) {
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	margin4(main, 15, 25, 25, 25)

	err, _ := gtk.LabelNew("")
	err.SetSingleLineMode(true)
	err.SetMarginBottom(10)

	email, _ := gtk.EntryNew()
	email.SetMarginBottom(15)
	email.SetInputPurpose(gtk.INPUT_PURPOSE_EMAIL)
	email.SetPlaceholderText("Email")

	password, _ := gtk.EntryNew()
	password.SetInputPurpose(gtk.INPUT_PURPOSE_PASSWORD)
	password.SetPlaceholderText("Password")
	password.SetVisibility(false)
	password.SetInvisibleChar('*')

	submit, _ := gtk.ButtonNewWithLabel("Login")

	l := &Login{
		Email:    email,
		Password: password,
		Submit:   submit,
		Error:    err,
	}

	submit.Connect("clicked", func() {
		main.SetSensitive(false)
		defer main.SetSensitive(true)

		if err := l.Login(); err != nil {
			l.Error.SetMarkup(fmt.Sprintf(
				`<span color="red">Error: %s</span>`,
				escape(strings.Title(err.Error())),
			))

			log.Errorln("Failed to login:", err)
		}
	})

	main.Add(email)
	main.Add(password)
	main.Add(submit)

	return l, nil
}

func (l *Login) Login() error {
	// 	email := l.Email.
	return nil
}
