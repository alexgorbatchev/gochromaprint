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

func TestReadWAVOddMetadataChunk(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(100))
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint32(44100))
	_ = binary.Write(buf, binary.LittleEndian, uint32(88200))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))

	// Odd-sized metadata chunk (e.g. 5 bytes + 1 pad byte)
	buf.WriteString("INFO")
	_ = binary.Write(buf, binary.LittleEndian, uint32(5))
	buf.WriteString("hello")
	buf.WriteByte(0) // padding byte per RIFF spec

	// data chunk
	buf.WriteString("data")
	samples := []int16{111, 222, 333}
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(samples)*2))
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}

	info, err := audio.ReadWAV(buf)
	if err != nil {
		t.Fatalf("ReadWAV failed on odd chunk: %v", err)
	}
	if !reflect.DeepEqual(info.Samples, samples) {
		t.Fatalf("samples = %v, want %v", info.Samples, samples)
	}
}

func TestReadWAVErrors(t *testing.T) {
	// Not RIFF
	if _, err := audio.ReadWAV(bytes.NewReader([]byte("NOT_A_WAV_FILE"))); err == nil {
		t.Fatal("expected error for non-RIFF")
	}

	// Truncated header
	if _, err := audio.ReadWAV(bytes.NewReader([]byte("RIFF"))); err == nil {
		t.Fatal("expected error for truncated header")
	}

	// Non-PCM format
	bufNonPCM := &bytes.Buffer{}
	bufNonPCM.WriteString("RIFF")
	_ = binary.Write(bufNonPCM, binary.LittleEndian, uint32(50))
	bufNonPCM.WriteString("WAVE")
	bufNonPCM.WriteString("fmt ")
	_ = binary.Write(bufNonPCM, binary.LittleEndian, uint32(16))
	_ = binary.Write(bufNonPCM, binary.LittleEndian, uint16(3)) // IEEE float
	_ = binary.Write(bufNonPCM, binary.LittleEndian, uint16(2))
	_ = binary.Write(bufNonPCM, binary.LittleEndian, uint32(44100))
	_ = binary.Write(bufNonPCM, binary.LittleEndian, uint32(176400))
	_ = binary.Write(bufNonPCM, binary.LittleEndian, uint16(4))
	_ = binary.Write(bufNonPCM, binary.LittleEndian, uint16(16))
	if _, err := audio.ReadWAV(bufNonPCM); err == nil {
		t.Fatal("expected error for non-PCM")
	}

	// Non-16 bit depth
	buf24 := &bytes.Buffer{}
	buf24.WriteString("RIFF")
	_ = binary.Write(buf24, binary.LittleEndian, uint32(50))
	buf24.WriteString("WAVE")
	buf24.WriteString("fmt ")
	_ = binary.Write(buf24, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf24, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf24, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf24, binary.LittleEndian, uint32(44100))
	_ = binary.Write(buf24, binary.LittleEndian, uint32(264600))
	_ = binary.Write(buf24, binary.LittleEndian, uint16(6))
	_ = binary.Write(buf24, binary.LittleEndian, uint16(24)) // 24-bit
	if _, err := audio.ReadWAV(buf24); err == nil {
		t.Fatal("expected error for 24-bit")
	}

	// fmt chunk too small
	bufSmallFmt := &bytes.Buffer{}
	bufSmallFmt.WriteString("RIFF")
	_ = binary.Write(bufSmallFmt, binary.LittleEndian, uint32(30))
	bufSmallFmt.WriteString("WAVE")
	bufSmallFmt.WriteString("fmt ")
	_ = binary.Write(bufSmallFmt, binary.LittleEndian, uint32(8))
	bufSmallFmt.WriteString("12345678")
	if _, err := audio.ReadWAV(bufSmallFmt); err == nil {
		t.Fatal("expected error for small fmt chunk")
	}
}
