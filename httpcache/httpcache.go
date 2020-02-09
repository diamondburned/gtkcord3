package httpcache

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/pkg/errors"
)

var cache sync.Map
var Client = http.DefaultClient

var MaxCacheSize = 1 * 1024 * 1024 // 4MB

func Delete(url string) {
	cache.Delete(url)
}

func Store(url string, data []byte) {
	cache.Store(url, data)
}

func Get(url string) []byte {
	v, ok := cache.Load(url)
	if ok {
		return v.([]byte)
	}
	return nil
}

func HTTPGet(url string) ([]byte, error) {
	if b := Get(url); b != nil {
		return b, nil
	}

	r, err := Client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to GET URL "+url)
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		return nil, fmt.Errorf("Bad status code %d for %s", r.StatusCode, url)
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to download image")
	}

	if len(b) > MaxCacheSize {
		return b, nil
	}

	Store(url, b)
	return b, nil
}
