package cache

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"sync"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
)

type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(
		c.p.X-c.r,
		c.p.Y-c.r,
		c.p.X+c.r,
		c.p.Y+c.r,
	)
}

func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

var bufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

type Processor func(image.Image) image.Image

func ProcessAnimation(data []byte, processors ...Processor) []byte {
	GIF, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return data
	}

	// Encode the GIF frame-by-frame
	for _, frame := range GIF.Image {
		var img = image.Image(frame)
		for _, proc := range processors {
			img = proc(img)
		}

		for x := 0; x < img.Bounds().Dx(); x++ {
			for y := 0; y < img.Bounds().Dy(); y++ {
				frame.Set(x, y, img.At(x, y))
			}
		}
	}

	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	buf.Reset()

	if err := gif.EncodeAll(buf, GIF); err != nil {
		return data
	}

	return buf.Bytes()
}

var pngEncoder = png.Encoder{
	CompressionLevel: png.BestSpeed,
}

func Process(data []byte, processors ...Processor) []byte {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}

	for _, proc := range processors {
		img = proc(img)
	}

	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	buf.Reset()

	if err := pngEncoder.Encode(buf, img); err != nil {
		return data
	}

	return buf.Bytes()
}

func Prepend(p1 Processor, pN ...Processor) []Processor {
	return append([]Processor{p1}, pN...)
}

func Resize(w, h int) Processor {
	return func(img image.Image) image.Image {
		return imaging.Fit(img, w, h, imaging.CatmullRom)
	}
}

// Round round-crops an image
func Round(src image.Image) image.Image {
	r := src.Bounds().Dx() / 2

	var dst = image.NewRGBA(image.Rect(
		0, 0,
		r*2, r*2,
	))

	draw.DrawMask(
		dst,
		src.Bounds(),
		src,
		image.ZP,
		&circle{
			p: image.Point{X: r, Y: r},
			r: r,
		},
		image.ZP,
		draw.Src,
	)

	return image.Image(dst)
}
