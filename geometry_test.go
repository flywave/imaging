package imaging

import (
	"image/color"
	"log"
	"math"
	"testing"
)

func TestExtentTransform(t *testing.T) {
	src, err := Open("testdata/flowers.png")
	if err != nil {
		log.Fatalf("failed to open image: %v", err)
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := Transform(src, w, h, EXTENT, []float64{0, 0, math.Round(float64(w) / 2), math.Round(float64(h) / 2)}, Box, true, color.Black)

	Save(dst, "./testdata/transform_extent.jpg")
}

func TestQuadTransform(t *testing.T) {
	src, err := Open("testdata/flowers.png")
	if err != nil {
		log.Fatalf("failed to open image: %v", err)
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := Transform(src, w, h, QUAD, []float64{0, 0, 0, math.Round(float64(h) / 2),
		math.Round(float64(w) / 2), math.Round(float64(h) / 2), math.Round(float64(w) / 2), 0}, Box, true, color.Black)

	Save(dst, "./testdata/transform_quad.jpg")
}

func TestAffineTransform(t *testing.T) {
	src, err := Open("testdata/flowers.png")
	if err != nil {
		log.Fatalf("failed to open image: %v", err)
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := Transform(src, int(math.Round(float64(w)/2)), int(math.Round(float64(h)/2)), AFFINE, []float64{2, 0, 0, 0, 2, 0}, Box, true, color.Black)

	Save(dst, "./testdata/transform_affine.jpg")
}

func TestMeshTransform(t *testing.T) {
	src, err := Open("testdata/flowers.png")
	if err != nil {
		log.Fatalf("failed to open image: %v", err)
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	meshq := make(map[[4]int][]float64)

	meshq[[4]int{0, 0, int(math.Round(float64(w) / 2)), int(math.Round(float64(h) / 2))}] = []float64{0, 0, 0, float64(h),
		float64(w), float64(h), float64(w), 0}
	meshq[[4]int{int(math.Round(float64(w) / 2)), int(math.Round(float64(h) / 2)), w, h}] = []float64{0, 0, 0, float64(h),
		float64(w), float64(h), float64(w), 0}

	dst := Transform(src, w, h, MESH, meshq, Box, true, color.Black)

	Save(dst, "./testdata/transform_mesh.jpg")
}
