package classifier

const (
	DefaultSampleRate   = 11025
	DefaultFrameSize    = 4096
	DefaultFrameOverlap = DefaultFrameSize - DefaultFrameSize/3 // 2730
)

var defaultChromaFilterCoefficients = []float64{0.25, 0.75, 1.0, 0.75, 0.25}

var classifiersTest1 = []Classifier{
	{Filter: Filter{Type: 0, Y: 0, Height: 3, Width: 15}, Quantizer: Quantizer{T0: 2.10543, T1: 2.45354, T2: 2.69414}},
	{Filter: Filter{Type: 1, Y: 0, Height: 4, Width: 14}, Quantizer: Quantizer{T0: -0.345922, T1: 0.0463746, T2: 0.446251}},
	{Filter: Filter{Type: 1, Y: 4, Height: 4, Width: 11}, Quantizer: Quantizer{T0: -0.392132, T1: 0.0291077, T2: 0.443391}},
	{Filter: Filter{Type: 3, Y: 0, Height: 4, Width: 14}, Quantizer: Quantizer{T0: -0.192851, T1: 0.00583535, T2: 0.204053}},
	{Filter: Filter{Type: 2, Y: 8, Height: 2, Width: 4}, Quantizer: Quantizer{T0: -0.0771619, T1: -0.00991999, T2: 0.0575406}},
	{Filter: Filter{Type: 5, Y: 6, Height: 2, Width: 15}, Quantizer: Quantizer{T0: -0.710437, T1: -0.518954, T2: -0.330402}},
	{Filter: Filter{Type: 1, Y: 9, Height: 2, Width: 16}, Quantizer: Quantizer{T0: -0.353724, T1: -0.0189719, T2: 0.289768}},
	{Filter: Filter{Type: 3, Y: 4, Height: 2, Width: 10}, Quantizer: Quantizer{T0: -0.128418, T1: -0.0285697, T2: 0.0591791}},
	{Filter: Filter{Type: 3, Y: 9, Height: 2, Width: 16}, Quantizer: Quantizer{T0: -0.139052, T1: -0.0228468, T2: 0.0879723}},
	{Filter: Filter{Type: 2, Y: 1, Height: 3, Width: 6}, Quantizer: Quantizer{T0: -0.133562, T1: 0.00669205, T2: 0.155012}},
	{Filter: Filter{Type: 3, Y: 3, Height: 6, Width: 2}, Quantizer: Quantizer{T0: -0.0267, T1: 0.00804829, T2: 0.0459773}},
	{Filter: Filter{Type: 2, Y: 8, Height: 1, Width: 10}, Quantizer: Quantizer{T0: -0.0972417, T1: 0.0152227, T2: 0.129003}},
	{Filter: Filter{Type: 3, Y: 4, Height: 4, Width: 14}, Quantizer: Quantizer{T0: -0.141434, T1: 0.00374515, T2: 0.149935}},
	{Filter: Filter{Type: 5, Y: 4, Height: 2, Width: 15}, Quantizer: Quantizer{T0: -0.64035, T1: -0.466999, T2: -0.285493}},
	{Filter: Filter{Type: 5, Y: 9, Height: 2, Width: 3}, Quantizer: Quantizer{T0: -0.322792, T1: -0.254258, T2: -0.174278}},
	{Filter: Filter{Type: 2, Y: 1, Height: 8, Width: 4}, Quantizer: Quantizer{T0: -0.0741375, T1: -0.00590933, T2: 0.0600357}},
}

var classifiersTest2 = []Classifier{
	{Filter: Filter{Type: 0, Y: 4, Height: 3, Width: 15}, Quantizer: Quantizer{T0: 1.98215, T1: 2.35817, T2: 2.63523}},
	{Filter: Filter{Type: 4, Y: 4, Height: 6, Width: 15}, Quantizer: Quantizer{T0: -1.03809, T1: -0.651211, T2: -0.282167}},
	{Filter: Filter{Type: 1, Y: 0, Height: 4, Width: 16}, Quantizer: Quantizer{T0: -0.298702, T1: 0.119262, T2: 0.558497}},
	{Filter: Filter{Type: 3, Y: 8, Height: 2, Width: 12}, Quantizer: Quantizer{T0: -0.105439, T1: 0.0153946, T2: 0.135898}},
	{Filter: Filter{Type: 3, Y: 4, Height: 4, Width: 8}, Quantizer: Quantizer{T0: -0.142891, T1: 0.0258736, T2: 0.200632}},
	{Filter: Filter{Type: 4, Y: 0, Height: 3, Width: 5}, Quantizer: Quantizer{T0: -0.826319, T1: -0.590612, T2: -0.368214}},
	{Filter: Filter{Type: 1, Y: 2, Height: 2, Width: 9}, Quantizer: Quantizer{T0: -0.557409, T1: -0.233035, T2: 0.0534525}},
	{Filter: Filter{Type: 2, Y: 7, Height: 3, Width: 4}, Quantizer: Quantizer{T0: -0.0646826, T1: 0.00620476, T2: 0.0784847}},
	{Filter: Filter{Type: 2, Y: 6, Height: 2, Width: 16}, Quantizer: Quantizer{T0: -0.192387, T1: -0.029699, T2: 0.215855}},
	{Filter: Filter{Type: 2, Y: 1, Height: 3, Width: 2}, Quantizer: Quantizer{T0: -0.0397818, T1: -0.00568076, T2: 0.0292026}},
	{Filter: Filter{Type: 5, Y: 10, Height: 1, Width: 15}, Quantizer: Quantizer{T0: -0.53823, T1: -0.369934, T2: -0.190235}},
	{Filter: Filter{Type: 3, Y: 6, Height: 2, Width: 10}, Quantizer: Quantizer{T0: -0.124877, T1: 0.0296483, T2: 0.139239}},
	{Filter: Filter{Type: 2, Y: 1, Height: 1, Width: 14}, Quantizer: Quantizer{T0: -0.101475, T1: 0.0225617, T2: 0.231971}},
	{Filter: Filter{Type: 3, Y: 5, Height: 6, Width: 4}, Quantizer: Quantizer{T0: -0.0799915, T1: -0.00729616, T2: 0.063262}},
	{Filter: Filter{Type: 1, Y: 9, Height: 2, Width: 12}, Quantizer: Quantizer{T0: -0.272556, T1: 0.019424, T2: 0.302559}},
	{Filter: Filter{Type: 3, Y: 4, Height: 2, Width: 14}, Quantizer: Quantizer{T0: -0.164292, T1: -0.0321188, T2: 0.0846339}},
}

// Config represents fingerprinting algorithm settings and coefficients.
type Config struct {
	Classifiers        []Classifier
	FilterCoefficients []float64
	Interpolate        bool
	RemoveSilence      bool
	SilenceThreshold   int
	FrameSize          int
	FrameOverlap       int
	SampleRate         int
	MaxFilterWidth     int
}

// ItemDuration returns duration in samples of one raw fingerprint item.
func (c *Config) ItemDuration() int {
	return c.FrameSize - c.FrameOverlap
}

// ItemDurationSeconds returns duration in seconds of one raw fingerprint item.
func (c *Config) ItemDurationSeconds() float64 {
	return float64(c.ItemDuration()) / float64(c.SampleRate)
}

// Delay returns internal buffer delay in samples.
func (c *Config) Delay() int {
	return ((len(c.FilterCoefficients) - 1) + (c.MaxFilterWidth - 1)) * c.ItemDuration() + c.FrameOverlap
}

// DelaySeconds returns internal buffer delay in seconds.
func (c *Config) DelaySeconds() float64 {
	return float64(c.Delay()) / float64(c.SampleRate)
}

func makeConfig(classifiers []Classifier, filterCoeffs []float64, interpolate, removeSilence bool, silenceThreshold, frameSize, frameOverlap, sampleRate int) Config {
	maxWidth := 0
	for _, cl := range classifiers {
		if cl.Filter.Width > maxWidth {
			maxWidth = cl.Filter.Width
		}
	}
	return Config{
		Classifiers:        classifiers,
		FilterCoefficients: filterCoeffs,
		Interpolate:        interpolate,
		RemoveSilence:      removeSilence,
		SilenceThreshold:   silenceThreshold,
		FrameSize:          frameSize,
		FrameOverlap:       frameOverlap,
		SampleRate:         sampleRate,
		MaxFilterWidth:     maxWidth,
	}
}

// NewConfigTest1 returns configuration for Algorithm Test1.
func NewConfigTest1() Config {
	return makeConfig(classifiersTest1, defaultChromaFilterCoefficients, false, false, 0, DefaultFrameSize, DefaultFrameOverlap, DefaultSampleRate)
}

// NewConfigTest2 returns configuration for Algorithm Test2 (default AcoustID algorithm).
func NewConfigTest2() Config {
	return makeConfig(classifiersTest2, defaultChromaFilterCoefficients, false, false, 0, DefaultFrameSize, DefaultFrameOverlap, DefaultSampleRate)
}

// NewConfigTest3 returns configuration for Algorithm Test3 (interpolated).
func NewConfigTest3() Config {
	return makeConfig(classifiersTest2, defaultChromaFilterCoefficients, true, false, 0, DefaultFrameSize, DefaultFrameOverlap, DefaultSampleRate)
}

// NewConfigTest4 returns configuration for Algorithm Test4 (with silence removal).
func NewConfigTest4() Config {
	return makeConfig(classifiersTest2, defaultChromaFilterCoefficients, false, true, 50, DefaultFrameSize, DefaultFrameOverlap, DefaultSampleRate)
}

// NewConfigTest5 returns configuration for Algorithm Test5 (2x finer resolution).
func NewConfigTest5() Config {
	frameSize := DefaultFrameSize / 2
	frameOverlap := frameSize - frameSize/2 // 2048 - 1024 / 2 -> wait!
	// In C++:
	// set_frame_size(kDefaultFrameSize / 2);
	// set_frame_overlap(kDefaultFrameSize / 2 - kDefaultFrameSize / 4);
	// 4096 / 2 = 2048
	// 4096 / 2 - 4096 / 4 = 2048 - 1024 = 1024 (increment is 2048 - 1024 = 1024)
	return makeConfig(classifiersTest2, defaultChromaFilterCoefficients, false, false, 0, frameSize, frameOverlap, DefaultSampleRate)
}
