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

	if img.NumColumns() != 2 {
		t.Fatalf("NumColumns = %d, want 2", img.NumColumns())
	}
	if img.NumRows() != 2 {
		t.Fatalf("NumRows = %d, want 2", img.NumRows())
	}

	flt1 := classifier.Filter{Type: 0, Y: 0, Height: 1, Width: 1}
	res0 := flt1.Apply(img, 0)
	if math.Abs(res0-0.0) > 1e-6 {
		t.Fatalf("Apply(0) = %g, want 0.0", res0)
	}

	res1 := flt1.Apply(img, 1)
	if math.Abs(res1-1.0986123) > 1e-6 {
		t.Fatalf("Apply(1) = %g, want 1.0986123", res1)
	}

	img.Reset()
	if img.NumRows() != 0 {
		t.Fatalf("NumRows after Reset = %d, want 0", img.NumRows())
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

	if classifier.Subtract(5.0, 3.0) != 2.0 {
		t.Fatal("Subtract(5, 3) != 2")
	}

	// Filter0
	f0 := classifier.Filter{Type: 0, Y: 0, Height: 2, Width: 2}
	val0 := f0.Apply(img, 0)
	if math.Abs(val0-math.Log(11.0)) > 1e-6 {
		t.Fatalf("Filter0: got %g", val0)
	}

	// Filter1
	f1 := classifier.Filter{Type: 1, Y: 0, Height: 2, Width: 2}
	val1 := f1.Apply(img, 0)
	if math.Abs(val1-math.Log(7.0/5.0)) > 1e-6 {
		t.Fatalf("Filter1: got %g, want %g", val1, math.Log(7.0/5.0))
	}

	// Filter2
	f2 := classifier.Filter{Type: 2, Y: 0, Height: 2, Width: 2}
	val2 := f2.Apply(img, 0)
	if math.Abs(val2-math.Log(2.0)) > 1e-6 {
		t.Fatalf("Filter2: got %g, want %g", val2, math.Log(2.0))
	}

	// Filter3
	f3 := classifier.Filter{Type: 3, Y: 0, Height: 2, Width: 2}
	val3 := f3.Apply(img, 0)
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

	// Invalid filter type returns 0.0
	fInvalid := classifier.Filter{Type: 99, Y: 0, Height: 1, Width: 1}
	if fInvalid.Apply(img, 0) != 0.0 {
		t.Fatal("expected 0.0 for invalid filter type")
	}
}

func TestImage(t *testing.T) {
	img := classifier.NewImage(3)
	if img.NumRows() != 0 {
		t.Fatalf("NumRows = %d, want 0", img.NumRows())
	}
	img.AddRow([]float64{1, 2, 3})
	img.AddRow([]float64{4, 5, 6})
	if img.NumRows() != 2 {
		t.Fatalf("NumRows = %d, want 2", img.NumRows())
	}
	if img.Row(1)[0] != 4.0 {
		t.Fatalf("Row(1)[0] = %g, want 4.0", img.Row(1)[0])
	}
	img0 := classifier.NewImage(0)
	if img0.NumRows() != 0 {
		t.Fatalf("img0.NumRows() = %d, want 0", img0.NumRows())
	}
}
