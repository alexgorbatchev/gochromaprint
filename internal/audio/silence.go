package audio

const defaultSilenceWindow = 55 // 5 ms at 11025 Hz

// MovingAverage tracks the moving average of a signal window.
type MovingAverage struct {
	buffer []int
	size   int
	offset int
	sum    int
	count  int
}

// NewMovingAverage creates a new moving average window of given size.
func NewMovingAverage(size int) *MovingAverage {
	return &MovingAverage{
		buffer: make([]int, size),
		size:   size,
	}
}

// AddValue appends a value to the moving window.
func (m *MovingAverage) AddValue(x int) {
	m.sum += x
	m.sum -= m.buffer[m.offset]
	if m.count < m.size {
		m.count++
	}
	m.buffer[m.offset] = x
	m.offset = (m.offset + 1) % m.size
}

// Average returns the current window average.
func (m *MovingAverage) Average() int {
	if m.count == 0 {
		return 0
	}
	return m.sum / m.count
}

// Reset clears the moving average.
func (m *MovingAverage) Reset() {
	for i := range m.buffer {
		m.buffer[i] = 0
	}
	m.offset = 0
	m.sum = 0
	m.count = 0
}

// Consumer receives a slice of 16-bit audio samples.
type Consumer interface {
	Consume(samples []int16)
}

// ConsumerFunc allows a plain function to act as an audio Consumer.
type ConsumerFunc func(samples []int16)

// Consume calls the underlying function.
func (f ConsumerFunc) Consume(samples []int16) {
	f(samples)
}

// SilenceRemover strips leading silence from audio streams.
type SilenceRemover struct {
	threshold int
	start     bool
	average   *MovingAverage
	consumer  Consumer
}

// NewSilenceRemover creates a new silence remover.
func NewSilenceRemover(threshold int, consumer Consumer) *SilenceRemover {
	return &SilenceRemover{
		threshold: threshold,
		start:     true,
		average:   NewMovingAverage(defaultSilenceWindow),
		consumer:  consumer,
	}
}

// SetThreshold updates the silence detection threshold.
func (sr *SilenceRemover) SetThreshold(threshold int) {
	sr.threshold = threshold
}

// Threshold returns the current silence threshold.
func (sr *SilenceRemover) Threshold() int {
	return sr.threshold
}

// Reset prepares for a new stream.
func (sr *SilenceRemover) Reset(numChannels int) bool {
	if numChannels != 1 {
		return false
	}
	sr.start = true
	sr.average.Reset()
	return true
}

// Consume processes samples, dropping leading silence until energy exceeds threshold.
func (sr *SilenceRemover) Consume(samples []int16) {
	if sr.start {
		for i, s := range samples {
			val := int(s)
			if val < 0 {
				val = -val
			}
			sr.average.AddValue(val)
			if sr.average.Average() > sr.threshold {
				sr.start = false
				samples = samples[i:]
				break
			}
		}
		if sr.start {
			return
		}
	}

	if len(samples) > 0 && sr.consumer != nil {
		sr.consumer.Consume(samples)
	}
}

// Flush performs any end-of-stream cleanup.
func (sr *SilenceRemover) Flush() {}
