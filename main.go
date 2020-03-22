package main

import (
	"flag"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/login"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/plugin"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/keyring"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/pkg/errors"

	// Profiler
	_ "net/http/pprof"
)

var profile bool

func init() {
	flag.BoolVar(&profile, "prof", false, "Enable the profiler")

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

		if err := plugin.StartPlugins(a); err != nil {
			log.Fatalln("Failed to initialize plugins:", err)
		}
	}
}

func main() {
	a, err := gtkcord.New()
	if err != nil {
		log.Fatalln("Failed to start gtkcord:", err)
	}

	a.Start()
	defer a.Wait()

	// Try and log in:
	if err := Login(Finish(a)); err != nil {
		log.Fatalln("Failed to login:", err)
	}

	if profile {
		// Profiler
		runtime.SetMutexProfileFraction(5)   // ???
		runtime.SetBlockProfileRate(5000000) // 5ms
		go http.ListenAndServe("localhost:6969", nil)
	}
}
