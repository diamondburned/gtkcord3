package tmp

import (
	"bytes"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// TempPath is populated on init().
var TempPath = filepath.Join(os.TempDir(), "gtkcord3")

func init() {
	os.MkdirAll(TempPath, os.ModeDir|os.ModePerm)
}

func joinPath(name string) string {
	return filepath.Join(TempPath, name)
}

func Store(name string, r io.Reader) error {
	f, err := os.OpenFile(joinPath(name),
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "Failed to open file "+name)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return errors.Wrap(err, "Failed to write to file "+name)
	}

	return nil
}

func StoreBytes(name string, b []byte) error {
	return Store(name, bytes.NewReader(b))
}

func get(name string) (*os.File, error) {
	f, err := os.Open(joinPath(name))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open file "+name)
	}
	return f, nil
}

func Get(name string) (io.ReadCloser, error) {
	return get(name)
}

func GetBytes(name string) ([]byte, error) {
	f, err := Get(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read file "+name)
	}

	return b, nil
}

var HTTP = http.Client{
	Timeout: 10 * time.Second,
}

func download(link string) (*os.File, string, error) {
	// Hash the URL, because we're lazy:
	var hashBytes = sha1.Sum([]byte(link))
	var hash = base32.StdEncoding.EncodeToString(hashBytes[:])

	// Try and get the path:
	u, err := url.Parse(link)
	if err != nil {
		return nil, "", errors.Wrap(err, "Invalid URL")
	}
	var ext = path.Ext(u.Path)

	var filepath = joinPath(hash + ext)

	if f, err := get(filepath); err == nil {
		return f, filepath, nil
	}

	r, err := HTTP.Get(u.String())
	if err != nil {
		return nil, "", errors.Wrap(err, "Failed to GET URL "+link)
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		return nil, "",
			fmt.Errorf("Bad status code %d for %s", r.StatusCode, link)
	}

	f, err := os.OpenFile(
		filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return nil, "", errors.Wrap(err, "Failed to open file to download to")
	}

	if _, err := io.Copy(f, r.Body); err != nil {
		return nil, "", errors.Wrap(err, "Failed to download to file")
	}

	return f, filepath, nil
}

func DownloadToPath(link string) (string, error) {
	_, path, err := download(link)
	return path, err
}

func Download(link string) (io.ReadCloser, error) {
	f, _, err := download(link)
	if err != nil {
		return nil, err
	}

	// Seek the file back to prepare it to be read from:
	if _, err := f.Seek(0, 0); err != nil {
		return nil, errors.Wrap(err, "Failed to prepare file for reading")
	}

	return f, nil
}
