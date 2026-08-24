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
	if sr.Reset(2) {
		t.Fatal("expected Reset(2) to fail for stereo")
	}
	sr.SetThreshold(10)
	if sr.Threshold() != 10 {
		t.Fatalf("Threshold = %d, want 10", sr.Threshold())
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
	if proc.TargetSampleRate() != 44100 {
		t.Fatalf("TargetSampleRate = %d, want 44100", proc.TargetSampleRate())
	}
	proc.SetTargetSampleRate(11025)
	if proc.TargetSampleRate() != 11025 {
		t.Fatalf("TargetSampleRate = %d, want 11025", proc.TargetSampleRate())
	}
	proc.SetConsumer(buf)

	if err := proc.Reset(44100, 1); err != nil {
		t.Fatal(err)
	}
	if err := proc.Reset(0, 1); err != audio.ErrInvalidSampleRate {
		t.Fatal("expected ErrInvalidSampleRate")
	}
	if err := proc.Reset(44100, 0); err != audio.ErrInvalidChannels {
		t.Fatal("expected ErrInvalidChannels")
	}
	if err := proc.Reset(500, 1); err != audio.ErrInvalidSampleRate {
		t.Fatal("expected ErrInvalidSampleRate")
	}

	_ = proc.Reset(44100, 1)
	proc.SetTargetSampleRate(44100)
	_ = proc.Reset(44100, 1)
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

func TestAudioProcessorMultiChannel(t *testing.T) {
	// 4-channel input
	input := []int16{
		10, 20, 30, 40, // avg = 25
		100, 200, 300, 400, // avg = 250
	}
	buf := &sampleBuffer{}
	proc := audio.NewProcessor(44100, buf)
	if err := proc.Reset(44100, 4); err != nil {
		t.Fatal(err)
	}
	proc.Consume(input)
	proc.Flush()

	expected := []int16{25, 250}
	if !reflect.DeepEqual(buf.data, expected) {
		t.Fatalf("got %v, want %v", buf.data, expected)
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

func TestAudioSlicer(t *testing.T) {
	frames := 0
	slicer := audio.NewSlicer(10, 5, func(b1, b2 []int16) {
		frames++
		if len(b1)+len(b2) != 10 {
			t.Fatalf("frame size = %d, want 10", len(b1)+len(b2))
		}
	})

	input := make([]int16, 25)
	slicer.Process(input)
	if frames != 4 {
		t.Fatalf("expected 4 frames, got %d", frames)
	}

	slicer.Reset()
	frames = 0
	slicer.Process(input[:8])
	if frames != 0 {
		t.Fatalf("expected 0 frames for sub-size input, got %d", frames)
	}
	slicer.Process(input[8:15])
	if frames != 2 {
		t.Fatalf("expected 2 frames after second chunk, got %d", frames)
	}
}
