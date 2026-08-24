package chroma

// Filter applies a temporal FIR filter across consecutive chroma feature frames.
type Filter struct {
	coefficients []float64
	length       int
	buffer       [8][]float64
	result       []float64
	bufferOffset int
	bufferSize   int
	consumer     Consumer
}

// NewFilter creates a new chroma FIR filter.
func NewFilter(coefficients []float64, consumer Consumer) *Filter {
	f := &Filter{
		coefficients: append([]float64(nil), coefficients...),
		length:       len(coefficients),
		result:       make([]float64, NumBands),
		bufferSize:   1,
		consumer:     consumer,
	}
	for i := range f.buffer {
		f.buffer[i] = make([]float64, NumBands)
	}
	return f
}

// Reset resets the filter state.
func (f *Filter) Reset() {
	f.bufferSize = 1
	f.bufferOffset = 0
}

// Consume receives a 12-dimensional chroma feature vector and filters it.
func (f *Filter) Consume(features []float64) {
	copy(f.buffer[f.bufferOffset], features)
	f.bufferOffset = (f.bufferOffset + 1) % 8

	if f.bufferSize >= f.length {
		offset := (f.bufferOffset + 8 - f.length) % 8
		for i := range f.result {
			f.result[i] = 0.0
		}
		for i := 0; i < NumBands; i++ {
			for j := 0; j < f.length; j++ {
				f.result[i] += f.buffer[(offset+j)%8][i] * f.coefficients[j]
			}
		}
		if f.consumer != nil {
			f.consumer.Consume(f.result)
		}
	} else {
		f.bufferSize++
	}
}
