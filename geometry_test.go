package imaging_test

import (
	"image/color"
	"log"
	"math"
	"testing"

	"github.com/flywave/imaging"
)

func TestTransform(t *testing.T) {
	src, err := imaging.Open("testdata/flowers.png")
	if err != nil {
		log.Fatalf("failed to open image: %v", err)
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := imaging.Transform(src, w, h, imaging.EXTENT, []float64{0, 0, math.Floor(float64(w) / 2), math.Floor(float64(h) / 2)}, imaging.Box, color.Black)

	imaging.Save(dst, "./fff.jpg")
}
