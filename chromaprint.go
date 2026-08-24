// Package gochromaprint provides audio fingerprinting for AcoustID in pure Go.
package gochromaprint

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/alexgorbatchev/gochromaprint/internal/audio"
	"github.com/alexgorbatchev/gochromaprint/internal/chroma"
	"github.com/alexgorbatchev/gochromaprint/internal/classifier"
	"github.com/alexgorbatchev/gochromaprint/internal/compress"
	"github.com/alexgorbatchev/gochromaprint/internal/fft"
)

// Version constants for the gochromaprint library.
//
// Versioning Policy:
// Major.Minor tracks the upstream Chromaprint version (e.g. 1.6).
// Patch is the gochromaprint library release version (e.g. 1.6.0).
const (
	UpstreamVersionMajor = 1
	UpstreamVersionMinor = 6
	VersionMajor         = 1
	VersionMinor         = 6
	VersionPatch         = 0
	Version              = "1.6.0"
)

// Algorithm specifies the chromaprint fingerprinting algorithm version.
type Algorithm int

const (
	// AlgorithmTest1 is algorithm version 1 (randomly selected test data).
	AlgorithmTest1 Algorithm = 0
	// AlgorithmTest2 is algorithm version 2 (standard AcoustID algorithm).
	AlgorithmTest2 Algorithm = 1
	// AlgorithmTest3 is algorithm version 3 (interpolated pitch).
	AlgorithmTest3 Algorithm = 2
	// AlgorithmTest4 is algorithm version 4 (leading silence removal).
	AlgorithmTest4 Algorithm = 3
	// AlgorithmTest5 is algorithm version 5 (2x higher resolution).
	AlgorithmTest5 Algorithm = 4
	// AlgorithmDefault is the default algorithm used by AcoustID.
	AlgorithmDefault = AlgorithmTest2
)

const (
	minFrequency = 28
	maxFrequency = 3520
)

var (
	// ErrContextNotStarted indicates the context has not been started with audio stream parameters.
	ErrContextNotStarted = errors.New("chromaprint: context not started")
	// ErrInvalidAlgorithm indicates an unknown algorithm was specified.
	ErrInvalidAlgorithm = errors.New("chromaprint: invalid algorithm")
	// ErrUnknownOption indicates an unknown configuration option name.
	ErrUnknownOption = errors.New("chromaprint: unknown option")
)

// Context generates audio fingerprints from streaming or in-memory PCM audio.
type Context struct {
	algorithm      Algorithm
	config         classifier.Config
	processor      *audio.Processor
	silenceRemover *audio.SilenceRemover
	slicer         *audio.Slicer
	fft            *fft.FFT
	fftOut         []float64
	chroma         *chroma.Extractor
	chromaFilter   *chroma.Filter
	chromaNorm     *chroma.Normalizer
	calculator     *classifier.FingerprintCalculator
	compressor     *compress.Compressor
	started        bool
}

func configForAlgorithm(algo Algorithm) (classifier.Config, error) {
	switch algo {
	case AlgorithmTest1:
		return classifier.NewConfigTest1(), nil
	case AlgorithmTest2:
		return classifier.NewConfigTest2(), nil
	case AlgorithmTest3:
		return classifier.NewConfigTest3(), nil
	case AlgorithmTest4:
		return classifier.NewConfigTest4(), nil
	case AlgorithmTest5:
		return classifier.NewConfigTest5(), nil
	default:
		return classifier.Config{}, fmt.Errorf("%w: %d", ErrInvalidAlgorithm, algo)
	}
}

// New creates and initializes a new chromaprint fingerprinting context.
func New(algo Algorithm) *Context {
	cfg, err := configForAlgorithm(algo)
	if err != nil {
		cfg = classifier.NewConfigTest2()
		algo = AlgorithmDefault
	}

	ctx := &Context{
		algorithm:  algo,
		config:     cfg,
		compressor: compress.NewCompressor(),
	}

	ctx.initPipeline()
	return ctx
}

func (c *Context) initPipeline() {
	// Stage 6: Fingerprint calculator
	c.calculator = classifier.NewFingerprintCalculator(c.config.Classifiers)

	// Stage 5: Chroma normalizer
	c.chromaNorm = chroma.NewNormalizer(c.calculator)

	// Stage 4: Chroma FIR filter
	c.chromaFilter = chroma.NewFilter(c.config.FilterCoefficients, c.chromaNorm)

	// Stage 3: Chroma extractor
	c.chroma = chroma.NewExtractor(minFrequency, maxFrequency, c.config.FrameSize, c.config.SampleRate, c.chromaFilter)
	c.chroma.SetInterpolate(c.config.Interpolate)

	// Stage 2: Real FFT
	c.fft = fft.New(c.config.FrameSize)
	c.fftOut = make([]float64, 1+c.config.FrameSize/2)

	// Slicer connects audio sample frames to FFT
	increment := c.config.FrameSize - c.config.FrameOverlap
	c.slicer = audio.NewSlicer(c.config.FrameSize, increment, func(b1, b2 []int16) {
		c.fft.ComputeFromSamples(b1, b2, c.fftOut)
		c.chroma.Consume(c.fftOut)
	})

	// Stage 1: Audio processor -> (optional SilenceRemover) -> Slicer
	var audioSink audio.Consumer = audio.ConsumerFunc(func(samples []int16) {
		c.slicer.Process(samples)
	})

	if c.config.RemoveSilence {
		c.silenceRemover = audio.NewSilenceRemover(c.config.SilenceThreshold, audioSink)
		audioSink = c.silenceRemover
	}

	c.processor = audio.NewProcessor(c.config.SampleRate, audioSink)
}

// Algorithm returns the configured algorithm.
func (c *Context) Algorithm() Algorithm {
	return c.algorithm
}

// Config returns the active algorithm configuration.
func (c *Context) Config() classifier.Config {
	return c.config
}

// SampleRate returns the internal sample rate (11025 Hz).
func (c *Context) SampleRate() int {
	return c.config.SampleRate
}

// NumChannels returns the internal channel count (1).
func (c *Context) NumChannels() int {
	return 1
}

// ItemDuration returns duration in samples of one raw fingerprint item.
func (c *Context) ItemDuration() int {
	return c.config.ItemDuration()
}

// ItemDurationMs returns duration in milliseconds of one raw fingerprint item.
func (c *Context) ItemDurationMs() int {
	return int(c.config.ItemDurationSeconds() * 1000.0)
}

// Delay returns internal buffer delay in samples.
func (c *Context) Delay() int {
	return c.config.Delay()
}

// DelayMs returns internal buffer delay in milliseconds.
func (c *Context) DelayMs() int {
	return int(c.config.DelaySeconds() * 1000.0)
}

// SetOption sets a configuration option (such as "silence_threshold").
func (c *Context) SetOption(name string, value int) error {
	switch name {
	case "silence_threshold":
		c.config.SilenceThreshold = value
		if c.silenceRemover != nil {
			c.silenceRemover.SetThreshold(value)
		}
		return nil
	default:
		return ErrUnknownOption
	}
}

// Start prepares the context for processing a new audio stream.
func (c *Context) Start(sampleRate, numChannels int) error {
	if err := c.processor.Reset(sampleRate, numChannels); err != nil {
		return err
	}
	if c.silenceRemover != nil {
		c.silenceRemover.Reset(1)
	}
	c.slicer.Reset()
	c.chroma.Reset()
	c.chromaFilter.Reset()
	c.chromaNorm.Reset()
	c.calculator.Reset()
	c.started = true
	return nil
}

// Feed consumes a block of 16-bit signed audio samples in native endianness.
func (c *Context) Feed(samples []int16) error {
	if !c.started {
		return ErrContextNotStarted
	}
	c.processor.Consume(samples)
	return nil
}

// Finish processes all remaining buffered audio samples.
func (c *Context) Finish() error {
	if !c.started {
		return ErrContextNotStarted
	}
	c.processor.Flush()
	return nil
}

// Fingerprint returns the calculated fingerprint as a base64-encoded URL-safe string.
func (c *Context) Fingerprint() (string, error) {
	raw := c.calculator.Fingerprint()
	compressed := c.compressor.Compress(raw, int(c.algorithm))
	return base64.RawURLEncoding.EncodeToString(compressed), nil
}

// RawFingerprint returns the calculated raw 32-bit subfingerprints.
func (c *Context) RawFingerprint() ([]uint32, error) {
	fp := c.calculator.Fingerprint()
	out := make([]uint32, len(fp))
	copy(out, fp)
	return out, nil
}

// RawFingerprintSize returns the number of subfingerprints currently accumulated.
func (c *Context) RawFingerprintSize() int {
	return len(c.calculator.Fingerprint())
}

// FingerprintHash returns a 32-bit locality-sensitive SimHash of the calculated fingerprint.
func (c *Context) FingerprintHash() (uint32, error) {
	return SimHash(c.calculator.Fingerprint()), nil
}

// Clear clears accumulated fingerprint items while allowing continued processing.
func (c *Context) Clear() {
	c.calculator.ClearFingerprint()
}

// EncodeFingerprint compresses and optionally base64-encodes a raw fingerprint.
func EncodeFingerprint(raw []uint32, algorithm Algorithm, base64Encode bool) ([]byte, error) {
	compressed := compress.Compress(raw, int(algorithm))
	if base64Encode {
		encoded := make([]byte, base64.RawURLEncoding.EncodedLen(len(compressed)))
		base64.RawURLEncoding.Encode(encoded, compressed)
		return encoded, nil
	}
	return compressed, nil
}

// DecodeFingerprint decompresses an encoded binary or base64 fingerprint into raw subfingerprints.
func DecodeFingerprint(encoded []byte, base64Decode bool) ([]uint32, Algorithm, error) {
	var payload []byte
	if base64Decode {
		var err error
		payload, err = base64.RawURLEncoding.DecodeString(string(encoded))
		if err != nil {
			return nil, 0, fmt.Errorf("base64 decode error: %w", err)
		}
	} else {
		payload = encoded
	}

	raw, algo, err := compress.Decompress(payload)
	if err != nil {
		return nil, 0, err
	}
	return raw, Algorithm(algo), nil
}

// DecodeFingerprintHeader decodes size and algorithm from an encoded fingerprint header.
func DecodeFingerprintHeader(encoded []byte, base64Decode bool) (size int, algorithm Algorithm, err error) {
	var payload []byte
	if base64Decode {
		limit := 8
		if len(encoded) < limit {
			limit = len(encoded)
		}
		var decErr error
		payload, decErr = base64.RawURLEncoding.DecodeString(string(encoded[:limit]))
		if decErr != nil {
			return 0, 0, fmt.Errorf("base64 decode error: %w", decErr)
		}
	} else {
		payload = encoded
	}

	sz, algo, decErr := compress.DecompressHeader(payload)
	if decErr != nil {
		return 0, 0, decErr
	}
	return sz, Algorithm(algo), nil
}
