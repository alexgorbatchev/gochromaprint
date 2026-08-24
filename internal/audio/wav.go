package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidWAV indicates a malformed or unsupported WAV file.
	ErrInvalidWAV = errors.New("invalid or unsupported WAV audio")
)

// WAVInfo contains metadata parsed from a WAV audio stream.
type WAVInfo struct {
	Channels      int
	SampleRate    int
	BitsPerSample int
	DurationSecs  float64
	Samples       []int16
}

// ReadWAV parses a 16-bit PCM WAV stream from r.
func ReadWAV(r io.Reader) (*WAVInfo, error) {
	var header [12]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("%w: reading RIFF header: %v", ErrInvalidWAV, err)
	}

	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, fmt.Errorf("%w: not a RIFF WAVE format", ErrInvalidWAV)
	}

	var channels, sampleRate, bitsPerSample int
	var samples []int16

	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(r, chunkHeader[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, fmt.Errorf("%w: reading chunk header: %v", ErrInvalidWAV, err)
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, fmtData); err != nil {
				return nil, fmt.Errorf("%w: reading fmt chunk: %v", ErrInvalidWAV, err)
			}
			if len(fmtData) < 16 {
				return nil, fmt.Errorf("%w: fmt chunk too small", ErrInvalidWAV)
			}

			formatTag := binary.LittleEndian.Uint16(fmtData[0:2])
			if formatTag != 1 { // 1 = PCM
				return nil, fmt.Errorf("%w: unsupported format tag %d (only PCM supported)", ErrInvalidWAV, formatTag)
			}

			channels = int(binary.LittleEndian.Uint16(fmtData[2:4]))
			sampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
			bitsPerSample = int(binary.LittleEndian.Uint16(fmtData[14:16]))
			if bitsPerSample != 16 {
				return nil, fmt.Errorf("%w: unsupported bits per sample %d (only 16-bit supported)", ErrInvalidWAV, bitsPerSample)
			}

		case "data":
			dataBytes := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, dataBytes); err != nil {
				return nil, fmt.Errorf("%w: reading data chunk: %v", ErrInvalidWAV, err)
			}

			numSamples := len(dataBytes) / 2
			samples = make([]int16, numSamples)
			for i := 0; i < numSamples; i++ {
				samples[i] = int16(binary.LittleEndian.Uint16(dataBytes[i*2:]))
			}

		default:
			// Skip unknown chunk
			if _, err := io.CopyN(io.Discard, r, int64(chunkSize)); err != nil {
				return nil, fmt.Errorf("%w: skipping chunk %s: %v", ErrInvalidWAV, chunkID, err)
			}
		}
	}

	if channels == 0 || sampleRate == 0 || len(samples) == 0 {
		return nil, fmt.Errorf("%w: missing fmt or data chunks", ErrInvalidWAV)
	}

	duration := float64(len(samples)) / float64(channels*sampleRate)

	return &WAVInfo{
		Channels:      channels,
		SampleRate:    sampleRate,
		BitsPerSample: bitsPerSample,
		DurationSecs:  duration,
		Samples:       samples,
	}, nil
}
