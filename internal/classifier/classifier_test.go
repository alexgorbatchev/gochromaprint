package classifier_test

import (
	"math"
	"testing"

	"github.com/alexgorbatchev/gochromaprint/internal/classifier"
)

func TestQuantizer(t *testing.T) {
	q := classifier.Quantizer{T0: 0.0, T1: 0.1, T2: 0.3}
	tests := []struct {
		val  float64
		want int
	}{
		{-0.1, 0},
		{0.0, 1},
		{0.03, 1},
		{0.1, 2},
		{0.13, 2},
		{0.3, 3},
		{0.33, 3},
		{1000.0, 3},
	}
	for _, tt := range tests {
		got := q.Quantize(tt.val)
		if got != tt.want {
			t.Fatalf("Quantize(%g) = %d, want %d", tt.val, got, tt.want)
		}
	}
}

func TestFilter0(t *testing.T) {
	data := [][]float64{
		{0.0, 1.0},
		{2.0, 3.0},
	}
	img := classifier.NewRollingIntegralImage(2)
	img.AddRow(data[0])
	img.AddRow(data[1])

	flt1 := classifier.Filter{Type: 0, Y: 0, Height: 1, Width: 1}
	res0 := flt1.Apply(img, 0)
	if math.Abs(res0-0.0) > 1e-6 {
		t.Fatalf("Apply(0) = %g, want 0.0", res0)
	}

	res1 := flt1.Apply(img, 1)
	if math.Abs(res1-1.0986123) > 1e-6 {
		t.Fatalf("Apply(1) = %g, want 1.0986123", res1)
	}
}

func TestFilterUtils(t *testing.T) {
	data := []float64{
		1.0, 2.0, 3.0,
		4.0, 5.0, 6.0,
		7.0, 8.0, 9.0,
	}
	img := classifier.NewRollingIntegralImage(3)
	img.AddRow(data[0:3])
	img.AddRow(data[3:6])
	img.AddRow(data[6:9])

	// Filter0 area tests
	flt0 := classifier.Filter{Type: 0, Y: 0, Height: 1, Width: 1}
	if math.Abs(classifier.Subtract(img.Area(0, 0, 1, 1), 0)-1.0) > 1e-6 {
		t.Fatalf("Filter0(0,0,1,1) failed")
	}
	if math.Abs(classifier.Subtract(img.Area(0, 0, 2, 2), 0)-12.0) > 1e-6 {
		t.Fatalf("Filter0(0,0,2,2) failed")
	}
	if math.Abs(classifier.Subtract(img.Area(0, 0, 3, 3), 0)-45.0) > 1e-6 {
		t.Fatalf("Filter0(0,0,3,3) failed")
	}
	_ = flt0
}
