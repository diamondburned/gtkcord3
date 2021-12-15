package login

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/diamondburned/arikawa/v2/state"
	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

var ErrDLNotFound = errors.New("DiscordLogin not found. Please install it from the GitHub page.")

type Login struct {
	*gtk.Box
	Token  *gtk.Entry
	Submit *gtk.Button
	Error  *gtk.Label

	// Button that opens discordlogin
	DLogin *gtk.Button

	LastError error
	LastToken string

	displayed bool
	finish    func(s *ningen.State)
}

func NewHeader() *handy.HeaderBar {
	h := handy.NewHeaderBar()
	h.SetShowCloseButton(true)
	h.SetTitle("Log into gtkcord3")

	return h
}

func NewLogin(finish func(s *ningen.State)) *Login {
	main := gtk.NewBox(gtk.OrientationVertical, 0)
	main.SetMarginTop(15)
	main.SetMarginBottom(50)
	main.SetMarginStart(35)
	main.SetMarginEnd(35)
	main.SetSizeRequest(250, -1)
	main.SetVAlign(gtk.AlignCenter)
	main.SetHAlign(gtk.AlignCenter)
	gtkutils.InjectCSS(main, "login", "")

	err := gtk.NewLabel("")
	err.SetSingleLineMode(false)
	err.SetLineWrap(true)
	err.SetLineWrapMode(pango.WrapWordChar)
	err.SetMarginBottom(10)
	err.SetHAlign(gtk.AlignStart)
	err.SetMarginStart(2)
	err.SetMarginEnd(2)

	token := gtk.NewEntry()
	token.SetMarginBottom(15)
	token.SetInputPurpose(gtk.InputPurposePassword)
	token.SetPlaceholderText("Token")
	token.SetVisibility(false)
	token.SetInvisibleChar('●')

	submit := gtk.NewButtonWithLabel("Login")
	submit.SetMarginBottom(15)
	gtkutils.InjectCSS(submit, "login", `
		button.login {
			background-color: #7289da;
			color: #FFFFFF;
		}
	`)

	retry := gtk.NewButtonWithLabel("Retry")
	gtkutils.InjectCSS(retry, "retry", "")

	dlogin := gtk.NewButtonWithLabel("Use DiscordLogin")
	gtkutils.InjectCSS(dlogin, "discordlogin", "")

	l := &Login{
		Box:    main,
		Token:  token,
		Submit: submit,
		Error:  err,
		DLogin: dlogin,

		finish: finish,
	}

	token.Connect("activate", l.Login)
	submit.Connect("clicked", l.Login)
	dlogin.Connect("clicked", l.DiscordLogin)

	subbtn := gtk.NewBox(gtk.OrientationHorizontal, 15)
	subbtn.SetHomogeneous(true)
	subbtn.Add(retry)
	subbtn.Add(dlogin)

	main.Add(err)
	main.Add(token)
	main.Add(submit)
	main.Add(subbtn)

	return l
}

func (l *Login) Run() {
	// Display the error if there's any:
	if l.LastError != nil {
		l.error(l.LastError)
	}

	if !l.displayed {
		p := window.SwitchToPage("login")
		p.SetHeader(NewHeader())
		p.SetChild(l)
		window.ShowAll()

		l.displayed = true
	}

	if l.LastToken != "" {
		l.Retry()
	}
}

func (l *Login) error(err error) {
	l.LastError = err

	l.Error.SetMarkup(fmt.Sprintf(
		`<span color="red">Error: %s</span>`,
		gtkutils.Escape(l.LastError.Error()),
	))
}

func (l *Login) Retry() {
	l.login(false)
}

func (l *Login) Login() {
	l.login(true)
}

func (l *Login) login(readForm bool) {
	window.Blur()

	if readForm {
		l.LastToken = l.Token.Text()
	}

	l.tryLoggingIn(func(err error) {
		if err != nil {
			log.Errorln("failed to login:", err)
			l.error(err)
		}

		window.Unblur()
	})
}

func (l *Login) DiscordLogin() {
	window.Blur()

	l.discordLogin(func(err error) {
		if err != nil {
			log.Errorln("Failed to login:", err)
			l.error(err)
		}

		window.Unblur()
	})
}

func (l *Login) discordLogin(f func(error)) {
	go func() {
		onErr := func(err error) {
			glib.IdleAdd(func() { f(err) })
		}

		path, err := LookPathExtras("discordlogin")
		if err != nil {
			// Open the GitHub page to DiscordLogin in the browser.
			go openDiscordLoginPage()

			onErr(ErrDLNotFound)
			return
		}

		cmd := exec.Command(path)
		cmd.Stderr = os.Stderr

		// UI will actually block during this time.

		b, err := cmd.Output()
		if err != nil {
			onErr(errors.Wrap(err, "DiscordLogin failed"))
			return
		}

		if len(b) == 0 {
			onErr(errors.New("DiscordLogin returned nothing, check Console."))
			return
		}

		glib.IdleAdd(func() {
			l.LastToken = string(b)
			l.tryLoggingIn(f)
		})
	}()
}

// endgame function
func (l *Login) tryLoggingIn(f func(error)) {
	token := l.LastToken

	go func() {
		onErr := func(err error) {
			glib.IdleAdd(func() { f(err) })
		}

		s, err := state.New(token)
		if err != nil {
			onErr(errors.Wrap(err, "error creating new state"))
			return
		}

		n, err := ningen.FromState(s)
		if err != nil {
			onErr(errors.Wrap(err, "ningen"))
			return
		}

		if err := n.Open(); err != nil {
			onErr(errors.Wrap(err, "open"))
			return
		}

		glib.IdleAdd(func() {
			f(nil)
			l.finish(n)
		})
	}()
}

func openDiscordLoginPage() {
	if err := open.Run("https://github.com/diamondburned/discordlogin"); err != nil {
		log.Errorln("Failed to open URL to DiscordLogin:", err)
	}
}

func LookPathExtras(file string) (string, error) {
	// Add extra PATHs, just in case:
	paths := filepath.SplitList(os.Getenv("PATH"))

	if gobin := os.Getenv("GOBIN"); gobin != "" {
		paths = append(paths, gobin)
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		paths = append(paths, gopath)
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, "go", "bin"))
	}

	const filename = "discordlogin"

	for _, dir := range paths {
		if dir == "" {
			dir = "."
		}

		path := filepath.Join(dir, filename)
		if err := findExecutable(path); err == nil {
			return path, nil
		}
	}

	return "", exec.ErrNotFound
}

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}
