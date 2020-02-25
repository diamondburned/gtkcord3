package main

import (
	"flag"
	"os"

	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/gtkcord/login"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/gtkcord/window"
	"github.com/diamondburned/gtkcord3/keyring"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/diamondburned/gtkcord3/ningen"
	"github.com/pkg/errors"
)

var ErrTokenNotProvided = errors.New("Token not in -t, $TOKEN, or keyring")

func LoadToken() string {
	var token = os.Getenv("TOKEN")
	if token != "" {
		return token
	}

	flag.StringVar(&token, "t", "", "Token")
	flag.Parse()

	return token
}

func LoadKeyring() (*state.State, error) {
	// Check if env vars or flags are set:
	token := LoadToken()

	// If it is, override it in the keyring and use it:
	if token != "" {
		return login.CreateState(token)
	}

	// Does the keyring have the token?
	token = keyring.Get()

	// Yes.
	if token != "" {
		return login.CreateState(token)
	}

	// No.
	return nil, ErrTokenNotProvided
}

func Login(finish func(s *state.State)) error {
	s, err := LoadKeyring()
	if err == nil {
		go finish(s)
		return nil
	}

	// No, so we need to display the login window:
	var l = semaphore.IdleMust(login.NewLogin, finish).(*login.Login)
	if err != ErrTokenNotProvided {
		l.LastError = err
	}

	semaphore.IdleMust(l.Display)

	return nil
}

func Finish(s *state.State) {
	// Store the token:
	keyring.Set(s.Token)

	n, err := ningen.Ningen(s)
	if err != nil {
		log.Fatalln("Failed to start the Discord wrapper:", err)
	}

	if err := gtkcord.Ready(n); err != nil {
		log.Fatalln("Failed to get gtkcord ready:", err)
	}
}

func main() {
	// Spawn a new window:
	if err := window.Init(); err != nil {
		log.Fatalln("Failed to initialize Gtk3 window:", err)
	}

	// Spawn the spinning circle:
	if err := semaphore.IdleMust(gtkcord.Init); err != nil {
		log.Fatalln("Can't create a Gtk3 window:", err.(error))
	}

	// Try and log in:
	if err := Login(Finish); err != nil {
		log.Fatalln("Failed to login:", err)
	}

	// Block until gtkcord dies:
	window.Wait()
}
