package cache

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/peterbourgon/diskv/v3"
	"github.com/pkg/errors"
)

var Client = http.Client{
	Timeout: 15 * time.Second,
}

var Path = filepath.Join(os.TempDir(), "gtkcord3")

var store *diskv.Diskv

func init() {
	store = diskv.New(diskv.Options{
		BasePath:          Path,
		AdvancedTransform: TransformURL,
		InverseTransform:  InverseTransformURL,
		CacheSizeMax:      5 * 1024 * 1024, // 5MB
	})
}

func TransformURL(s string) *diskv.PathKey {
	u, err := url.Parse(s)
	if err != nil {
		return &diskv.PathKey{
			FileName: SanitizeString(s),
		}
	}

	return &diskv.PathKey{
		FileName: SanitizeString(u.EscapedPath() + "?" + u.RawQuery),
		Path:     []string{u.Hostname()},
	}
}

func InverseTransformURL(pk *diskv.PathKey) string {
	// like fuck do I know
	return ""
}

// SanitizeString makes the string friendly to put into the file system. It
// converts anything that isn't a digit or letter into underscores.
func SanitizeString(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '#' {
			return r
		}

		return '_'
	}, str)
}

func Get(url string) ([]byte, error) {
	return get(url, "")
}

func get(url, suffix string) ([]byte, error) {
	if suffix != "" {
		suffix = "#" + suffix
	}

	b, err := store.Read(url + suffix)
	if err == nil {
		return b, nil
	}

	r, err := Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		return nil, fmt.Errorf("Bad status code %d for %s", r.StatusCode, url)
	}

	b, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to download image")
	}

	if len(b) == 0 {
		return nil, errors.New("nil body")
	}

	if err := store.Write(url+suffix, b); err != nil {
		log.Errorln("Failed to store:", err)
	}

	return b, nil
}

func GetPixbuf(url string, pp ...Processor) (*gdk.Pixbuf, error) {
	b, err := get(url, "image")
	if err != nil {
		return nil, err
	}

	if len(pp) > 0 {
		b = Process(b, pp...)
	}

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a pixbuf_loader")
	}

	pixbuf, err := l.WriteAndReturnPixbuf(b)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load pixbuf")
	}

	return pixbuf, nil
}

func SetImage(url string, img *gtk.Image, pp ...Processor) error {
	b, err := get(url, "image")
	if err != nil {
		return err
	}

	if len(pp) > 0 {
		b = Process(b, pp...)
	}

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create a pixbuf_loader")
	}

	if _, err := l.Write(b); err != nil {
		l.Unref()
		return errors.Wrap(err, "Failed to write to pixbuf_loader")
	}

	if err := l.Close(); err != nil {
		l.Unref()
		return errors.Wrap(err, "Failed to close pixbuf_loader")
	}

	p, err := l.GetPixbuf()
	if err != nil {
		l.Unref()
		return errors.Wrap(err, "Failed to get pixbuf from pixbuf_loader")
	}

	semaphore.IdleMust(img.SetFromPixbuf, p)

	return nil
}

func SetAnimation(url string, img *gtk.Image, pp ...Processor) error {
	b, err := get(url, "animation")
	if err != nil {
		return err
	}

	if len(pp) > 0 {
		b = ProcessAnimation(b, pp...)
	}

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create a pixbuf_loader")
	}

	if _, err := l.Write(b); err != nil {
		l.Unref()
		return errors.Wrap(err, "Failed to write to pixbuf_loader")
	}

	if err := l.Close(); err != nil {
		l.Unref()
		return errors.Wrap(err, "Failed to close pixbuf_loader")
	}

	p, err := l.GetAnimation()
	if err != nil {
		l.Unref()
		return errors.Wrap(err, "Failed to get pixbuf from pixbuf_loader")
	}

	semaphore.IdleMust(img.SetFromAnimation, p)

	return nil
}
