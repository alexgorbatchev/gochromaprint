package audio_test

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/alexgorbatchev/gochromaprint/internal/audio"
)

func createWAV(channels, sampleRate int, samples []int16) []byte {
	buf := &bytes.Buffer{}
	// RIFF header
	buf.WriteString("RIFF")
	dataSize := uint32(len(samples) * 2)
	fileSize := uint32(36 + dataSize)
	_ = binary.Write(buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16)) // chunk size
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	byteRate := uint32(sampleRate * channels * 2)
	_ = binary.Write(buf, binary.LittleEndian, byteRate)
	blockAlign := uint16(channels * 2)
	_ = binary.Write(buf, binary.LittleEndian, blockAlign)
	_ = binary.Write(buf, binary.LittleEndian, uint16(16)) // 16 bits

	// data chunk
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, dataSize)
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}

	return buf.Bytes()
}

func TestReadWAV(t *testing.T) {
	samples := []int16{100, 200, 300, 400, 500, 600}
	wavBytes := createWAV(2, 44100, samples)

	info, err := audio.ReadWAV(bytes.NewReader(wavBytes))
	if err != nil {
		t.Fatalf("ReadWAV failed: %v", err)
	}

	if info.Channels != 2 {
		t.Fatalf("Channels = %d, want 2", info.Channels)
	}
	if info.SampleRate != 44100 {
		t.Fatalf("SampleRate = %d, want 44100", info.SampleRate)
	}
	if !reflect.DeepEqual(info.Samples, samples) {
		t.Fatalf("samples = %v, want %v", info.Samples, samples)
	}
}
