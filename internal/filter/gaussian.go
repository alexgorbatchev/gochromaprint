package filter

import "math"

type reflectIterator struct {
	size    int
	pos     int
	forward bool
}

func newReflectIterator(size int) *reflectIterator {
	return &reflectIterator{
		size:    size,
		pos:     0,
		forward: true,
	}
}

func (it *reflectIterator) MoveForward() {
	if it.forward {
		if it.pos+1 == it.size {
			it.forward = false
		} else {
			it.pos++
		}
	} else {
		if it.pos == 0 {
			it.forward = true
		} else {
			it.pos--
		}
	}
}

func (it *reflectIterator) MoveBack() {
	if it.forward {
		if it.pos == 0 {
			it.forward = false
		} else {
			it.pos--
		}
	} else {
		if it.pos+1 == it.size {
			it.forward = true
		} else {
			it.pos++
		}
	}
}

// BoxFilter applies a moving box filter with reflection at boundaries.
func BoxFilter(input []float64, w int) []float64 {
	size := len(input)
	output := make([]float64, size)
	if w == 0 || size == 0 {
		return output
	}

	wl := w / 2
	wr := w - wl

	it1 := newReflectIterator(size)
	it2 := newReflectIterator(size)
	for i := 0; i < wl; i++ {
		it1.MoveBack()
		it2.MoveBack()
	}

	var sum float64
	for i := 0; i < w; i++ {
		sum += input[it2.pos]
		it2.MoveForward()
	}

	outIdx := 0
	if size > w {
		for i := 0; i < wl; i++ {
			output[outIdx] = sum / float64(w)
			outIdx++
			sum += input[it2.pos] - input[it1.pos]
			it1.MoveForward()
			it2.MoveForward()
		}
		for i := 0; i < size-w-1; i++ {
			output[outIdx] = sum / float64(w)
			outIdx++
			sum += input[it2.pos] - input[it1.pos]
			it1.pos++
			it2.pos++
		}
		for i := 0; i < wr+1; i++ {
			output[outIdx] = sum / float64(w)
			outIdx++
			sum += input[it2.pos] - input[it1.pos]
			it1.MoveForward()
			it2.MoveForward()
		}
	} else {
		for i := 0; i < size; i++ {
			output[outIdx] = sum / float64(w)
			outIdx++
			sum += input[it2.pos] - input[it1.pos]
			it1.MoveForward()
			it2.MoveForward()
		}
	}

	return output
}

// GaussianFilter performs N-pass box filter approximation of a 1D Gaussian blur.
func GaussianFilter(input []float64, sigma float64, n int) []float64 {
	w := int(math.Floor(math.Sqrt(12*sigma*sigma/float64(n) + 1)))
	wl := w
	if w%2 == 0 {
		wl = w - 1
	}
	wu := wl + 2

	m := int(math.Round((12*sigma*sigma - float64(n*wl*wl) - float64(4*n*wl) - float64(3*n)) / float64(-4*wl-4)))

	data := append([]float64(nil), input...)

	for i := 0; i < m; i++ {
		data = BoxFilter(data, wl)
	}
	for i := m; i < n; i++ {
		data = BoxFilter(data, wu)
	}

	return data
}
