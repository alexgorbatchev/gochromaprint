package chroma

// Resampler averages groups of consecutive chroma vectors by a given integer factor.
type Resampler struct {
	factor    int
	iteration int
	result    []float64
	consumer  Consumer
}

// NewResampler creates a new chroma resampler with downsample factor.
func NewResampler(factor int, consumer Consumer) *Resampler {
	return &Resampler{
		factor:   factor,
		result:   make([]float64, NumBands),
		consumer: consumer,
	}
}

// Reset resets the resampler state.
func (r *Resampler) Reset() {
	r.iteration = 0
	for i := range r.result {
		r.result[i] = 0.0
	}
}

// Consume receives a 12-dimensional chroma feature vector and averages when factor frames are accumulated.
func (r *Resampler) Consume(features []float64) {
	for i := 0; i < NumBands; i++ {
		r.result[i] += features[i]
	}
	r.iteration++
	if r.iteration == r.factor {
		for i := 0; i < NumBands; i++ {
			r.result[i] /= float64(r.factor)
		}
		if r.consumer != nil {
			r.consumer.Consume(r.result)
		}
		r.Reset()
	}
}
