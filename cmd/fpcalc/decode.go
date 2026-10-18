package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/alexgorbatchev/gochromaprint"
)

const maxDecoderErrorBytes = 16 * 1024

type partialDecodeError struct{ err error }

func (e *partialDecodeError) Error() string { return e.err.Error() }
func (e *partialDecodeError) Unwrap() error { return e.err }

type decoderErrors struct{ bytes.Buffer }

func (b *decoderErrors) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len() < maxDecoderErrorBytes {
		_, err := b.Buffer.Write(p[:min(n, maxDecoderErrorBytes-b.Len())])
		if err != nil {
			return 0, err
		}
	}
	return n, nil
}

func inputArgs(filename string, opts *cliOptions) []string {
	args := []string{}
	if opts.format != "" {
		args = append(args, "-f", opts.format)
	}
	if opts.rate > 0 {
		args = append(args, "-ar", strconv.Itoa(opts.rate))
	}
	if opts.channels > 0 {
		args = append(args, "-ch_layout", strconv.Itoa(opts.channels)+"c")
	}
	return append(args, "-i", filename)
}

// Container duration is independent of the amount fingerprinted, as in fpcalc.
// Pipes have no seekable container duration and report zero.
func inputDuration(filename string, opts *cliOptions) (float64, error) {
	if filename == "pipe:0" {
		return 0, nil
	}
	args := append([]string{"-v", "error"}, inputArgs(filename, opts)...)
	args = append(args, "-show_entries", "stream=duration:format=duration", "-select_streams", "a:0", "-of", "json")
	data, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		return 0, fmt.Errorf("reading audio duration: %w", err)
	}
	var result struct {
		Streams []struct{ Duration string }
		Format  struct{ Duration string }
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, err
	}
	duration := result.Format.Duration
	if len(result.Streams) > 0 && result.Streams[0].Duration != "" {
		duration = result.Streams[0].Duration
	}
	value, err := strconv.ParseFloat(duration, 64)
	if err != nil {
		return 0, nil
	} // FFmpeg omits duration for unknown-length streams.
	return float64(int64(value*1000)) / 1000, nil
}

func processFile(filename string, opts *cliOptions, first bool, out io.Writer) error {
	if err := validateOptions(opts); err != nil {
		return err
	}
	if filename == "-" {
		filename = "pipe:0"
	}
	duration, err := inputDuration(filename, opts)
	if err != nil {
		return err
	}
	fp := gochromaprint.New(gochromaprint.Algorithm(opts.algorithm - 1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	args := append([]string{"-v", "error", "-nostdin"}, inputArgs(filename, opts)...)
	args = append(args, "-vn", "-ac", strconv.Itoa(fp.NumChannels()), "-ar", strconv.Itoa(fp.SampleRate()))
	args = append(args, "-af", "aresample=resampler=swr:filter_size=16:phase_shift=8:linear_interp=1:cutoff=0.8")
	if opts.length > 0 {
		args = append(args, "-t", strconv.FormatFloat(opts.length, 'f', -1, 64))
	}
	args = append(args, "-f", "s16le", "-c:a", "pcm_s16le", "pipe:1")
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if filename == "pipe:0" {
		cmd.Stdin = os.Stdin
	}
	var stderr decoderErrors
	cmd.Stderr = &stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("starting FFmpeg: %w", err)
	}
	readErr := fingerprintStream(pipe, fp, opts, first, duration, out)
	if readErr != nil {
		cancel()
	}
	waitErr := cmd.Wait()
	if readErr != nil {
		return readErr
	}
	if waitErr != nil && !opts.ignoreErrors {
		return &partialDecodeError{fmt.Errorf("decoding audio: %w: %s", waitErr, stderr.String())}
	}
	return nil
}

func fingerprintStream(r io.Reader, fp *gochromaprint.Context, opts *cliOptions, first bool, duration float64, out io.Writer) error {
	rate := fp.SampleRate()
	if err := fp.Start(rate, 1); err != nil {
		return err
	}
	timestamp := 0.0
	if opts.ts {
		timestamp = float64(time.Now().UnixMicro()) / 1e6
	}
	chunkLimit := int(opts.chunk * float64(rate))
	if opts.chunk > 0 && chunkLimit == 0 {
		return errors.New("chunk duration is shorter than one audio sample")
	}
	extra := 0
	overlap := 0.0
	if chunkLimit > 0 && opts.overlap {
		extra = fp.Delay()
		overlap = float64(fp.DelayMs()) / 1000
	}
	chunkSize := 0
	emitted := false
	emit := func() error {
		if err := fp.Finish(); err != nil {
			return err
		}
		chunkDuration := float64(chunkSize-extra)/float64(rate) + overlap
		if chunkLimit == 0 {
			chunkDuration = duration
		}
		if err := printChunkResult(fp, opts, first && !emitted, timestamp, chunkDuration, out); err != nil {
			return err
		}
		emitted = true
		timestamp += chunkDuration
		if opts.ts {
			timestamp = float64(time.Now().UnixMicro()) / 1e6
		}
		if opts.overlap {
			fp.Clear()
			timestamp -= overlap
		} else if err := fp.Start(rate, 1); err != nil {
			return err
		}
		extra = 0
		chunkSize = 0
		return nil
	}
	const blockSamples = 4096
	data := make([]byte, blockSamples*2)
	samples := make([]int16, blockSamples)
	for {
		limit := len(data)
		if chunkLimit > 0 {
			limit = min(limit, (chunkLimit+extra-chunkSize)*2)
		}
		n, err := io.ReadFull(r, data[:limit])
		if n%2 != 0 {
			return errors.New("decoder returned an incomplete audio sample")
		}
		for i := range n / 2 {
			samples[i] = int16(binary.LittleEndian.Uint16(data[2*i:]))
		}
		if n > 0 {
			if feedErr := fp.Feed(samples[:n/2]); feedErr != nil {
				return feedErr
			}
			chunkSize += n / 2
		}
		if chunkLimit > 0 && chunkSize == chunkLimit+extra {
			if emitErr := emit(); emitErr != nil {
				return emitErr
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				return err
			}
			break
		}
	}
	if chunkSize > 0 {
		return emit()
	}
	if !emitted {
		return errors.New("not enough audio data")
	}
	return nil
}

func validDuration(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0) && value < float64(math.MaxInt64/192000)
}
