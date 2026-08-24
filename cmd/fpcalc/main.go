package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexgorbatchev/gochromaprint"
	"github.com/alexgorbatchev/gochromaprint/internal/audio"
)

var version = gochromaprint.Version

type formatType string

const (
	formatText  formatType = "text"
	formatJSON  formatType = "json"
	formatPlain formatType = "plain"
)

type cliOptions struct {
	format      string
	channels    int
	rate        int
	length      float64
	algorithm   int
	raw         bool
	signed      bool
	jsonOutput  bool
	textOutput  bool
	plainOutput bool
}

type jsonResult struct {
	Duration    float64 `json:"duration"`
	Fingerprint any     `json:"fingerprint"`
}

func readAudio(r io.Reader, defaultChannels, defaultRate int, maxLengthSecs float64) ([]int16, int, int, float64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("reading input: %w", err)
	}

	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		wavInfo, err := audio.ReadWAV(bytes.NewReader(data))
		if err == nil {
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

func processFile(filename string, opts *cliOptions, out io.Writer) error {
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

	samples, channels, rate, duration, err := readAudio(r, opts.channels, opts.rate, opts.length)
	if err != nil {
		return err
	}

	algo := gochromaprint.Algorithm(opts.algorithm - 1)
	if opts.algorithm < 1 || opts.algorithm > 5 {
		algo = gochromaprint.AlgorithmDefault
	}

	ctx := gochromaprint.New(algo)
	if err := ctx.Start(rate, channels); err != nil {
		return fmt.Errorf("initializing fingerprinter: %w", err)
	}

	if err := ctx.Feed(samples); err != nil {
		return fmt.Errorf("processing audio: %w", err)
	}

	if err := ctx.Finish(); err != nil {
		return fmt.Errorf("finishing audio processing: %w", err)
	}

	rawFp, err := ctx.RawFingerprint()
	if err != nil {
		return fmt.Errorf("getting raw fingerprint: %w", err)
	}

	if len(rawFp) == 0 {
		return errors.New("empty fingerprint: not enough audio data")
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
		res := jsonResult{
			Duration:    duration,
			Fingerprint: fpVal,
		}
		enc := json.NewEncoder(out)
		if err := enc.Encode(res); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	opts := &cliOptions{}

	rootCmd := &cobra.Command{
		Use:   "fpcalc [OPTIONS] FILE [FILE...]",
		Short: "Generate fingerprints from audio files/streams",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, file := range args {
				if err := processFile(file, opts, os.Stdout); err != nil {
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
	flags.IntVarP(&opts.algorithm, "algorithm", "a", 2, "Algorithm method (1-5, default 2)")
	flags.BoolVar(&opts.raw, "raw", false, "Output uncompressed raw fingerprints")
	flags.BoolVar(&opts.signed, "signed", false, "Output uncompressed integers as signed")
	flags.BoolVar(&opts.jsonOutput, "json", false, "Output in JSON format")
	flags.BoolVar(&opts.textOutput, "text", false, "Output in text format")
	flags.BoolVar(&opts.plainOutput, "plain", false, "Output only fingerprint string")
	flags.StringVarP(&opts.format, "format", "f", "", "Output format (text, json, plain)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
