package fft_test

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/alexgorbatchev/gochromaprint/internal/fft"
)

func naiveDFT(in []float64) []float64 {
	n := len(in)
	half := n / 2
	out := make([]float64, half+1)
	for k := 0; k <= half; k++ {
		var sum complex128
		for j := 0; j < n; j++ {
			angle := -2.0 * math.Pi * float64(k*j) / float64(n)
			sum += complex(in[j], 0) * cmplx.Rect(1, angle)
		}
		out[k] = real(sum)*real(sum) + imag(sum)*imag(sum)
	}
	return out
}

func TestFFTAccuracy(t *testing.T) {
	frameSizes := []int{64, 128, 512, 4096}
	for _, size := range frameSizes {
		f := fft.New(size)
		if f.FrameSize() != size {
			t.Fatalf("FrameSize = %d, want %d", f.FrameSize(), size)
		}
		in := make([]float64, size)
		for i := 0; i < size; i++ {
			in[i] = math.Sin(2*math.Pi*float64(i)*5/float64(size)) + 0.5*math.Cos(2*math.Pi*float64(i)*13/float64(size))
		}

		windowed := make([]float64, size)
		win := f.Window()
		for i := 0; i < size; i++ {
			windowed[i] = in[i] * win[i]
		}
		want := naiveDFT(windowed)

		got := make([]float64, size/2+1)
		f.Compute(in, got)

		for k := 0; k <= size/2; k++ {
			diff := math.Abs(got[k] - want[k])
			if diff > 1e-9 {
				t.Fatalf("size %d bin %d: got %g, want %g, diff %g", size, k, got[k], want[k], diff)
			}
		}
	}
}

func TestFFTInvalidSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on non-power-of-2 size")
		}
	}()
	fft.New(100)
}
