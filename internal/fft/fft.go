package fft

import (
	"math"
	"math/bits"
)

// FFT computes the power spectrum of real-valued audio frames using Radix-2 Fast Fourier Transform.
type FFT struct {
	frameSize    int
	window       []float64
	bitRev       []int
	twiddleRe    []float64
	twiddleIm    []float64
	realBuf      []float64
	imagBuf      []float64
}

// New creates a new FFT processor for the given frame size (must be a power of 2).
func New(frameSize int) *FFT {
	if frameSize <= 0 || (frameSize&(frameSize-1)) != 0 {
		panic("frameSize must be a positive power of 2")
	}

	numBits := bits.TrailingZeros(uint(frameSize))
	bitRev := make([]int, frameSize)
	for i := 0; i < frameSize; i++ {
		bitRev[i] = int(bits.Reverse(uint(i)) >> (bits.UintSize - numBits))
	}

	half := frameSize / 2
	twiddleRe := make([]float64, half)
	twiddleIm := make([]float64, half)
	for i := 0; i < half; i++ {
		angle := -2.0 * math.Pi * float64(i) / float64(frameSize)
		twiddleRe[i] = math.Cos(angle)
		twiddleIm[i] = math.Sin(angle)
	}

	// Prepare Hamming window scaled by 1.0 / 32767.0
	window := make([]float64, frameSize)
	const scale = 1.0 / 32767.0
	for i := 0; i < frameSize; i++ {
		window[i] = scale * (0.54 - 0.46*math.Cos(float64(i)*2.0*math.Pi/float64(frameSize-1)))
	}

	return &FFT{
		frameSize: frameSize,
		window:    window,
		bitRev:    bitRev,
		twiddleRe: twiddleRe,
		twiddleIm: twiddleIm,
		realBuf:   make([]float64, frameSize),
		imagBuf:   make([]float64, frameSize),
	}
}

// FrameSize returns the FFT frame size.
func (f *FFT) FrameSize() int {
	return f.frameSize
}

// Window returns the precomputed Hamming window.
func (f *FFT) Window() []float64 {
	return f.window
}

// ComputeFromSamples applies the Hamming window to 16-bit input samples and computes the power spectrum.
// out must have length at least 1 + frameSize/2.
func (f *FFT) ComputeFromSamples(samples1, samples2 []int16, out []float64) {
	n1 := len(samples1)
	for i := 0; i < n1; i++ {
		f.realBuf[i] = float64(samples1[i]) * f.window[i]
		f.imagBuf[i] = 0
	}
	n2 := len(samples2)
	for i := 0; i < n2; i++ {
		f.realBuf[n1+i] = float64(samples2[i]) * f.window[n1+i]
		f.imagBuf[n1+i] = 0
	}
	for i := n1 + n2; i < f.frameSize; i++ {
		f.realBuf[i] = 0
		f.imagBuf[i] = 0
	}

	f.transform(out)
}

// Compute computes the power spectrum from real-valued input slice.
func (f *FFT) Compute(in []float64, out []float64) {
	for i := 0; i < f.frameSize; i++ {
		if i < len(in) {
			f.realBuf[i] = in[i] * f.window[i]
		} else {
			f.realBuf[i] = 0
		}
		f.imagBuf[i] = 0
	}

	f.transform(out)
}

func (f *FFT) transform(out []float64) {
	n := f.frameSize

	// Bit-reversal permutation in-place
	for i := 0; i < n; i++ {
		j := f.bitRev[i]
		if i < j {
			f.realBuf[i], f.realBuf[j] = f.realBuf[j], f.realBuf[i]
			f.imagBuf[i], f.imagBuf[j] = f.imagBuf[j], f.imagBuf[i]
		}
	}

	// Cooley-Tukey Radix-2 decimation-in-time
	for step := 2; step <= n; step <<= 1 {
		halfStep := step >> 1
		twiddleStep := n / step

		for i := 0; i < n; i += step {
			twiddleIdx := 0
			for j := i; j < i+halfStep; j++ {
				tr := f.twiddleRe[twiddleIdx]
				ti := f.twiddleIm[twiddleIdx]
				twiddleIdx += twiddleStep

				k := j + halfStep
				rk := f.realBuf[k]
				ik := f.imagBuf[k]

				tReal := tr*rk - ti*ik
				tImag := tr*ik + ti*rk

				f.realBuf[k] = f.realBuf[j] - tReal
				f.imagBuf[k] = f.imagBuf[j] - tImag
				f.realBuf[j] += tReal
				f.imagBuf[j] += tImag
			}
		}
	}

	// Compute magnitude squared (power spectrum) for k = 0 .. n/2
	half := n / 2
	for i := 0; i <= half; i++ {
		r := f.realBuf[i]
		im := f.imagBuf[i]
		out[i] = r*r + im*im
	}
}
