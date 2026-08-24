package chroma

import "math"

// Normalizer normalizes 12-dimensional chroma feature vectors using Euclidean L2 norm.
type Normalizer struct {
	consumer Consumer
	buf      []float64
}

// NewNormalizer creates a new Euclidean normalizer.
func NewNormalizer(consumer Consumer) *Normalizer {
	return &Normalizer{
		consumer: consumer,
		buf:      make([]float64, NumBands),
	}
}

// Reset resets the normalizer state.
func (n *Normalizer) Reset() {}

// Consume normalizes the features in-place and passes them to the consumer.
func (n *Normalizer) Consume(features []float64) {
	copy(n.buf, features)
	var sumSquares float64
	for _, v := range n.buf {
		sumSquares += v * v
	}

	norm := 0.0
	if sumSquares > 0 {
		norm = math.Sqrt(sumSquares)
	}

	if norm < 0.01 {
		for i := range n.buf {
			n.buf[i] = 0.0
		}
	} else {
		for i := range n.buf {
			n.buf[i] /= norm
		}
	}

	if n.consumer != nil {
		n.consumer.Consume(n.buf)
	}
}
