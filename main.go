package main

import (
	"flag"
	"os"
	"runtime/debug"

	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/login"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/keyring"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/pkg/errors"
	"github.com/pkg/profile"
)

func init() {
	// AGGRESSIVE GC
	debug.SetGCPercent(50)
}

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

func LoadKeyring() (*ningen.State, error) {
	// Check if env vars or flags are set:
	token := LoadToken()

	// If it is, override it in the keyring and use it:
	if token != "" {
		return ningen.Connect(token)
	}

	// Does the keyring have the token?
	token = keyring.Get()

	// Yes.
	if token != "" {
		return ningen.Connect(token)
	}

	// No.
	return nil, ErrTokenNotProvided
}

func Login(finish func(s *ningen.State)) error {
	s, err := LoadKeyring()
	if err == nil {
		go finish(s)
		return nil
	}

	// No, so we need to display the login window:
	log.Println("Summoning the Login window")
	semaphore.IdleMust(func() {
		var l = login.NewLogin(finish)
		if err != ErrTokenNotProvided {
			l.LastError = err
		}
		l.Display()
	})

	return nil
}

func Finish(a *gtkcord.Application) func(s *ningen.State) {
	return func(s *ningen.State) {
		// Store the token:
		keyring.Set(s.Token)

		if err := a.Ready(s); err != nil {
			log.Fatalln("Failed to get gtkcord ready:", err)
		}
	}
}

func main() {
	defer profile.Start(profile.BlockProfile).Stop()

	// Spawn a new window:
	if err := window.Init(); err != nil {
		log.Fatalln("Failed to initialize Gtk3 window:", err)
	}

	v, err := semaphore.Idle(gtkcord.New)
	if err != nil {
		log.Fatalln("Can't create a Gtk3 window:", err)
	}
	a := v.(*gtkcord.Application)

	// Try and log in:
	if err := Login(Finish(a)); err != nil {
		log.Fatalln("Failed to login:", err)
	}

	// Block until gtkcord dies:
	window.Wait()
}
