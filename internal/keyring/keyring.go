package keyring

import (
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/zalando/go-keyring"
)

func Get() string {
	k, err := keyring.Get("gtkcord3", "token")
	if err != nil {
		log.Errorln("[non-fatal] Failed to get Gtkcord token from keyring")
	}

	if k == "" {
		log.Infoln("Keyring token is empty.")
	}

	return k
}

func Set(token string) {
	if err := keyring.Set("gtkcord3", "token", token); err != nil {
		log.Errorln("[non-fatal] failed to set Gtkcord token to keyring")
	}
}

func Delete() {
	if err := keyring.Delete("gtkcord3", "token"); err != nil {
		log.Errorln("[non-fatal] failed to delete Gtkcord token from keyring")
	}
}
