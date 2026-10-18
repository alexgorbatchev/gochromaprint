package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexgorbatchev/gochromaprint"
)

func TestNativeParity(t *testing.T) {
	native := os.Getenv("CHROMAPRINT_NATIVE_FPCALC")
	if native == "" {
		t.Skip("set CHROMAPRINT_NATIVE_FPCALC to run upstream parity matrix")
	}
	for algo := 1; algo <= 5; algo++ {
		for _, chunk := range []float64{0, 3} {
			for _, overlap := range []bool{false, true} {
				t.Run(fmt.Sprintf("algorithm%d/chunk%g/overlap%t", algo, chunk, overlap), func(t *testing.T) {
					file := filepath.Join("..", "..", "testdata", "test.mp3")
					args := []string{"-json", "-algorithm", fmt.Sprint(algo), "-chunk", fmt.Sprint(chunk), file}
					if overlap {
						args = append([]string{"-overlap"}, args...)
					}
					want, err := exec.Command(native, args...).Output()
					nativeErr := err
					var got bytes.Buffer
					opts := &cliOptions{algorithm: algo, length: 120, chunk: chunk, overlap: overlap, jsonOutput: true}
					goErr := processFile(file, opts, true, &got)
					if (nativeErr != nil) != (goErr != nil) {
						t.Fatalf("native error %v, Go error %v", nativeErr, goErr)
					}
					if nativeErr != nil {
						return
					}
					decode := func(data []byte) []any {
						t.Helper()
						var values []any
						dec := json.NewDecoder(bytes.NewReader(data))
						for dec.More() {
							var value any
							if err := dec.Decode(&value); err != nil {
								t.Fatal(err)
							}
							values = append(values, value)
						}
						return values
					}
					if fmt.Sprint(decode(got.Bytes())) != fmt.Sprint(decode(want)) {
						t.Fatalf("got %s\nwant %s", got.Bytes(), want)
					}
				})
			}
		}
	}
}

func TestCommandExecution(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
		fail bool
	}{
		{"help", []string{"--help"}, "Input format name", false},
		{"version", []string{"--version"}, version, false},
		{"missing", nil, "", true},
		{"invalid", []string{"--algorithm", "6", "missing"}, "", true},
		{"compressed", []string{"--json", "../../testdata/test.mp3"}, `"fingerprint": "AQ`, false},
		{"output ordering", []string{"--json", "--plain", "--text", "../../testdata/test.mp3"}, "DURATION=10", false},
		{"invalid output bool", []string{"--json=nonsense", "missing"}, "", true},
		{"invalid channels", []string{"--channels=0", "missing"}, "", true},
		{"ignore open errors", []string{"--ignore-errors", "missing"}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newCommand()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if (err != nil) != tc.fail {
				t.Fatalf("error %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output %s", &out)
			}
		})
	}
}

type failedOutput struct{}

func (failedOutput) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestResultWriteFailures(t *testing.T) {
	data := createTestWAV(1, 11025, 5)
	samples := make([]int16, (len(data)-44)/2)
	for i := range samples {
		samples[i] = int16(i * 10)
	}
	fp := gochromaprint.New(gochromaprint.AlgorithmDefault)
	if err := fp.Start(11025, 1); err != nil {
		t.Fatal(err)
	}
	if err := fp.Feed(samples); err != nil {
		t.Fatal(err)
	}
	if err := fp.Finish(); err != nil {
		t.Fatal(err)
	}
	for _, opts := range []cliOptions{{}, {ts: true}, {plainOutput: true}, {jsonOutput: true}} {
		for _, first := range []bool{false, true} {
			err := printChunkResult(fp, &opts, first, 0, 5, failedOutput{})
			if !errors.Is(err, io.ErrClosedPipe) {
				t.Fatalf("write error %v", err)
			}
		}
	}
	err := fingerprintStream(bytes.NewReader(data[44:]), gochromaprint.New(gochromaprint.AlgorithmDefault), &cliOptions{algorithm: 2, chunk: 3}, true, 5, failedOutput{})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("stream write error %v", err)
	}
}

func TestMainVersion(t *testing.T) {
	args := os.Args
	os.Args = []string{"fpcalc", "--version"}
	defer func() { os.Args = args }()
	main()
}

func TestExitCodes(t *testing.T) {
	err := &partialDecodeError{io.ErrUnexpectedEOF}
	if exitCode(fmt.Errorf("input: %w", err)) != 3 || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal("partial decode error lost status or cause")
	}
	if exitCode(io.ErrUnexpectedEOF) != 2 {
		t.Fatal("input failure must exit 2")
	}
}

func TestStreamFailures(t *testing.T) {
	for _, tc := range []struct {
		name  string
		r     io.Reader
		chunk float64
	}{
		{"empty", bytes.NewReader(nil), 0},
		{"odd byte", bytes.NewReader([]byte{1}), 0},
		{"tiny chunk", bytes.NewReader(nil), 0.000001},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := fingerprintStream(tc.r, gochromaprint.New(gochromaprint.AlgorithmDefault), &cliOptions{algorithm: 2, chunk: tc.chunk}, true, 0, io.Discard)
			if err == nil {
				t.Fatal("expected stream failure")
			}
		})
	}
}

func TestDecoderErrorBound(t *testing.T) {
	var b decoderErrors
	p := bytes.Repeat([]byte("error"), maxDecoderErrorBytes)
	for range 2 {
		n, err := b.Write(p)
		if err != nil || n != len(p) {
			t.Fatalf("write %d, %v", n, err)
		}
	}
	if b.Len() != maxDecoderErrorBytes {
		t.Fatalf("unbounded diagnostic bytes: %d", b.Len())
	}
}

func TestNormalizeLiteralFlags(t *testing.T) {
	got := normalizeArgs([]string{"-json", "--", "-rate"})
	if strings.Join(got, " ") != "--json -- -rate" {
		t.Fatal(got)
	}
}

func TestCompressedInputParity(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("FFmpeg integration requires ffmpeg")
	}
	file := filepath.Join("..", "..", "testdata", "test.mp3")
	var out bytes.Buffer
	if err := processFile(file, &cliOptions{algorithm: 2, length: 120, jsonOutput: true}, true, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"fingerprint": "AQ`) {
		t.Fatalf("missing decoded fingerprint: %s", &out)
	}
}

func TestInputFormatParity(t *testing.T) {
	if err := validateOptions(&cliOptions{algorithm: 2, format: "s16le"}); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidDurations(t *testing.T) {
	for _, opts := range []cliOptions{{algorithm: 2, length: -1}, {algorithm: 2, chunk: -1}} {
		if err := validateOptions(&opts); err == nil {
			t.Fatalf("accepted invalid options: %+v", opts)
		}
	}
}
