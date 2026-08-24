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

func TestFilterTypes(t *testing.T) {
	data := []float64{
		1.0, 2.0, 3.0,
		3.0, 4.0, 5.0,
		6.0, 7.0, 8.0,
	}
	img := classifier.NewRollingIntegralImage(3)
	img.AddRow(data[0:3])
	img.AddRow(data[3:6])
	img.AddRow(data[6:9])

	// Filter0
	f0 := classifier.Filter{Type: 0, Y: 0, Height: 2, Width: 2}
	val0 := f0.Apply(img, 0)
	if math.Abs(val0-math.Log(11.0)) > 1e-6 {
		t.Fatalf("Filter0: got %g", val0)
	}

	// Filter1
	f1 := classifier.Filter{Type: 1, Y: 0, Height: 2, Width: 2}
	val1 := f1.Apply(img, 0)
	// top row (0..1) sum: 1+3=4; bottom row (1..2) sum: 2+4=6
	// SubtractLog(6, 4) = log(7/5)
	if math.Abs(val1-math.Log(7.0/5.0)) > 1e-6 {
		t.Fatalf("Filter1: got %g, want %g", val1, math.Log(7.0/5.0))
	}

	// Filter2
	f2 := classifier.Filter{Type: 2, Y: 0, Height: 2, Width: 2}
	val2 := f2.Apply(img, 0)
	// left col (0..1) sum: 3+4=7; right col (1..2) sum: 1+2=3
	// SubtractLog(7, 3) = log(8/4) = log(2)
	if math.Abs(val2-math.Log(2.0)) > 1e-6 {
		t.Fatalf("Filter2: got %g, want %g", val2, math.Log(2.0))
	}

	// Filter3
	f3 := classifier.Filter{Type: 3, Y: 0, Height: 2, Width: 2}
	val3 := f3.Apply(img, 0)
	// a = bottom-left(3) + top-right(2) = 5
	// b = top-left(1) + bottom-right(4) = 5
	// SubtractLog(5, 5) = 0
	if math.Abs(val3-0.0) > 1e-6 {
		t.Fatalf("Filter3: got %g, want 0", val3)
	}

	// Filter4 (3 rows)
	f4 := classifier.Filter{Type: 4, Y: 0, Height: 3, Width: 3}
	val4 := f4.Apply(img, 0)
	if val4 == 0.0 {
		t.Fatalf("Filter4: unexpected 0.0")
	}

	// Filter5 (3 cols)
	f5 := classifier.Filter{Type: 5, Y: 0, Height: 3, Width: 3}
	val5 := f5.Apply(img, 0)
	if val5 == 0.0 {
		t.Fatalf("Filter5: unexpected 0.0")
	}
}
