package audio_test

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/alexgorbatchev/gochromaprint/internal/audio"
)

type sampleBuffer struct {
	data []int16
}

func (b *sampleBuffer) Consume(samples []int16) {
	b.data = append(b.data, samples...)
}

func loadAudioFile(t *testing.T, filename string) []int16 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", filename))
	if err != nil {
		t.Fatalf("failed to read testdata %s: %v", filename, err)
	}

	samples := make([]int16, len(raw)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(raw[i*2:]))
	}
	return samples
}

func TestMovingAverage(t *testing.T) {
	avg := audio.NewMovingAverage(2)
	if avg.Average() != 0 {
		t.Fatalf("expected 0, got %d", avg.Average())
	}
	avg.AddValue(100)
	if avg.Average() != 100 {
		t.Fatalf("expected 100, got %d", avg.Average())
	}
	avg.AddValue(50)
	if avg.Average() != 75 {
		t.Fatalf("expected 75, got %d", avg.Average())
	}
	avg.AddValue(0)
	if avg.Average() != 25 {
		t.Fatalf("expected 25, got %d", avg.Average())
	}
	avg.AddValue(1000)
	if avg.Average() != 500 {
		t.Fatalf("expected 500, got %d", avg.Average())
	}
}

func TestSilenceRemoverPassThrough(t *testing.T) {
	samples := []int16{1000, 2000, 3000, 4000, 5000, 6000}
	buf := &sampleBuffer{}
	sr := audio.NewSilenceRemover(0, buf)
	if !sr.Reset(1) {
		t.Fatal("Reset failed")
	}
	sr.Consume(samples)
	sr.Flush()

	if !reflect.DeepEqual(buf.data, samples) {
		t.Fatalf("got %v, want %v", buf.data, samples)
	}
}

func TestSilenceRemoverLeadingSilence(t *testing.T) {
	data1 := []int16{0, 60, 0, 1000, 2000, 0, 4000, 5000, 0}
	want := []int16{1000, 2000, 0, 4000, 5000, 0}

	buf := &sampleBuffer{}
	sr := audio.NewSilenceRemover(100, buf)
	if !sr.Reset(1) {
		t.Fatal("Reset failed")
	}
	sr.Consume(data1)
	sr.Flush()

	if !reflect.DeepEqual(buf.data, want) {
		t.Fatalf("got %v, want %v", buf.data, want)
	}
}

func TestAudioProcessorPassThrough(t *testing.T) {
	data := loadAudioFile(t, "test_mono_44100.raw")
	buf := &sampleBuffer{}
	proc := audio.NewProcessor(44100, buf)
	if err := proc.Reset(44100, 1); err != nil {
		t.Fatal(err)
	}
	proc.Consume(data)
	proc.Flush()

	if len(buf.data) != len(data) {
		t.Fatalf("len = %d, want %d", len(buf.data), len(data))
	}
	for i := range data {
		if buf.data[i] != data[i] {
			t.Fatalf("diff at %d: got %d, want %d", i, buf.data[i], data[i])
		}
	}
}

func TestAudioProcessorStereoToMono(t *testing.T) {
	data1 := loadAudioFile(t, "test_stereo_44100.raw")
	data2 := loadAudioFile(t, "test_mono_44100.raw")

	buf := &sampleBuffer{}
	proc := audio.NewProcessor(44100, buf)
	if err := proc.Reset(44100, 2); err != nil {
		t.Fatal(err)
	}
	proc.Consume(data1)
	proc.Flush()

	if len(buf.data) != len(data2) {
		t.Fatalf("len = %d, want %d", len(buf.data), len(data2))
	}
	for i := range data2 {
		if buf.data[i] != data2[i] {
			t.Fatalf("diff at %d: got %d, want %d", i, buf.data[i], data2[i])
		}
	}
}

func TestAudioProcessorResampleMono(t *testing.T) {
	data1 := loadAudioFile(t, "test_mono_44100.raw")
	data2 := loadAudioFile(t, "test_mono_11025.raw")

	buf := &sampleBuffer{}
	proc := audio.NewProcessor(11025, buf)
	if err := proc.Reset(44100, 1); err != nil {
		t.Fatal(err)
	}
	proc.Consume(data1)
	proc.Flush()

	if len(buf.data) != len(data2) {
		t.Fatalf("len = %d, want %d", len(buf.data), len(data2))
	}
	for i := range data2 {
		if buf.data[i] != data2[i] {
			t.Fatalf("diff at %d: got %d, want %d", i, buf.data[i], data2[i])
		}
	}
}

func TestAudioProcessorStereoToMonoAndResample(t *testing.T) {
	data1 := loadAudioFile(t, "test_stereo_44100.raw")
	data2 := loadAudioFile(t, "test_mono_11025.raw")

	buf := &sampleBuffer{}
	proc := audio.NewProcessor(11025, buf)
	if err := proc.Reset(44100, 2); err != nil {
		t.Fatal(err)
	}
	proc.Consume(data1)
	proc.Flush()

	if len(buf.data) != len(data2) {
		t.Fatalf("len = %d, want %d", len(buf.data), len(data2))
	}
	for i := range data2 {
		if buf.data[i] != data2[i] {
			t.Fatalf("diff at %d: got %d, want %d", i, buf.data[i], data2[i])
		}
	}
}

func TestAudioProcessorResampleMonoNonInteger(t *testing.T) {
	data1 := loadAudioFile(t, "test_mono_44100.raw")
	data2 := loadAudioFile(t, "test_mono_8000.raw")

	buf := &sampleBuffer{}
	proc := audio.NewProcessor(8000, buf)
	if err := proc.Reset(44100, 1); err != nil {
		t.Fatal(err)
	}
	proc.Consume(data1)
	proc.Flush()

	if len(buf.data) != len(data2) {
		t.Fatalf("len = %d, want %d", len(buf.data), len(data2))
	}
	for i := range data2 {
		diff := math.Abs(float64(buf.data[i] - data2[i]))
		if diff > 3 {
			t.Fatalf("diff at %d > 3: got %d, want %d", i, buf.data[i], data2[i])
		}
	}
}

func TestAudioProcessorChunkedConsume(t *testing.T) {
	data := loadAudioFile(t, "test_stereo_44100.raw")

	expected := &sampleBuffer{}
	ref := audio.NewProcessor(11025, expected)
	_ = ref.Reset(44100, 2)
	ref.Consume(data)
	ref.Flush()

	chunkSizes := []int{1, 2, 3, 5, 7, 17, 511, 997, 4096}
	for _, chunkSize := range chunkSizes {
		buf := &sampleBuffer{}
		proc := audio.NewProcessor(11025, buf)
		_ = proc.Reset(44100, 2)
		for offset := 0; offset < len(data); offset += chunkSize {
			end := offset + chunkSize
			if end > len(data) {
				end = len(data)
			}
			proc.Consume(data[offset:end])
		}
		proc.Flush()

		if len(buf.data) != len(expected.data) {
			t.Fatalf("chunkSize %d: len = %d, want %d", chunkSize, len(buf.data), len(expected.data))
		}
		for i := range expected.data {
			if buf.data[i] != expected.data[i] {
				t.Fatalf("chunkSize %d diff at %d: got %d, want %d", chunkSize, i, buf.data[i], expected.data[i])
			}
		}
	}
}
