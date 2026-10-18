package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/alexgorbatchev/godeps"
	"github.com/spf13/cobra"

	"github.com/alexgorbatchev/gochromaprint"
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

type outputFlag struct {
	opts   *cliOptions
	format formatType
}

func (f outputFlag) String() string { return "false" }
func (f outputFlag) Type() string   { return "bool" }
func (f outputFlag) Set(value string) error {
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	if enabled {
		f.opts.jsonOutput = f.format == formatJSON
		f.opts.plainOutput = f.format == formatPlain
		f.opts.textOutput = f.format == formatText
	}
	return nil
}

func validateOptions(opts *cliOptions) error {
	if opts.upgrade {
		return nil
	}
	if opts.algorithm < 1 || opts.algorithm > 5 {
		return fmt.Errorf("algorithm must be between 1 and 5 (got %d)", opts.algorithm)
	}
	if !validDuration(opts.length) || !validDuration(opts.chunk) || opts.rate < 0 || opts.channels < 0 {
		return errors.New("duration, rate, and channels must not be negative")
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
	if opts.jsonOutput {
		selectedFormat = formatJSON
	} else if opts.plainOutput {
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
		encoded, err := json.Marshal(fpVal)
		if err != nil {
			return err
		}
		if opts.chunk > 0 {
			_, err = fmt.Fprintf(out, "{\"timestamp\": %.2f, \"duration\": %.2f, \"fingerprint\": %s}\n", timestamp, duration, encoded)
		} else {
			_, err = fmt.Fprintf(out, "{\"duration\": %.2f, \"fingerprint\": %s}\n", duration, encoded)
		}
		return err
	}

	return nil
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
	literal := false
	for i, arg := range args {
		if arg == "--" {
			literal = true
		}
		if !literal && longFlags[arg] {
			res[i] = "-" + arg
		} else {
			res[i] = arg
		}
	}
	return res
}

func newCommand() *cobra.Command {
	opts := &cliOptions{}

	rootCmd := &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,
		Use:           "fpcalc [OPTIONS] FILE [FILE...]",
		Short:         "Generate fingerprints from audio files/streams",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("channels") && opts.channels <= 0 {
				return errors.New("input channels must be positive")
			}
			return validateOptions(opts)
		},
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
				return runUpgrade(cmd.Context(), cmd.OutOrStdout())
			}

			for _, file := range args {
				err := processFile(file, opts, true, cmd.OutOrStdout())
				if err != nil {
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
	for _, item := range []struct {
		format formatType
		help   string
	}{{formatJSON, "Output in JSON format"}, {formatText, "Output in text format"}, {formatPlain, "Output only fingerprint string"}} {
		flags.Var(outputFlag{opts, item.format}, string(item.format), item.help)
		flags.Lookup(string(item.format)).NoOptDefVal = "true"
	}
	flags.StringVarP(&opts.format, "format", "f", "", "Input format name (for example s16le)")
	flags.BoolVar(&opts.ignoreErrors, "ignore-errors", false, "Ignore errors on individual files")
	flags.BoolVar(&opts.upgrade, "upgrade", false, "Upgrade fpcalc to the latest released version")

	return rootCmd
}

func main() {
	rootCmd := newCommand()
	rootCmd.SetArgs(normalizeArgs(os.Args[1:]))
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(exitCode(err))
	}
}

func exitCode(err error) int {
	var partial *partialDecodeError
	if errors.As(err, &partial) {
		return 3
	}
	return 2
}
