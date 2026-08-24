package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alexgorbatchev/godeps"
	"github.com/spf13/cobra"

	"github.com/alexgorbatchev/gochromaprint"
	"github.com/alexgorbatchev/gochromaprint/internal/audio"
)

var version = gochromaprint.Version

var upgradeSelf = func(ctx context.Context, currentVersion string) (string, error) {
	return godeps.UpgradeSelf(ctx, "alexgorbatchev", "gochromaprint", currentVersion)
}

type formatType string

const (
	formatText  formatType = "text"
	formatJSON  formatType = "json"
	formatPlain formatType = "plain"
)

type cliOptions struct {
	format       string
	channels     int
	rate         int
	length       float64
	chunk        float64
	algorithm    int
	overlap      bool
	ts           bool
	raw          bool
	signed       bool
	jsonOutput   bool
	textOutput   bool
	plainOutput  bool
	ignoreErrors bool
	upgrade      bool
}

type jsonResult struct {
	Timestamp   *float64 `json:"timestamp,omitempty"`
	Duration    float64  `json:"duration"`
	Fingerprint any      `json:"fingerprint"`
}

func readAudio(r io.Reader, defaultChannels, defaultRate int, maxLengthSecs float64) ([]int16, int, int, float64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("reading input: %w", err)
	}

	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		wavInfo, err := audio.ReadWAV(bytes.NewReader(data))
		if err != nil {
			return nil, 0, 0, 0, fmt.Errorf("parsing WAV: %w", err)
		}

		samples := wavInfo.Samples
		duration := wavInfo.DurationSecs
		if maxLengthSecs > 0 {
			maxSamples := int(maxLengthSecs * float64(wavInfo.SampleRate*wavInfo.Channels))
			if len(samples) > maxSamples {
				samples = samples[:maxSamples]
				duration = float64(len(samples)) / float64(wavInfo.SampleRate*wavInfo.Channels)
			}
		}
		return samples, wavInfo.Channels, wavInfo.SampleRate, duration, nil
	}

	channels := defaultChannels
	if channels <= 0 {
		channels = 2
	}
	rate := defaultRate
	if rate <= 0 {
		rate = 44100
	}

	numSamples := len(data) / 2
	samples := make([]int16, numSamples)
	for i := 0; i < numSamples; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}

	duration := float64(len(samples)) / float64(channels*rate)
	if maxLengthSecs > 0 {
		maxSamples := int(maxLengthSecs * float64(rate*channels))
		if len(samples) > maxSamples {
			samples = samples[:maxSamples]
			duration = float64(len(samples)) / float64(rate*channels)
		}
	}

	return samples, channels, rate, duration, nil
}

func validateOptions(opts *cliOptions) error {
	if opts.upgrade {
		return nil
	}
	if opts.algorithm < 1 || opts.algorithm > 5 {
		return fmt.Errorf("algorithm must be between 1 and 5 (got %d)", opts.algorithm)
	}
	if opts.format != "" && opts.format != "text" && opts.format != "json" && opts.format != "plain" {
		return fmt.Errorf("unsupported format %q (must be text, json, or plain)", opts.format)
	}
	return nil
}

func printChunkResult(ctx *gochromaprint.Context, opts *cliOptions, first bool, timestamp float64, duration float64, out io.Writer) error {
	rawFp, err := ctx.RawFingerprint()
	if err != nil {
		return fmt.Errorf("getting raw fingerprint: %w", err)
	}

	if len(rawFp) == 0 {
		if first {
			return errors.New("empty fingerprint: not enough audio data")
		}
		return nil
	}

	selectedFormat := formatText
	if opts.jsonOutput || opts.format == "json" {
		selectedFormat = formatJSON
	} else if opts.plainOutput || opts.format == "plain" {
		selectedFormat = formatPlain
	}

	var fpStr string
	if opts.raw {
		var parts []string
		for _, val := range rawFp {
			if opts.signed {
				parts = append(parts, strconv.Itoa(int(int32(val))))
			} else {
				parts = append(parts, strconv.FormatUint(uint64(val), 10))
			}
		}
		fpStr = strings.Join(parts, ",")
	} else {
		fpStr, err = ctx.Fingerprint()
		if err != nil {
			return fmt.Errorf("getting fingerprint: %w", err)
		}
	}

	switch selectedFormat {
	case formatText:
		if !first {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
		if opts.ts {
			if _, err := fmt.Fprintf(out, "TIMESTAMP=%.2f\n", timestamp); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(out, "DURATION=%d\nFINGERPRINT=%s\n", int(duration), fpStr)
		return err

	case formatPlain:
		_, err = fmt.Fprintln(out, fpStr)
		return err

	case formatJSON:
		var fpVal any = fpStr
		if opts.raw {
			if opts.signed {
				signedSlice := make([]int32, len(rawFp))
				for i, v := range rawFp {
					signedSlice[i] = int32(v)
				}
				fpVal = signedSlice
			} else {
				fpVal = rawFp
			}
		}
		var tsPtr *float64
		if opts.chunk > 0 || opts.ts {
			tsPtr = &timestamp
		}
		res := jsonResult{
			Timestamp:   tsPtr,
			Duration:    duration,
			Fingerprint: fpVal,
		}
		enc := json.NewEncoder(out)
		return enc.Encode(res)
	}

	return nil
}

func processFile(filename string, opts *cliOptions, isFirstFile bool, out io.Writer) error {
	if err := validateOptions(opts); err != nil {
		return err
	}

	var r io.Reader
	if filename == "-" || filename == "pipe:0" {
		r = os.Stdin
	} else {
		f, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("opening file %s: %w", filename, err)
		}
		defer func() {
			_ = f.Close()
		}()
		r = f
	}

	samples, channels, rate, fullDuration, err := readAudio(r, opts.channels, opts.rate, opts.length)
	if err != nil {
		return err
	}

	algo := gochromaprint.Algorithm(opts.algorithm - 1)
	ctx := gochromaprint.New(algo)
	if err := ctx.Start(rate, channels); err != nil {
		return fmt.Errorf("initializing fingerprinter: %w", err)
	}

	// Chunked mode
	if opts.chunk > 0 {
		var ts float64
		if opts.ts {
			ts = float64(time.Now().UnixNano()) / 1e9
		}

		chunkLimit := int(opts.chunk * float64(rate))
		extraChunkLimit := 0
		overlapSecs := 0.0
		if opts.overlap {
			extraChunkLimit = ctx.Delay()
			overlapSecs = float64(ctx.DelayMs()) / 1000.0
		}

		firstChunk := true
		offset := 0
		totalAudioFrames := len(samples) / channels

		for offset < totalAudioFrames {
			targetFrames := chunkLimit + extraChunkLimit
			remainingFrames := totalAudioFrames - offset
			takeFrames := remainingFrames
			if takeFrames > targetFrames {
				takeFrames = targetFrames
			}

			startSample := offset * channels
			endSample := (offset + takeFrames) * channels
			if err := ctx.Feed(samples[startSample:endSample]); err != nil {
				return err
			}
			offset += takeFrames

			if takeFrames == targetFrames || offset >= totalAudioFrames {
				if err := ctx.Finish(); err != nil {
					return err
				}

				chunkDuration := float64(takeFrames-extraChunkLimit)/float64(rate) + overlapSecs
				if err := printChunkResult(ctx, opts, firstChunk && isFirstFile, ts, chunkDuration, out); err != nil {
					return err
				}

				if opts.ts {
					ts = float64(time.Now().UnixNano()) / 1e9
				} else {
					ts += chunkDuration
				}

				if opts.overlap {
					ctx.Clear()
					ts -= overlapSecs
				} else {
					if err := ctx.Start(rate, channels); err != nil {
						return err
					}
				}

				if firstChunk {
					extraChunkLimit = 0
					firstChunk = false
				}
			}
		}
		return nil
	}

	// Normal single-result mode
	if err := ctx.Feed(samples); err != nil {
		return fmt.Errorf("processing audio: %w", err)
	}

	if err := ctx.Finish(); err != nil {
		return fmt.Errorf("finishing audio processing: %w", err)
	}

	var ts float64
	if opts.ts {
		ts = float64(time.Now().UnixNano()) / 1e9
	}

	return printChunkResult(ctx, opts, isFirstFile, ts, fullDuration, out)
}

func runUpgrade(ctx context.Context, out io.Writer) error {
	_, _ = fmt.Fprintf(out, "Checking for newer fpcalc release (current version: %s)...\n", version)
	latestVer, err := upgradeSelf(ctx, version)
	if err != nil {
		if strings.Contains(err.Error(), "already at the latest version") {
			_, _ = fmt.Fprintf(out, "fpcalc is already up to date (version %s).\n", version)
			return nil
		}
		return fmt.Errorf("upgrade failed: %w", err)
	}
	_, _ = fmt.Fprintf(out, "Successfully upgraded fpcalc to version %s!\n", latestVer)
	return nil
}

// normalizeArgs rewrites single-dash long flags (e.g. -rate, -channels, -length, -algorithm, -upgrade)
// into standard double-dash flags (--rate, --channels, etc.) for drop-in compatibility with C++ fpcalc.
func normalizeArgs(args []string) []string {
	longFlags := map[string]bool{
		"-rate":          true,
		"-channels":      true,
		"-length":        true,
		"-chunk":         true,
		"-algorithm":     true,
		"-overlap":       true,
		"-ts":            true,
		"-raw":           true,
		"-signed":        true,
		"-json":          true,
		"-text":          true,
		"-plain":         true,
		"-format":        true,
		"-ignore-errors": true,
		"-upgrade":       true,
		"-version":       true,
		"-help":          true,
	}

	res := make([]string, len(args))
	for i, arg := range args {
		if longFlags[arg] {
			res[i] = "-" + arg
		} else {
			res[i] = arg
		}
	}
	return res
}

func main() {
	opts := &cliOptions{}

	rootCmd := &cobra.Command{
		Use:   "fpcalc [OPTIONS] FILE [FILE...]",
		Short: "Generate fingerprints from audio files/streams",
		Args: func(cmd *cobra.Command, args []string) error {
			if opts.upgrade {
				return nil
			}
			if len(args) < 1 {
				return errors.New("no input files")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.upgrade {
				return runUpgrade(cmd.Context(), os.Stdout)
			}

			for i, file := range args {
				err := processFile(file, opts, i == 0, os.Stdout)
				if err != nil {
					if opts.ignoreErrors {
						fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
						continue
					}
					return err
				}
			}
			return nil
		},
	}

	rootCmd.Version = version
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	flags := rootCmd.Flags()
	flags.IntVarP(&opts.channels, "channels", "c", 0, "Number of channels in input audio")
	flags.IntVarP(&opts.rate, "rate", "r", 0, "Sample rate of input audio")
	flags.Float64VarP(&opts.length, "length", "t", 120.0, "Restrict duration in seconds (default 120)")
	flags.Float64Var(&opts.chunk, "chunk", 0, "Split input audio into chunks of this duration")
	flags.IntVarP(&opts.algorithm, "algorithm", "a", 2, "Algorithm method (1-5, default 2)")
	flags.BoolVar(&opts.overlap, "overlap", false, "Overlap chunks slightly to cover edges")
	flags.BoolVar(&opts.ts, "ts", false, "Output UNIX timestamps for chunked results")
	flags.BoolVar(&opts.raw, "raw", false, "Output uncompressed raw fingerprints")
	flags.BoolVar(&opts.signed, "signed", false, "Output uncompressed integers as signed")
	flags.BoolVar(&opts.jsonOutput, "json", false, "Output in JSON format")
	flags.BoolVar(&opts.textOutput, "text", false, "Output in text format")
	flags.BoolVar(&opts.plainOutput, "plain", false, "Output only fingerprint string")
	flags.StringVarP(&opts.format, "format", "f", "", "Output format (text, json, plain)")
	flags.BoolVar(&opts.ignoreErrors, "ignore-errors", false, "Ignore errors on individual files")
	flags.BoolVar(&opts.upgrade, "upgrade", false, "Upgrade fpcalc to the latest released version")

	rootCmd.SetArgs(normalizeArgs(os.Args[1:]))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
