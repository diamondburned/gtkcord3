package cache

import (
	"fmt"
	"io"
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

// DO NOT TOUCH.
const (
	CacheHash   = "Aethel1s"
	CachePrefix = "gtkcord3"
)

var (
	DirName = CachePrefix + "-" + CacheHash
	Temp    = os.TempDir()
	Path    = filepath.Join(Temp, DirName)
)

var store *diskv.Diskv

func init() {
	cleanUpCache()

	store = diskv.New(diskv.Options{
		BasePath:          Path,
		AdvancedTransform: TransformURL,
		InverseTransform:  InverseTransformURL,
	})
}

func cleanUpCache() {
	tmp, err := os.Open(Temp)
	if err != nil {
		return
	}

	dirs, err := tmp.Readdirnames(-1)
	if err != nil {
		return
	}

	for _, d := range dirs {
		if strings.HasPrefix(d, CachePrefix) && d != DirName {
			path := filepath.Join(Temp, d)
			log.Infoln("Deleting", path)
			os.RemoveAll(path)
		}
	}
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
	b, err := get(url, "")
	if err != nil {
		return b, err
	}

	return b, nil
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
	return GetPixbufScaled(url, 0, 0, pp...)
}

func GetPixbufScaled(url string, w, h int, pp ...Processor) (*gdk.Pixbuf, error) {
	b, err := get(url, "image")
	if err != nil {
		return nil, err
	}

	if len(pp) > 0 {
		b = Process(b, pp...)
	}

	v, err := semaphore.Idle(func() (*gdk.Pixbuf, error) {
		l, err := gdk.PixbufLoaderNew()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create a pixbuf_loader")
		}

		if w > 0 && h > 0 {
			l.SetSize(w, h)
		}

		pixbuf, err := l.WriteAndReturnPixbuf(b)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load pixbuf")
		}

		return pixbuf, nil
	})

	if err != nil {
		return nil, err
	}

	return v.(*gdk.Pixbuf), nil
}

func SetImage(url string, img *gtk.Image, pp ...Processor) error {
	return SetImageScaled(url, img, 0, 0, pp...)
}

func SetImageScaled(url string, img *gtk.Image, w, h int, pp ...Processor) error {
	b, err := get(url, "image")
	if err != nil {
		return err
	}

	if len(pp) > 0 {
		b = Process(b, pp...)
	}

	v, _ := semaphore.Idle(func() error {
		l, err := gdk.PixbufLoaderNew()
		if err != nil {
			return errors.Wrap(err, "Failed to create a pixbuf_loader")
		}

		if w > 0 && h > 0 {
			l.SetSize(w, h)
		}

		l.Connect("area-updated", func() {
			p, err := l.GetPixbuf()
			if err != nil {
				log.Errorln("Failed to get pixbuf during area-updated:", err)
				return
			}

			img.SetFromPixbuf(p)
		})

		if _, err := l.Write(b); err != nil {
			return errors.Wrap(err, "Failed to write to pixbuf_loader")
		}

		if err := l.Close(); err != nil {
			return errors.Wrap(err, "Failed to close pixbuf_loader")
		}

		return nil
	})

	if v != nil {
		return v.(error)
	}

	return nil
}

// SetImageAsync is not cached.
func SetImageAsync(url string, img *gtk.Image, w, h int) error {
	r, err := Client.Get(url)
	if err != nil {
		return errors.Wrap(err, "Failed to GET "+url)
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		return fmt.Errorf("Bad status code %d for %s", r.StatusCode, url)
	}

	var gif = strings.Contains(url, ".gif")
	var l *gdk.PixbufLoader

	v, _ := semaphore.Idle(func() (err error) {
		l, err = gdk.PixbufLoaderNew()
		if err != nil {
			return errors.Wrap(err, "Failed to create a pixbuf_loader")
		}

		if w > 0 && h > 0 {
			l.SetSize(w, h)
		}

		l.Connect("area-prepared", func() {
			if gif {
				p, err := l.GetAnimation()
				if err != nil {
					log.Errorln("Failed to get pixbuf during area-prepared:", err)
					return
				}
				img.SetFromAnimation(p)

			} else {
				p, err := l.GetPixbuf()
				if err != nil {
					log.Errorln("Failed to get animation during area-prepared:", err)
					return
				}
				img.SetFromPixbuf(p)
			}
		})

		return nil
	})

	if v != nil {
		return v.(error)
	}

	if _, err := io.Copy(l, r.Body); err != nil {
		return errors.Wrap(err, "Failed to stream to pixbuf_loader")
	}

	if v, _ := semaphore.Idle(l.Close); v != nil {
		return errors.Wrap(v.(error), "Failed to close pixbuf_loader")
	}

	return nil
}
