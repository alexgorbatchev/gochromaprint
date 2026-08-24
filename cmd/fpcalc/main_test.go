package main

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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
		opts := &cliOptions{channels: 1, rate: 44100, length: 120, algorithm: 2}
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
		opts := &cliOptions{channels: 1, rate: 44100, length: 120, algorithm: 2, plainOutput: true}
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
		opts := &cliOptions{channels: 1, rate: 44100, length: 120, algorithm: 2, jsonOutput: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, `"duration":4`) || !strings.Contains(out, `"fingerprint":"AQAAC`) {
			t.Fatalf("unexpected JSON output: %q", out)
		}
	})

	t.Run("RawSigned", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{channels: 1, rate: 44100, length: 120, algorithm: 2, raw: true, signed: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, "DURATION=4\nFINGERPRINT=") {
			t.Fatalf("unexpected raw signed output: %q", out)
		}
	})

	t.Run("ChunkMode", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{channels: 1, rate: 44100, length: 120, chunk: 3.0, algorithm: 2, textOutput: true}
		if err := processFile(testFile, opts, true, buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, "DURATION=3\n") {
			t.Fatalf("expected chunk duration 3 in output: %q", out)
		}
	})

	t.Run("InvalidAlgorithm", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &cliOptions{channels: 1, rate: 44100, length: 120, algorithm: 99}
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
