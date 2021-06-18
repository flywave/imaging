package imaging

import (
	"image"
	"image/color"
)

func affineTransform(x, y float64, data []float64) (xout, yout float64) {
	a := data[:]
	a0 := a[0]
	a1 := a[1]
	a2 := a[2]
	a3 := a[3]
	a4 := a[4]
	a5 := a[5]

	xin := float64(x) + 0.5
	yin := float64(y) + 0.5

	xout = a0*xin + a1*yin + a2
	yout = a3*xin + a4*yin + a5

	return
}

func perspectiveTransform(x, y float64, data []float64) (xout, yout float64) {
	a := data[:]
	a0 := a[0]
	a1 := a[1]
	a2 := a[2]
	a3 := a[3]
	a4 := a[4]
	a5 := a[5]
	a6 := a[6]
	a7 := a[7]

	xin := float64(x) + 0.5
	yin := float64(y) + 0.5

	xout = (a0*xin + a1*yin + a2) / (a6*xin + a7*yin + 1)
	yout = (a3*xin + a4*yin + a5) / (a6*xin + a7*yin + 1)

	return
}

func quadTransform(x, y float64, data []float64) (xout, yout float64) {
	a := data[:]
	a0 := a[0]
	a1 := a[1]
	a2 := a[2]
	a3 := a[3]
	a4 := a[4]
	a5 := a[5]
	a6 := a[6]
	a7 := a[7]

	xin := float64(x) + 0.5
	yin := float64(y) + 0.5

	xout = a0 + a1*xin + a2*yin + a3*xin*yin
	yout = a4 + a5*xin + a6*yin + a7*xin*yin

	return
}

type ImagingTransformMap func(x, y float64, data []float64) (xout, yout float64)

func genericTransform(img image.Image, outImage *image.NRGBA, x0, y0, x1, y1 float64, transform ImagingTransformMap, data []float64, filter ResampleFilter) {
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()

	if srcW <= 0 || srcH <= 0 {
		return
	}

	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > float64(srcW) {
		x1 = float64(srcW)
	}
	if y1 > float64(srcH) {
		y1 = float64(srcH)
	}

	if x0 == 0 && y0 == 0 && float64(srcW) == x1 && float64(srcH) == y1 {
		*outImage = *Clone(img)
	}
	dstW, dstH := int(x1-x0), int(y1-y0)
	var dst *image.NRGBA
	if outImage == nil {
		dst = image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	} else {
		dst = outImage
	}
	src := newScanner(img)
	var xx, yy float64
	weights := precomputeWeights(dstH, srcH, filter)
	parallel(int(y0), int(y1), func(ys <-chan int) {
		scanLine := make([]uint8, dstH*4)
		for y := range ys {
			for x := x0; x < x1; x++ {
				xx, yy = transform(x-x0, float64(y)-y0, data)
				src.scan(int(xx), int(yy), int(xx)+1, int(yy)+1, scanLine)
				for y := range weights {
					var r, g, b, a float64
					for _, w := range weights[y] {
						i := w.index * 4
						s := scanLine[i : i+4 : i+4]
						aw := float64(s[3]) * w.weight
						r += float64(s[0]) * aw
						g += float64(s[1]) * aw
						b += float64(s[2]) * aw
						a += aw
					}
					if a != 0 {
						aInv := 1 / a
						j := y*dst.Stride + int(x)*4
						d := dst.Pix[j : j+4 : j+4]
						d[0] = clamp(r * aInv)
						d[1] = clamp(g * aInv)
						d[2] = clamp(b * aInv)
						d[3] = clamp(a)
					}
				}
			}
		}
	})
	outImage = dst
}

func imagingTransform(img image.Image, outImage *image.NRGBA, method TransformsMethod, x0, y0, x1, y1 float64, data []float64, filter ResampleFilter) {
	var transform ImagingTransformMap

	switch method {
	case AFFINE:
		transform = affineTransform
		break
	case PERSPECTIVE:
		transform = perspectiveTransform
		break
	case QUAD:
		transform = quadTransform
		break
	default:
		return
	}

	genericTransform(img, outImage, x0, y0, x1, y1, transform, data, filter)
}

type TransformsMethod uint32

const (
	AFFINE      TransformsMethod = 0
	EXTENT      TransformsMethod = 1
	PERSPECTIVE TransformsMethod = 2
	QUAD        TransformsMethod = 3
	MESH        TransformsMethod = 4
)

func transformer(box [4]float64, image image.Image, outImage *image.NRGBA, method TransformsMethod, data []float64, filter ResampleFilter, fillcolor color.Color) {
	w := box[2] - box[0]
	h := box[3] - box[1]

	if method == AFFINE {
		data = data[0:6]
	} else if method == EXTENT {
		x0, y0, x1, y1 := data[0], data[1], data[2], data[3]
		xs := (x1 - x0) / w
		ys := (y1 - y0) / h
		method = AFFINE
		data = []float64{xs, 0, x0, 0, ys, y0}
	} else if method == PERSPECTIVE {
		data = data[0:8]
	} else if method == QUAD {
		nw := data[0:2]
		sw := data[2:4]
		se := data[4:6]
		ne := data[6:8]
		x0, y0 := nw[0], nw[1]
		As := 1.0 / w
		At := 1.0 / h
		data =
			[]float64{
				x0,
				(ne[0] - x0) * As,
				(sw[0] - x0) * At,
				(se[0] - sw[0] - ne[0] + x0) * As * At,
				y0,
				(ne[1] - y0) * As,
				(sw[1] - y0) * At,
				(se[1] - sw[1] - ne[1] + y0) * As * At,
			}
	} else {
		panic("unknown transformation method")
	}

	imagingTransform(image, outImage, method, box[0], box[1], box[2], box[3], data, filter)
}

func Transform(dst image.Image, width, height int, method TransformsMethod, data interface{}, filter ResampleFilter, fillcolor color.Color) *image.NRGBA {
	im := image.NewNRGBA(image.Rect(0, 0, width, height))

	if method == MESH {
		if qdata, ok := data.(map[[4]float64][]float64); !ok {
			return nil
		} else {
			for box, quad := range qdata {
				transformer(box, dst, im, QUAD, quad, filter, fillcolor)
			}
		}
	} else {
		transformer([4]float64{0, 0, float64(width), float64(height)}, dst, im, method, data.([]float64), filter, fillcolor)
	}

	return im
}
