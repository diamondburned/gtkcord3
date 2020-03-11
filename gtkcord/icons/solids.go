package icons

import (
	"image"
)

// func SolidCircle(sz int, hex uint32) image.Image {
// 	img := image.NewRGBA(image.Rect(0, 0, sz, sz))
// 	rad := sz / 2
// 	r, g, b := discord.Color(hex).RGB()

// 	for x := 0; x < sz; x++ {
// 		for y := 0; y < sz; y++ {
// 			if CircleAt(rad, x, y) {
// 				img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
// 			}
// 		}
// 	}

// 	return img
// }

func Solid(w, h int, r, g, b, a uint8) image.Image {
	i := image.NewNRGBA(image.Rect(0, 0, w, h))
	for j := 0; j < len(i.Pix); j += 4 {
		i.Pix[j+0] = r
		i.Pix[j+1] = g
		i.Pix[j+2] = b
		i.Pix[j+3] = a
	}
	return i
}
