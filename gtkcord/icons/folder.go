package icons

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"log"
)

//go:embed folder.png
var folderPNG []byte

var folderImage *image.RGBA

func init() {
	i, err := png.Decode(bytes.NewReader(folderPNG))
	if err != nil {
		log.Panicln("folderPNG error:", err)
	}
	folderImage = i.(*image.RGBA)
}

func Folder(color uint32) *image.RGBA {
	img := image.RGBA{
		Pix:    append([]uint8(nil), folderImage.Pix...),
		Stride: folderImage.Stride,
		Rect:   folderImage.Rect,
	}

	cell := [4]byte{
		uint8((color >> 16) & 255),
		uint8((color >> 8) & 255),
		uint8(color & 255),
		0xFF,
	}

	for i := 0; i < len(img.Pix); i += 4 {
		// 0R 1G 2B 4A
		alpha := img.Pix[i+3]
		if alpha == 0 {
			continue
		}

		cell[3] = alpha
		copy(img.Pix, cell[:])
	}

	return &img
}
