package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

var Path string

func init() {
	// Load the config dir:
	d, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln("Failed to get config dir:", err)
	}

	// Fill Path:
	Path = filepath.Join(d, "gtkcord")

	// Ensure it exists:
	if err := os.Mkdir(Path, 0755|os.ModeDir); err != nil && !os.IsExist(err) {
		log.Fatalln("Failed to make config dir:", err)
	}
}

// MustRead ensures the config directory actually exists.
func MustRead(dir string) (files []os.FileInfo, path string, err error) {
	// Make a full path:
	dir = filepath.Join(Path, dir)

	f, err := ioutil.ReadDir(dir)
	if err == nil {
		return f, dir, nil
	}

	// If the error is not a "does not exist" error, that means something is
	// wrong.
	if !os.IsNotExist(err) {
		return nil, "", errors.Wrap(err, "Failed to read directory")
	}

	// Directory doesn't exist, try making it:
	if err := os.Mkdir(dir, 0755|os.ModeDir); err != nil && !os.IsExist(err) {
		return nil, "", errors.Wrap(err, "Failed to make directory")
	}

	return []os.FileInfo{}, dir, nil
}

func MarshalToFile(file string, from interface{}) error {
	file = filepath.Join(Path, file)

	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_SYNC|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrap(err, "Failed to open file")
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")

	if err := enc.Encode(from); err != nil {
		return errors.Wrap(err, "Failed to marshal given struct")
	}

	return nil
}

func UnmarshalFromFile(file string, to interface{}) error {
	file = filepath.Join(Path, file)

	f, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		// Ignore does not exist error, leave struct as it is.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(to); err != nil {
		return errors.Wrap(err, "Failed to unmarshal to given struct")
	}

	return nil
}
