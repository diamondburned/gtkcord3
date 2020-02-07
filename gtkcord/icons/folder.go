package icons

import "image"

// base64 encoded PNG
const folderIconData = `iVBORw0KGgoAAAANSUhEUgAAAIAAAACACAYAAADDPmHLAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAABOvQAATr0Bc2poFAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAMtSURBVHic7d2/a11lGMDx7ylVtCEggim0LhKdlIC4CBLsYh3qoIUiTi7iVoQiiJODxKF0EMTRgnSqSvwPHDIkQqFTh+LUwUYTwUHBmJLyOFwXf3Lek9z7vvc83w+c7X1vHs77JTm5uRCQJEmSJEmSJEmSJEmSJEnS3OuGbAp4DHgDeBZYONKJyvwK3AW2gG862K84Sw4Brwf8EhCNXT8HfBTwSO17NFoBLwccNHDY/3d9H/Bi7Xs1OgFdwO0GDrjPtR/wWu17NioBKw0cbMm1F/Bc7fvWumMFa5+Y2hTT8RDwecADtQdpWUkAD05tiul5Gni/9hAtKwlgXn0QcLb2EK3KEMAx4FrA6dqDtChDAABLwFc+D/xTlgAAngfWag/RmkwBALwbcL72EC3JFkAHfBawXHuQVmQLACZ/K1gPeLj2IC3IGADACvBx7SFakDUAgLcD3qw9RG3Haw9Q2acB52oPUWAX+AHYADY7uH/YF8wewAJwofYQA+0GXAE+6eD3oS+S+UfAvFsCLgO3Ap4Z+iIGMP+Wgc2hH4IxgHFYZPKr7ZOlGw1gPB4FrpZuMoBxWQ14pWSDAYzPWyWLDWB8Xip5m9sAxucEBQ+DBjBOvT/9ZADjtNh3oQEkZwDJGUByBpCcASRnAMkZQHIGkJwBJGcAyRlAcgaQnAEkZwDJGUByBpCcASRnAMkZQHIGkJwBJGcAyRlAcgaQnAEkZwDJGUByBpCcASRnAMkZQHIGkJwBJGcAyRlAcgaQnAEkZwDJGUByBpCcASRnAMkZQHIGkJwBJGcAyRlAcgaQnAEkZwDJGUByBpBcSQD3pjaFjlrvsyoJ4McBg6iO7b4LSwK4A0TxKJq1YHJWvfQOoIMd4OaAgTRbNzr4qe/i0ofA64XrNXtflCzuShb/+U+JvwMeL9mnmdkGnurgt74bir4DdLAHXMJngRYFcLHk8GHA+wAdfAmsle7T1H3YwfrMvlrAOwEHAeFV9ToIeG9mB/+3CFYDvm3gJmS9tgJeOMwZFj0E/kcEHXAGeBVYBU4BJw/7uvpXO0we9DaAr4GNDp/HJEmSJEmSJEmSJEnSX/0BhUkn0FnyE5kAAAAASUVORK5CYII`

var folderIcon = FromPNG(folderIconData).(*image.NRGBA)

func Folder(color uint32) image.Image {
	copy := &image.NRGBA{
		Pix:    append([]uint8{}, folderIcon.Pix...),
		Stride: folderIcon.Stride,
		Rect:   folderIcon.Rect,
	}

	var (
		r = uint8((color >> 16) & 255)
		g = uint8((color >> 8) & 255)
		b = uint8(color & 255)
	)

	for i := 0; i < len(copy.Pix); i += 4 {
		// 0R 1G 2B 4A
		if copy.Pix[i+3] == 0 {
			continue
		}

		copy.Pix[i+0] = r
		copy.Pix[i+1] = g
		copy.Pix[i+2] = b
	}

	return copy
}
