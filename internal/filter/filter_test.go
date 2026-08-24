package filter_test

import (
	"math"
	"testing"

	"github.com/alexgorbatchev/gochromaprint/internal/filter"
)

func TestBoxFilter(t *testing.T) {
	input := []float64{1.0, 2.0, 4.0}

	// Width 1
	out1 := filter.BoxFilter(input, 1)
	if out1[0] != 1.0 || out1[1] != 2.0 || out1[2] != 4.0 {
		t.Fatalf("BoxFilter(1) = %v", out1)
	}

	// Width 2
	out2 := filter.BoxFilter(input, 2)
	if out2[0] != 1.0 || out2[1] != 1.5 || out2[2] != 3.0 {
		t.Fatalf("BoxFilter(2) = %v", out2)
	}

	// Width 3
	out3 := filter.BoxFilter(input, 3)
	if math.Abs(out3[0]-1.333333333) > 1e-6 || math.Abs(out3[1]-2.333333333) > 1e-6 || math.Abs(out3[2]-3.333333333) > 1e-6 {
		t.Fatalf("BoxFilter(3) = %v", out3)
	}
}

func TestGaussianFilter(t *testing.T) {
	input := []float64{1.0, 2.0, 4.0}
	out1 := filter.GaussianFilter(input, 1.6, 3)
	if math.Abs(out1[0]-1.88888889) > 1e-5 || math.Abs(out1[1]-2.33333333) > 1e-5 || math.Abs(out1[2]-2.77777778) > 1e-5 {
		t.Fatalf("GaussianFilter(1.6, 3) = %v", out1)
	}

	out2 := filter.GaussianFilter(input, 3.6, 4)
	if math.Abs(out2[0]-2.3322449) > 1e-5 || math.Abs(out2[1]-2.33306122) > 1e-5 || math.Abs(out2[2]-2.33469388) > 1e-5 {
		t.Fatalf("GaussianFilter(3.6, 4) = %v", out2)
	}
}

func TestGradient(t *testing.T) {
	if filter.Gradient(nil) != nil {
		t.Fatal("expected nil for empty")
	}

	g1 := filter.Gradient([]float64{1.0})
	if len(g1) != 1 || g1[0] != 0.0 {
		t.Fatalf("Gradient(1) = %v", g1)
	}

	g2 := filter.Gradient([]float64{1.0, 2.0})
	if len(g2) != 2 || g2[0] != 1.0 || g2[1] != 1.0 {
		t.Fatalf("Gradient(2) = %v", g2)
	}

	g3 := filter.Gradient([]float64{1.0, 2.0, 4.0})
	if len(g3) != 3 || g3[0] != 1.0 || g3[1] != 1.5 || g3[2] != 2.0 {
		t.Fatalf("Gradient(3) = %v", g3)
	}

	g4 := filter.Gradient([]float64{1.0, 2.0, 4.0, 10.0})
	if len(g4) != 4 || g4[0] != 1.0 || g4[1] != 1.5 || g4[2] != 4.0 || g4[3] != 6.0 {
		t.Fatalf("Gradient(4) = %v", g4)
	}
}
