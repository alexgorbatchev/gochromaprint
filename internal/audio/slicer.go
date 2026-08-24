package audio

// Slicer splits continuous audio stream into overlapping frames of length `size`
// advancing by `increment` samples each step.
type Slicer struct {
	size      int
	increment int
	buffer    []int16
	callback  func(b1, b2 []int16)
}

// NewSlicer creates an audio slicer with frame size and increment.
func NewSlicer(size, increment int, callback func(b1, b2 []int16)) *Slicer {
	return &Slicer{
		size:      size,
		increment: increment,
		buffer:    make([]int16, 0, size*2),
		callback:  callback,
	}
}

// Reset clears the slicer's internal buffer.
func (s *Slicer) Reset() {
	s.buffer = s.buffer[:0]
}

// Process feeds incoming audio samples through the slicer and triggers callback for each full frame.
func (s *Slicer) Process(input []int16) {
	inIdx := 0
	inLen := len(input)

	// If we have buffered data
	for len(s.buffer) > 0 && len(s.buffer)+(inLen-inIdx) >= s.size {
		needed := s.size - len(s.buffer)
		s.callback(s.buffer, input[inIdx:inIdx+needed])

		if len(s.buffer) >= s.increment {
			copy(s.buffer, s.buffer[s.increment:])
			s.buffer = s.buffer[:len(s.buffer)-s.increment]
		} else {
			inIdx += s.increment - len(s.buffer)
			s.buffer = s.buffer[:0]
		}
	}

	if len(s.buffer) == 0 {
		for inLen-inIdx >= s.size {
			s.callback(input[inIdx:inIdx+s.size], nil)
			inIdx += s.increment
		}
	}

	if inIdx < inLen {
		s.buffer = append(s.buffer, input[inIdx:]...)
	}
}
