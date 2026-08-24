package resample_test

import (
	"math"
	"testing"

	"github.com/alexgorbatchev/gochromaprint/internal/resample"
)

func TestResampler(t *testing.T) {
	rates := []struct {
		inRate  int
		outRate int
	}{
		{44100, 11025},
		{48000, 11025},
		{8000, 11025},
		{22050, 11025},
	}

	for _, tc := range rates {
		r := resample.New(tc.outRate, tc.inRate)
		if r == nil {
			t.Fatalf("failed to create resampler for %d -> %d", tc.inRate, tc.outRate)
		}

		// Feed a 440 Hz sine wave
		srcLen := tc.inRate * 2 // 2 seconds
		src := make([]int16, srcLen)
		for i := 0; i < srcLen; i++ {
			src[i] = int16(16000 * math.Sin(2*math.Pi*440*float64(i)/float64(tc.inRate)))
		}

		dst := make([]int16, tc.outRate*3)
		consumed, produced := r.Resample(dst, src, true)
		if consumed == 0 || produced == 0 {
			t.Fatalf("consumed=%d, produced=%d for %d -> %d", consumed, produced, tc.inRate, tc.outRate)
		}

		expectedProduced := tc.outRate * 2
		diff := math.Abs(float64(produced - expectedProduced))
		if diff > float64(tc.outRate)/10 { // Within 100ms
			t.Fatalf("produced %d samples, expected approx %d", produced, expectedProduced)
		}
	}
}
