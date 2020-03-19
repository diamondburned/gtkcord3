package cache

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var Client = http.Client{
	Timeout: 15 * time.Second,
}

// DO NOT TOUCH.
const (
	CacheHash   = "astolfo"
	CachePrefix = "gtkcord3"
)

var (
	DirName = CachePrefix + "-" + CacheHash
	Temp    = os.TempDir()
	Path    = filepath.Join(Temp, DirName)
)

// var store *diskv.Diskv

func init() {
	cleanUpCache()
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
		if strings.HasPrefix(d, CachePrefix+"-") && d != DirName {
			path := filepath.Join(Temp, d)
			log.Infoln("Deleting", path)
			os.RemoveAll(path)
		}
	}
}

func TransformURL(s string, w, h int, gif bool) string {
	var sizeSuffix string
	if w > 0 && h > 0 && gif {
		sizeSuffix = "_sz" + strconv.Itoa(w) + "x" + strconv.Itoa(h)
	}

	u, err := url.Parse(s)
	if err != nil {
		return filepath.Join(Path, SanitizeString(s)+sizeSuffix)
	}

	path := filepath.Join(Path, u.Hostname())

	if err := os.MkdirAll(path, 0755|os.ModeDir); err != nil {
		log.Errorln("Failed to mkdir:", err)
	}

	return filepath.Join(path, SanitizeString(u.EscapedPath()+"?"+u.RawQuery)+sizeSuffix)
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

// var fileIO sync.Mutex

func download(url string, pp []Processor, gif bool) ([]byte, error) {
	r, err := Client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to GET")
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		return nil, fmt.Errorf("Bad status code %d for %s", r.StatusCode, url)
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to download image")
	}

	if len(b) == 0 {
		return nil, errors.New("nil body")
	}

	if len(pp) > 0 {
		if gif {
			b, err = ProcessAnimation(b, pp)
		} else {
			b, err = Process(b, pp)
		}
	}

	return b, err
}

// get doesn't check if the file exists
func get(url, dst string, pp []Processor, gif bool) error {
	// Unlock FileIO mutex to allow concurrent requests.
	// fileIO.Unlock()
	// defer fileIO.Lock()

	b, err := download(url, pp, gif)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(dst, b, 0755); err != nil {
		return errors.Wrap(err, "Failed to write file to "+dst)
	}

	return nil
}

func GetPixbuf(url string, pp ...Processor) (*gdk.Pixbuf, error) {
	return GetPixbufScaled(url, 0, 0, pp...)
}

func GetPixbufScaled(url string, w, h int, pp ...Processor) (*gdk.Pixbuf, error) {
	// Transform URL:
	dst := TransformURL(url, w, h, false)

	// fileIO.Lock()
	// defer fileIO.Unlock()

	// Try and get the Pixbuf from file:
	p, err := gdk.PixbufNewFromFileAtScale(dst, w, h, true)
	if err == nil {
		return p, nil
	}

	// If resize is requested, we resize using Go's instead.
	// if w > 0 && h > 0 {
	// 	pp = append(pp, Resize(w, h))
	// }

	// Get the image into file (dst)
	if err := get(url, dst, pp, false); err != nil {
		return nil, err
	}

	p, err = gdk.PixbufNewFromFileAtScale(dst, w, h, true)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get pixbuf")
	}

	return p, nil
}

func SetImage(url string, img *gtk.Image, pp ...Processor) error {
	return SetImageScaled(url, img, 0, 0, pp...)
}

func SetImageScaled(url string, img *gtk.Image, w, h int, pp ...Processor) error {
	// Transform URL:
	gif := strings.Contains(url, "gif")
	dst := TransformURL(url, w, h, gif)

	// fileIO.Lock()
	// defer fileIO.Unlock()

	// Try and get the Pixbuf from file:
	if !gif {
		p, err := gdk.PixbufNewFromFileAtScale(dst, w, h, true)
		if err == nil {
			semaphore.IdleMust(img.SetFromPixbuf, p)
			return nil
		}
	} else {
		p, err := gdk.PixbufAnimationNewFromFile(dst)
		if err == nil {
			semaphore.IdleMust(img.SetFromAnimation, p)
			return nil
		}
	}

	// If resize is requested, we resize using Go's instead.
	// Only use this for GIF animations.
	if w > 0 && h > 0 && gif {
		pp = append(pp, Resize(w, h))
	}

	// Get the image into file (dst)
	if err := get(url, dst, pp, gif); err != nil {
		return err
	}

	if !gif {
		p, err := gdk.PixbufNewFromFileAtScale(dst, w, h, true)
		if err != nil {
			os.Remove(dst)
			return errors.Wrap(err, "Failed to get pixbuf after downloading")
		}
		semaphore.IdleMust(img.SetFromPixbuf, p)
	} else {
		p, err := gdk.PixbufAnimationNewFromFile(dst)
		if err != nil {
			os.Remove(dst)
			return errors.Wrap(err, "Failed to get animation after downloading")
		}
		semaphore.IdleMust(img.SetFromAnimation, p)
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

	l, err = gdk.PixbufLoaderNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create a pixbuf_loader")
	}

	if w > 0 && h > 0 {
		gtkutils.Connect(l, "size-prepared", func(_ *glib.Object, imgW, imgH int) {
			l.SetSize(maxSize(imgW, imgH, w, h))
		})
	}

	var p interface{}
	var pMu sync.Mutex

	gtkutils.Connect(l, "area-prepared", func() {
		pMu.Lock()
		defer pMu.Unlock()

		if gif {
			p, err = l.GetAnimation()
			if err != nil || p == nil {
				log.Errorln("Failed to get animation during area-prepared:", err)
				return
			}
		} else {
			p, err = l.GetPixbuf()
			if err != nil || p == nil {
				log.Errorln("Failed to get pixbuf during area-prepared:", err)
				return
			}
		}
	})

	gtkutils.Connect(l, "area-updated", func() {
		pMu.Lock()
		defer pMu.Unlock()

		switch {
		case p == nil:
			return
		case gif:
			semaphore.IdleMust(img.SetFromAnimation, p)
		default:
			semaphore.IdleMust(img.SetFromPixbuf, p)
		}
	})

	if err != nil {
		return err
	}

	defer l.Close()

	if _, err := io.Copy(l, r.Body); err != nil {
		return errors.Wrap(err, "Failed to stream to pixbuf_loader")
	}

	return nil
}

func AsyncFetch(url string, img *gtk.Image, w, h int, pp ...Processor) {
	semaphore.IdleMust(gtkutils.ImageSetIcon, img, "image-missing", 24)

	go func() {
		var err error
		if len(pp) == 0 {
			err = SetImageAsync(url, img, w, h)
		} else {
			err = SetImageScaled(url, img, w, h, pp...)
		}
		if err != nil {
			log.Errorln("Failed to get image", url+":", err)
			return
		}
	}()
}

func maxSize(w, h, maxW, maxH int) (int, int) {
	if w < maxW && h < maxH {
		return w, h
	}

	if w > h {
		h = h * maxW / w
		w = maxW
	} else {
		w = w * maxH / h
		h = maxH
	}

	return w, h
}
