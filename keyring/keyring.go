package keyring

import (
	"github.com/diamondburned/gtkcord3/log"
	"github.com/zalando/go-keyring"
)

func Get() string {
	k, err := keyring.Get("gtkcord", "token")
	if err != nil {
		log.Errorln("[non-fatal] Failed to get Gtkcord token from keyring")
	}

	return k
}

func Set(token string) {
	if err := keyring.Set("gtkcord", "token", token); err != nil {
		log.Errorln("[non-fatal] Failed to set Gtkcord token to keyring")
	}
}
