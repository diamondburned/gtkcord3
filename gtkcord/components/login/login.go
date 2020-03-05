package login

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/window"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/diamondburned/gtkcord3/ningen"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
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
	InfoButton *gtk.Button

	LastError error

	displayed bool
	finish    func(s *ningen.State)
}

func NewHeader() gtkutils.ExtendedWidget {
	l, _ := gtk.LabelNew("")
	l.SetMarkup(`<span weight="bold">gtkcord3 Login</span>`)
	l.SetMarginStart(50)
	return l
}

func NewLogin(finish func(s *ningen.State)) *Login {
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.SetMarginTop(15)
	main.SetMarginBottom(50)
	main.SetMarginStart(35)
	main.SetMarginEnd(35)
	main.SetSizeRequest(250, -1)
	main.SetVAlign(gtk.ALIGN_CENTER)

	err, _ := gtk.LabelNew("")
	err.SetSingleLineMode(false)
	err.SetLineWrap(true)
	err.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	err.SetMarginBottom(10)
	err.SetHAlign(gtk.ALIGN_START)
	err.SetMarginStart(2)
	err.SetMarginEnd(2)

	token, _ := gtk.EntryNew()
	token.SetMarginBottom(15)
	token.SetInputPurpose(gtk.INPUT_PURPOSE_PASSWORD)
	token.SetPlaceholderText("Token")
	token.SetVisibility(false)
	token.SetInvisibleChar('●')

	submit, _ := gtk.ButtonNewWithLabel("Login")
	submit.SetMarginBottom(15)
	gtkutils.InjectCSSUnsafe(submit, "login", `
		button.login {
			background-color: #7289da;
			color: #FFFFFF;
		}
	`)

	info, _ := gtk.ButtonNewWithLabel("Use DiscordLogin")
	gtkutils.InjectCSSUnsafe(info, "discordlogin", "")

	l := &Login{
		Box:        main,
		Token:      token,
		Submit:     submit,
		Error:      err,
		InfoButton: info,

		finish: finish,
	}

	token.Connect("activate", l.Login)
	submit.Connect("clicked", l.Login)
	info.Connect("clicked", l.DiscordLogin)

	main.Add(err)
	main.Add(token)
	main.Add(submit)
	main.Add(info)

	return l
}

func (l *Login) Display() {
	// Display the error if there's any:
	if l.LastError != nil {
		l.error(l.LastError)
	}

	if !l.displayed {
		window.Resize(500, 200)
		window.HeaderDisplay(NewHeader())
		window.Display(l)
		window.ShowAll()

		l.displayed = true
	}
}

func (l *Login) error(err error) {
	l.LastError = err

	l.Error.SetMarkup(fmt.Sprintf(
		`<span color="red">Error: %s</span>`,
		gtkutils.Escape(l.LastError.Error()),
	))
}

func (l *Login) Login() {
	l.Box.SetSensitive(false)
	defer l.Box.SetSensitive(true)

	if err := l.login(); err != nil {
		log.Errorln("Failed to login:", err)
		l.error(err)
		return
	}
}

func (l *Login) login() error {
	token, err := l.Token.GetText()
	if err != nil {
		return errors.Wrap(err, "Failed to get text")
	}

	return l.tryLoggingIn(token)
}

func (l *Login) DiscordLogin() {
	window.Blur()
	defer window.Unblur()

	if err := l.discordLogin(); err != nil {
		log.Errorln("Failed to login:", err)
		l.error(err)
		return
	}
}

func (l *Login) discordLogin() error {
	path, err := LookPathExtras("discordlogin")
	if err != nil {
		// Open the GitHub page to DiscordLogin in the browser.
		go openDiscordLoginPage()

		return ErrDLNotFound
	}

	cmd := &exec.Cmd{Path: path}
	cmd.Stderr = os.Stderr

	// UI will actually block during this time.

	b, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "DiscordLogin failed")
	}

	if len(b) == 0 {
		return errors.New("DiscordLogin returned nothing, check Console.")
	}

	return l.tryLoggingIn(string(b))
}

// endgame function
func (l *Login) tryLoggingIn(token string) error {
	s, err := CreateState(token)
	if err != nil {
		return err
	}

	// Finish with the callback:
	go l.finish(s)
	return nil
}

func openDiscordLoginPage() {
	if err := open.Run("https://github.com/diamondburned/discordlogin"); err != nil {
		log.Errorln("Failed to open URL to DiscordLogin:", err)
	}
}

func CreateState(token string) (*ningen.State, error) {
	return ningen.Connect(token)
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
