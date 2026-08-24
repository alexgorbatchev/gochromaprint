package audio

import (
	"errors"

	"github.com/alexgorbatchev/gochromaprint/internal/resample"
)

const (
	minSampleRate = 1000
	maxBufferSize = 1024 * 32
)

var (
	// ErrInvalidChannels indicates invalid channel count.
	ErrInvalidChannels = errors.New("number of audio channels must be positive")
	// ErrInvalidSampleRate indicates invalid sample rate.
	ErrInvalidSampleRate = errors.New("sample rate must be greater than 1000 Hz")
)

// Processor handles audio downmixing, resampling, and buffer management.
type Processor struct {
	targetSampleRate int
	numChannels      int
	buffer           []int16
	bufferOffset     int
	resampleBuffer   []int16
	leftover         []int16
	resampler        *resample.Resampler
	consumer         Consumer
}

// NewProcessor creates an audio processor targeting the given sample rate.
func NewProcessor(targetSampleRate int, consumer Consumer) *Processor {
	return &Processor{
		targetSampleRate: targetSampleRate,
		buffer:           make([]int16, maxBufferSize),
		resampleBuffer:   make([]int16, maxBufferSize),
		consumer:         consumer,
	}
}

// TargetSampleRate returns the configured target sample rate.
func (p *Processor) TargetSampleRate() int {
	return p.targetSampleRate
}

// SetTargetSampleRate updates the target sample rate.
func (p *Processor) SetTargetSampleRate(rate int) {
	p.targetSampleRate = rate
}

// SetConsumer sets the downstream audio consumer.
func (p *Processor) SetConsumer(consumer Consumer) {
	p.consumer = consumer
}

// Reset reinitializes the processor for a new stream with sampleRate and numChannels.
func (p *Processor) Reset(sampleRate, numChannels int) error {
	if numChannels <= 0 {
		return ErrInvalidChannels
	}
	if sampleRate <= minSampleRate {
		return ErrInvalidSampleRate
	}

	p.bufferOffset = 0
	p.leftover = p.leftover[:0]
	p.numChannels = numChannels

	if sampleRate != p.targetSampleRate {
		p.resampler = resample.New(p.targetSampleRate, sampleRate)
	} else {
		p.resampler = nil
	}

	return nil
}

func (p *Processor) loadMono(input []int16, length int) {
	dst := p.buffer[p.bufferOffset : p.bufferOffset+length]
	copy(dst, input[:length])
}

func (p *Processor) loadStereo(input []int16, length int) {
	dst := p.buffer[p.bufferOffset : p.bufferOffset+length]
	srcIdx := 0
	for i := 0; i < length; i++ {
		dst[i] = int16((int32(input[srcIdx]) + int32(input[srcIdx+1])) / 2)
		srcIdx += 2
	}
}

func (p *Processor) loadMultiChannel(input []int16, length int) {
	dst := p.buffer[p.bufferOffset : p.bufferOffset+length]
	srcIdx := 0
	channels := p.numChannels
	for i := 0; i < length; i++ {
		var sum int32
		for c := 0; c < channels; c++ {
			sum += int32(input[srcIdx+c])
		}
		dst[i] = int16(sum / int32(channels))
		srcIdx += channels
	}
}

func (p *Processor) load(input []int16, length int) int {
	available := len(p.buffer) - p.bufferOffset
	if length > available {
		length = available
	}

	switch p.numChannels {
	case 1:
		p.loadMono(input, length)
	case 2:
		p.loadStereo(input, length)
	default:
		p.loadMultiChannel(input, length)
	}

	p.bufferOffset += length
	return length
}

func (p *Processor) resample() {
	if p.resampler != nil {
		consumed, produced := p.resampler.Resample(p.resampleBuffer, p.buffer[:p.bufferOffset], true)
		if produced > maxBufferSize {
			produced = maxBufferSize
		}
		if p.consumer != nil && produced > 0 {
			p.consumer.Consume(p.resampleBuffer[:produced])
		}
		remaining := p.bufferOffset - consumed
		if remaining > 0 {
			copy(p.buffer, p.buffer[consumed:p.bufferOffset])
		} else if remaining < 0 {
			remaining = 0
		}
		p.bufferOffset = remaining
	} else {
		if p.consumer != nil && p.bufferOffset > 0 {
			p.consumer.Consume(p.buffer[:p.bufferOffset])
		}
		p.bufferOffset = 0
	}
}

func (p *Processor) consumeAligned(input []int16) {
	length := len(input) / p.numChannels
	srcIdx := 0

	for length > 0 {
		consumed := p.load(input[srcIdx:], length)
		srcIdx += consumed * p.numChannels
		length -= consumed

		if p.bufferOffset == len(p.buffer) {
			p.resample()
			if p.bufferOffset == len(p.buffer) {
				return
			}
		}
	}
}

// Consume feeds a chunk of multi-channel interleaved audio samples.
func (p *Processor) Consume(input []int16) {
	inLen := len(input)
	srcIdx := 0

	if len(p.leftover) > 0 {
		for inLen > 0 && len(p.leftover) < p.numChannels {
			p.leftover = append(p.leftover, input[srcIdx])
			srcIdx++
			inLen--
		}
		if len(p.leftover) < p.numChannels {
			return
		}
		p.consumeAligned(p.leftover)
		p.leftover = p.leftover[:0]
	}

	remainder := inLen % p.numChannels
	alignedLen := inLen - remainder
	if alignedLen > 0 {
		p.consumeAligned(input[srcIdx : srcIdx+alignedLen])
		srcIdx += alignedLen
	}
	if remainder > 0 {
		p.leftover = append(p.leftover, input[srcIdx:]...)
	}
}

// Flush flushes any remaining buffered samples through the processing pipeline.
func (p *Processor) Flush() {
	if p.bufferOffset > 0 {
		p.resample()
	}
}
