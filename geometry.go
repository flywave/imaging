package imaging

import (
	"image"
	"image/color"
	"math"
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

func precomputeWeightsForTransform(dstSize, srcSize int, scale float64, filter ResampleFilter) [][]indexWeight {
	du := scale
	if scale < 1.0 {
		scale = 1.0
	}
	ru := math.Ceil(scale * filter.Support)

	out := make([][]indexWeight, dstSize)
	tmp := make([]indexWeight, 0, dstSize*int(ru+2)*2)

	for v := 0; v < dstSize; v++ {
		fu := (float64(v)+0.5)*du - 0.5

		begin := int(math.Ceil(fu - ru))
		if begin < 0 {
			begin = 0
		}
		end := int(math.Floor(fu + ru))
		if end > srcSize-1 {
			end = srcSize - 1
		}

		var sum float64
		for u := begin; u <= end; u++ {
			w := filter.Kernel((float64(u) - fu) / scale)
			if w != 0 {
				sum += w
				tmp = append(tmp, indexWeight{index: u, weight: w})
			}
		}
		if sum != 0 {
			for i := range tmp {
				tmp[i].weight /= sum
			}
		}

		out[v] = tmp
		tmp = tmp[len(tmp):]
	}

	return out
}

func filterApply(img image.Image, outImage []uint8, x, y float64, xw, yw [][]indexWeight, filter ResampleFilter) bool {
	if filter.Support == 0.0 {
		c := img.At(int(x+0.5), int(y+0.5))
		r, g, b, a := c.RGBA()
		outImage[0] = uint8(r * 255 / 65535)
		outImage[1] = uint8(g * 255 / 65535)
		outImage[2] = uint8(b * 255 / 65535)
		outImage[3] = uint8(a * 255 / 65535)
		return true
	}
	xx, yy := int(x+0.5), int(y+0.5)
	var r, g, b, a float64
	for _, w := range xw[xx] {
		s := img.At(w.index/2, int(y+0.5))
		cr, cg, cb, ca := s.RGBA()

		aw := float64(ca) * w.weight
		r += float64(cr) * aw
		g += float64(cg) * aw
		b += float64(cb) * aw
		a += aw
	}

	for _, h := range yw[yy] {
		s := img.At(int(x+0.5), h.index/2)
		cr, cg, cb, ca := s.RGBA()

		aw := float64(ca) * h.weight
		r += float64(cr) * aw
		g += float64(cg) * aw
		b += float64(cb) * aw
		a += aw
	}
	if a != 0 {
		aInv := 1 / a
		outImage[0] = clamp(r * 255 / 65535 * aInv)
		outImage[1] = clamp(g * 255 / 65535 * aInv)
		outImage[2] = clamp(b * 255 / 65535 * aInv)
		outImage[3] = clamp(a * 255 / 65535)
	}

	return true
}

type ImagingTransformMap func(x, y float64, data []float64) (xout, yout float64)

func genericTransform(img image.Image, outImage *image.NRGBA, x0, y0, x1, y1 float64, transform ImagingTransformMap, data []float64, filter ResampleFilter, fill bool, fillColor color.Color) {
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
	var xx, yy float64

	xw, yw := transform(float64(dstW), float64(dstH), data)

	x_ws := precomputeWeightsForTransform(srcW, dstW, float64(dstW)/xw, filter)
	y_ws := precomputeWeightsForTransform(srcH, dstH, float64(dstH)/yw, filter)

	scanLine := make([]uint8, 4)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			xx, yy = transform(float64(x-x0), float64(y-y0), data)
			if !filterApply(img, scanLine, xx, yy, x_ws, y_ws, filter) {
				if fill {
					r, g, b, a := fillColor.RGBA()
					scanLine[0], scanLine[1], scanLine[2], scanLine[3] = uint8(r), uint8(g), uint8(b), uint8(a)
				}
			}
			j := int(y)*outImage.Stride + int(x)*4
			d := dst.Pix[j : j+4 : j+4]
			copy(d, scanLine)
		}
	}
	outImage = dst
}

func imagingTransform(img image.Image, outImage *image.NRGBA, method TransformsMethod, x0, y0, x1, y1 float64, data []float64, filter ResampleFilter, fill bool, fillColor color.Color) {
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

	genericTransform(img, outImage, x0, y0, x1, y1, transform, data, filter, fill, fillColor)
}

type TransformsMethod uint32

const (
	AFFINE      TransformsMethod = 0
	EXTENT      TransformsMethod = 1
	PERSPECTIVE TransformsMethod = 2
	QUAD        TransformsMethod = 3
	MESH        TransformsMethod = 4
)

func transformer(box [4]float64, image image.Image, outImage *image.NRGBA, method TransformsMethod, data []float64, filter ResampleFilter, fill bool, fillColor color.Color) {
	w := box[2] - box[0]
	h := box[3] - box[1]

	if method == AFFINE {
		data = data[0:6]
	} else if method == EXTENT {
		x0, y0, x1, y1 := data[0], data[1], data[2], data[3]
		xs := (x1 - x0) / float64(w)
		ys := (y1 - y0) / float64(h)
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
		As := 1.0 / float64(w)
		At := 1.0 / float64(h)
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

	imagingTransform(image, outImage, method, box[0], box[1], box[2], box[3], data, filter, fill, fillColor)
}

func Transform(dst image.Image, width, height int, method TransformsMethod, data interface{}, filter ResampleFilter, fill bool, fillcolor color.Color) *image.NRGBA {
	im := image.NewNRGBA(image.Rect(0, 0, width, height))

	if method == MESH {
		if qdata, ok := data.(map[[4]float64][]float64); !ok {
			return nil
		} else {
			for box, quad := range qdata {
				transformer(box, dst, im, QUAD, quad, filter, fill, fillcolor)
			}
		}
	} else {
		transformer([4]float64{0, 0, float64(width), float64(height)}, dst, im, method, data.([]float64), filter, fill, fillcolor)
	}

	return im
}
