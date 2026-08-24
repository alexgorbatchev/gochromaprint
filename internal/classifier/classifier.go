package classifier

import (
	"math"
)

// SubtractLog compares two values using log((1+a)/(1+b)).
func SubtractLog(a, b float64) float64 {
	return math.Log((1.0 + a) / (1.0 + b))
}

// Subtract computes a - b.
func Subtract(a, b float64) float64 {
	return a - b
}

// Filter represents a Haar-like 2D rectangular filter applied to an integral image.
type Filter struct {
	Type   int
	Y      int
	Height int
	Width  int
}

// Apply computes the filter response at horizontal offset x.
func (f Filter) Apply(img *RollingIntegralImage, x int) float64 {
	switch f.Type {
	case 0:
		a := img.Area(x, f.Y, x+f.Width, f.Y+f.Height)
		return SubtractLog(a, 0)
	case 1:
		h2 := f.Height / 2
		a := img.Area(x, f.Y+h2, x+f.Width, f.Y+f.Height)
		b := img.Area(x, f.Y, x+f.Width, f.Y+h2)
		return SubtractLog(a, b)
	case 2:
		w2 := f.Width / 2
		a := img.Area(x+w2, f.Y, x+f.Width, f.Y+f.Height)
		b := img.Area(x, f.Y, x+w2, f.Y+f.Height)
		return SubtractLog(a, b)
	case 3:
		w2 := f.Width / 2
		h2 := f.Height / 2
		a := img.Area(x, f.Y+h2, x+w2, f.Y+f.Height) + img.Area(x+w2, f.Y, x+f.Width, f.Y+h2)
		b := img.Area(x, f.Y, x+w2, f.Y+h2) + img.Area(x+w2, f.Y+h2, x+f.Width, f.Y+f.Height)
		return SubtractLog(a, b)
	case 4:
		h3 := f.Height / 3
		a := img.Area(x, f.Y+h3, x+f.Width, f.Y+2*h3)
		b := img.Area(x, f.Y, x+f.Width, f.Y+h3) + img.Area(x, f.Y+2*h3, x+f.Width, f.Y+f.Height)
		return SubtractLog(a, b)
	case 5:
		w3 := f.Width / 3
		a := img.Area(x+w3, f.Y, x+2*w3, f.Y+f.Height)
		b := img.Area(x, f.Y, x+w3, f.Y+f.Height) + img.Area(x+2*w3, f.Y, x+f.Width, f.Y+f.Height)
		return SubtractLog(a, b)
	}
	return 0.0
}

// Quantizer quantizes a filter response into 4 discrete buckets (0, 1, 2, 3).
type Quantizer struct {
	T0, T1, T2 float64
}

// Quantize returns the quantized bucket (0, 1, 2, 3) for value.
func (q Quantizer) Quantize(value float64) int {
	if value < q.T1 {
		if value < q.T0 {
			return 0
		}
		return 1
	}
	if value < q.T2 {
		return 2
	}
	return 3
}

// Classifier combines a filter and a quantizer.
type Classifier struct {
	Filter    Filter
	Quantizer Quantizer
}

// Classify classifies an image patch at offset x into 0, 1, 2, or 3.
func (c Classifier) Classify(img *RollingIntegralImage, x int) int {
	val := c.Filter.Apply(img, x)
	return c.Quantizer.Quantize(val)
}
