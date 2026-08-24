package chroma

// Filter applies a temporal FIR filter across consecutive chroma feature frames.
type Filter struct {
	coefficients []float64
	length       int
	bufferCap    int
	buffer       [][]float64
	result       []float64
	bufferOffset int
	bufferSize   int
	consumer     Consumer
}

// NewFilter creates a new chroma FIR filter.
func NewFilter(coefficients []float64, consumer Consumer) *Filter {
	length := len(coefficients)
	bufCap := 8
	if length > bufCap {
		bufCap = length
	}

	buf := make([][]float64, bufCap)
	for i := range buf {
		buf[i] = make([]float64, NumBands)
	}

	return &Filter{
		coefficients: append([]float64(nil), coefficients...),
		length:       length,
		bufferCap:    bufCap,
		buffer:       buf,
		result:       make([]float64, NumBands),
		bufferSize:   1,
		consumer:     consumer,
	}
}

// Reset resets the filter state.
func (f *Filter) Reset() {
	f.bufferSize = 1
	f.bufferOffset = 0
}

// Consume receives a 12-dimensional chroma feature vector and filters it.
func (f *Filter) Consume(features []float64) {
	copy(f.buffer[f.bufferOffset], features)
	f.bufferOffset = (f.bufferOffset + 1) % f.bufferCap

	if f.bufferSize >= f.length {
		offset := (f.bufferOffset + f.bufferCap - f.length) % f.bufferCap
		for i := range f.result {
			f.result[i] = 0.0
		}
		for i := 0; i < NumBands; i++ {
			for j := 0; j < f.length; j++ {
				f.result[i] += f.buffer[(offset+j)%f.bufferCap][i] * f.coefficients[j]
			}
		}
		if f.consumer != nil {
			f.consumer.Consume(f.result)
		}
	} else {
		f.bufferSize++
	}
}
