package chroma

import (
	"math"
)

const (
	NumBands = 12
	baseFreq = 440.0 / 16.0 // 27.5 Hz
)

func freqToOctave(freq float64) float64 {
	return math.Log(freq/baseFreq) / math.Log(2.0)
}

func indexToFreq(i, frameSize, sampleRate int) float64 {
	return float64(i) * float64(sampleRate) / float64(frameSize)
}

func freqToIndex(freq float64, frameSize, sampleRate int) int {
	return int(math.Round(float64(frameSize) * freq / float64(sampleRate)))
}

// Consumer receives a 12-dimensional chroma feature vector.
type Consumer interface {
	Consume(features []float64)
}

// ConsumerFunc allows a function to satisfy Consumer.
type ConsumerFunc func(features []float64)

// Consume calls the underlying function.
func (f ConsumerFunc) Consume(features []float64) {
	f(features)
}

// Extractor computes 12-dimensional chroma features from FFT power spectrum frames.
type Extractor struct {
	interpolate bool
	notes       []byte
	notesFrac   []float64
	minIndex    int
	maxIndex    int
	features    []float64
	consumer    Consumer
}

// NewExtractor creates a new chroma feature extractor.
func NewExtractor(minFreq, maxFreq, frameSize, sampleRate int, consumer Consumer) *Extractor {
	e := &Extractor{
		notes:     make([]byte, frameSize),
		notesFrac: make([]float64, frameSize),
		features:  make([]float64, NumBands),
		consumer:  consumer,
	}
	e.prepareNotes(float64(minFreq), float64(maxFreq), frameSize, sampleRate)
	return e
}

// SetInterpolate enables or disables note pitch interpolation.
func (e *Extractor) SetInterpolate(interpolate bool) {
	e.interpolate = interpolate
}

// Interpolate returns whether interpolation is enabled.
func (e *Extractor) Interpolate() bool {
	return e.interpolate
}

func (e *Extractor) prepareNotes(minFreq, maxFreq float64, frameSize, sampleRate int) {
	minIdx := freqToIndex(minFreq, frameSize, sampleRate)
	if minIdx < 1 {
		minIdx = 1
	}
	maxIdx := freqToIndex(maxFreq, frameSize, sampleRate)
	if maxIdx > frameSize/2 {
		maxIdx = frameSize / 2
	}
	e.minIndex = minIdx
	e.maxIndex = maxIdx

	for i := e.minIndex; i < e.maxIndex; i++ {
		freq := indexToFreq(i, frameSize, sampleRate)
		octave := freqToOctave(freq)
		note := float64(NumBands) * (octave - math.Floor(octave))
		e.notes[i] = byte(note)
		e.notesFrac[i] = note - float64(e.notes[i])
	}
}

// Reset resets the extractor state.
func (e *Extractor) Reset() {
}

// Consume processes an FFT power spectrum frame and emits 12 chroma features to the consumer.
func (e *Extractor) Consume(frame []float64) {
	for i := range e.features {
		e.features[i] = 0.0
	}

	for i := e.minIndex; i < e.maxIndex; i++ {
		note := int(e.notes[i])
		energy := frame[i]
		if e.interpolate {
			note2 := note
			a := 1.0
			if e.notesFrac[i] < 0.5 {
				note2 = (note + NumBands - 1) % NumBands
				a = 0.5 + e.notesFrac[i]
			} else if e.notesFrac[i] > 0.5 {
				note2 = (note + 1) % NumBands
				a = 1.5 - e.notesFrac[i]
			}
			e.features[note] += energy * a
			e.features[note2] += energy * (1.0 - a)
		} else {
			e.features[note] += energy
		}
	}

	if e.consumer != nil {
		e.consumer.Consume(e.features)
	}
}
