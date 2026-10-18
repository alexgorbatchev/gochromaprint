package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func createTestWAV(channels, sampleRate int, durationSecs float64) []byte {
	numSamples := int(float64(channels*sampleRate) * durationSecs)
	samples := make([]int16, numSamples)
	for i := range samples {
		samples[i] = int16(i * 10)
	}

	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	dataSize := uint32(len(samples) * 2)
	fileSize := uint32(36 + dataSize)
	_ = binary.Write(buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate*channels*2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels*2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))

	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, dataSize)
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}

func TestNormalizeArgs(t *testing.T) {
	in := []string{"-rate", "44100", "-channels", "2", "-length", "60", "-json", "-overlap", "-upgrade", "file.wav"}
	want := []string{"--rate", "44100", "--channels", "2", "--length", "60", "--json", "--overlap", "--upgrade", "file.wav"}
	got := normalizeArgs(in)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeArgs = %v, want %v", got, want)
	}
}

func TestCLIProcessFile(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "test_stereo_44100.raw")

	t.Run("TextDefault", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, "DURATION=4\n") || !strings.Contains(out, "FINGERPRINT=AQAAC") {
			t.Fatalf("unexpected text output: %q", out)
		}
	})

	t.Run("Plain", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2, plainOutput: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := strings.TrimSpace(buf.String())
		if !strings.HasPrefix(out, "AQAAC") {
			t.Fatalf("unexpected plain output: %q", out)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2, jsonOutput: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, `"duration": 4.00`) || !strings.Contains(out, `"fingerprint": "AQAAC`) {
			t.Fatalf("unexpected JSON output: %q", out)
		}
	})

	t.Run("RawSigned", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2, raw: true, signed: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, "DURATION=4\nFINGERPRINT=") {
			t.Fatalf("unexpected raw signed output: %q", out)
		}
	})

	t.Run("RawSignedJSON", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2, raw: true, signed: true, jsonOutput: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, `"duration": 4.00`) || !strings.Contains(out, `"fingerprint": [-554577065`) {
			t.Fatalf("unexpected raw signed JSON output: %q", out)
		}
	})

	t.Run("RawUnsignedJSON", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2, raw: true, jsonOutput: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, `"duration": 4.00`) || !strings.Contains(out, `"fingerprint": [3740390231`) {
			t.Fatalf("unexpected raw unsigned JSON output: %q", out)
		}
	})

	t.Run("TimestampOutput", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2, ts: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, "TIMESTAMP=") {
			t.Fatalf("expected TIMESTAMP= in output: %q", out)
		}
	})

	t.Run("SecondFileSpacing", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2}
		if err := processFile(testFile, opts, false, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.HasPrefix(out, "\n") {
			t.Fatalf("expected leading newline for subsequent files: %q", out)
		}
	})

	t.Run("ChunkMode", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, chunk: 3.0, algorithm: 2, textOutput: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, "DURATION=3\n") {
			t.Fatalf("expected chunk duration 3 in output: %q", out)
		}
	})

	t.Run("ChunkModeWithOverlapAndTS", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, chunk: 3.0, overlap: true, ts: true, algorithm: 2, textOutput: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, "TIMESTAMP=") {
			t.Fatalf("expected TIMESTAMP in chunk output: %q", out)
		}
	})

	t.Run("InvalidAlgorithm", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 99}
		if err := processFile(testFile, opts, true, buf); err == nil {
			t.Fatal("expected error for invalid algorithm")
		}
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{channels: 1, rate: 44100, length: 120, algorithm: 2, format: "xml"}
		if err := processFile(testFile, opts, true, buf); err == nil {
			t.Fatal("expected error for invalid format")
		}
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2}
		if err := processFile("nonexistent-file.raw", opts, true, buf); err == nil {
			t.Fatal("expected error for non-existent file")
		}
	})
}

func TestReadAudioWAV(t *testing.T) {
	wavData := createTestWAV(1, 11025, 5.0)
	tmpPath := filepath.Join(t.TempDir(), "test.wav")
	if err := os.WriteFile(tmpPath, wavData, 0644); err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	opts := &cliOptions{length: 4.0, algorithm: 2}
	if err := processFile(tmpPath, opts, true, buf); err != nil {
		t.Fatalf("failed to process WAV file: %v", err)
	}
	if !strings.Contains(buf.String(), "DURATION=5\n") {
		t.Fatalf("expected DURATION=4 in output: %q", buf.String())
	}
}

func TestUpgradeFlag(t *testing.T) {
	origUpgradeSelf := upgradeSelf
	defer func() { upgradeSelf = origUpgradeSelf }()

	t.Run("AlreadyUpToDate", func(t *testing.T) {
		upgradeSelf = func(ctx context.Context, currentVersion string) (string, error) {
			return "", errors.New("already at the latest version")
		}
		buf := &bytes.Buffer{}
		if err := runUpgrade(context.Background(), buf); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if !strings.Contains(buf.String(), "already up to date") {
			t.Fatalf("unexpected output: %q", buf.String())
		}
	})

	t.Run("UpgradeSuccess", func(t *testing.T) {
		upgradeSelf = func(ctx context.Context, currentVersion string) (string, error) {
			return "1.6.1", nil
		}
		buf := &bytes.Buffer{}
		if err := runUpgrade(context.Background(), buf); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if !strings.Contains(buf.String(), "Successfully upgraded fpcalc to version 1.6.1") {
			t.Fatalf("unexpected output: %q", buf.String())
		}
	})

	t.Run("UpgradeFailure", func(t *testing.T) {
		upgradeSelf = func(ctx context.Context, currentVersion string) (string, error) {
			return "", errors.New("network unreachable")
		}
		buf := &bytes.Buffer{}
		if err := runUpgrade(context.Background(), buf); err == nil {
			t.Fatal("expected error on upgrade failure")
		}
	})
}

func TestStdinReading(t *testing.T) {
	testData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "test_stereo_44100.raw"))
	if err != nil {
		t.Fatal(err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write(testData)
	}()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	buf := &bytes.Buffer{}
	opts := &cliOptions{format: "s16le", channels: 1, rate: 44100, length: 120, algorithm: 2}
	if err := processFile("-", opts, true, buf); err != nil {
		t.Fatalf("failed reading stdin: %v", err)
	}
	if !strings.Contains(buf.String(), "FINGERPRINT=") {
		t.Fatalf("unexpected stdin output: %q", buf.String())
	}
}
