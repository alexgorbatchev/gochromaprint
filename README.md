# gochromaprint

[![Go Reference](https://pkg.go.dev/badge/github.com/alexgorbatchev/gochromaprint.svg)](https://pkg.go.dev/github.com/alexgorbatchev/gochromaprint)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexgorbatchev/gochromaprint)](https://goreportcard.com/report/github.com/alexgorbatchev/gochromaprint)

A pure Go port of [Chromaprint](https://acoustid.org/chromaprint), the audio fingerprinting library used by [AcoustID](https://acoustid.org).

- **Pure Go Library**: No CGO or external decoder is needed when feeding decoded PCM samples.
- **Chromaprint Algorithms**: Test1–Test5, including default Algorithm 2, with upstream fingerprint regression fixtures.
- **Built-in Sinc Resampler & Radix-2 Real FFT**: Native polyphase sinc resampling and high-performance FFT.
- **Fingerprint Matcher & SimHash**: Full support for fingerprint comparison, segment alignment, and locality-sensitive hashing.
- **Includes `fpcalc` CLI**: Drop-in compatible command-line tool.

---

## Installation

### Go Library

```bash
go get github.com/alexgorbatchev/gochromaprint
```

### `fpcalc` CLI

`fpcalc` targets the upstream C++ command's audio processing and output contracts. The Go package exposes Go functions; it does not export the native C ABI.

#### Pre-built Binaries (GitHub Releases)

Download pre-built binaries for macOS (Darwin), Linux, or Windows (amd64 / arm64) directly from the [GitHub Releases](https://github.com/alexgorbatchev/gochromaprint/releases) page:

Install [FFmpeg](https://ffmpeg.org/download.html), including `ffmpeg` and `ffprobe`, on `PATH` for the CLI. The Go library itself consumes PCM and does not launch these tools.

Once installed, `fpcalc` can also self-upgrade to the latest release in-place:

```bash
fpcalc -upgrade
```

---

## Library Usage

### Generating a Fingerprint from Audio

```go
package main

import (
	"fmt"
	"log"

	"github.com/alexgorbatchev/gochromaprint"
)

func main() {
	// Create context with default AcoustID algorithm (Algorithm 2)
	ctx := gochromaprint.New(gochromaprint.AlgorithmDefault)

	sampleRate := 44100
	numChannels := 2

	// Initialize processing pipeline
	if err := ctx.Start(sampleRate, numChannels); err != nil {
		log.Fatalf("failed to start context: %v", err)
	}

	// Feed interleaved 16-bit signed PCM audio samples (in native endianness)
	var samples []int16 = getAudioSamples()
	if err := ctx.Feed(samples); err != nil {
		log.Fatalf("failed to feed samples: %v", err)
	}

	// Finalize buffered audio
	if err := ctx.Finish(); err != nil {
		log.Fatalf("failed to finish: %v", err)
	}

	// Get base64-encoded URL-safe fingerprint string
	fp, err := ctx.Fingerprint()
	if err != nil {
		log.Fatalf("failed to get fingerprint: %v", err)
	}
	fmt.Printf("Fingerprint: %s\n", fp)

	// Get 32-bit locality-sensitive SimHash
	hash, err := ctx.FingerprintHash()
	if err != nil {
		log.Fatalf("failed to get hash: %v", err)
	}
	fmt.Printf("Hash: %d\n", hash)
}
```

### Decoding & Encoding Fingerprints

```go
// Decode a base64-encoded compressed fingerprint to raw 32-bit subfingerprints
raw, algorithm, err := gochromaprint.DecodeFingerprint([]byte(fpString), true)
if err != nil {
    log.Fatal(err)
}

// Compress raw fingerprints back into base64 string
encoded, err := gochromaprint.EncodeFingerprint(raw, algorithm, true)
```

### Comparing & Matching Fingerprints

```go
config := gochromaprint.New(gochromaprint.AlgorithmDefault).Config()
matcher := gochromaprint.NewMatcher(config)

segments, err := matcher.Match(fp1, fp2)
if err != nil {
    log.Fatal(err)
}

for _, seg := range segments {
    fmt.Printf("Match: Pos1=%d, Pos2=%d, Duration=%d, Score=%.2f (PublicScore=%d)\n",
        seg.Pos1, seg.Pos2, seg.Duration, seg.Score, seg.PublicScore())
}
```

---

## `fpcalc` CLI

`fpcalc` accepts single-dash (`-rate`, `-channels`, `-length`) and double-dash (`--rate`, `--json`) arguments. FFmpeg decodes supported containers and compressed formats as a stream; raw PCM requires an explicit input format. No audio files are rewritten. The default fingerprint limit is 120 seconds; `-length 0` processes the whole stream. Reported non-chunked duration describes the full input, independently of that limit. Unknown duration on stdin is reported as zero.

```bash
# Generate fingerprint from a WAV or raw PCM audio file
fpcalc audio.wav
fpcalc -format s16le -c 2 -r 44100 audio.raw

# Output JSON
fpcalc --json audio.wav

# Output raw uncompressed subfingerprints
fpcalc --raw audio.wav

# Output only the fingerprint string
fpcalc --plain audio.wav

# Self-upgrade to the latest GitHub release
fpcalc -upgrade
```

---

## Algorithms

| Algorithm | Description |
|-----------|-------------|
| `AlgorithmTest1` (0) | Algorithm version 1 (randomly selected test data) |
| `AlgorithmTest2` (1) | Algorithm version 2 (**Default AcoustID Algorithm**) |
| `AlgorithmTest3` (2) | Experimental non-overlapping frames; interpolation remains disabled to match upstream |
| `AlgorithmTest4` (3) | Algorithm version 4 (automatic leading silence removal) |
| `AlgorithmTest5` (4) | Algorithm version 5 (2x higher temporal resolution) |

---

## Versioning Policy

`gochromaprint` version numbers follow SemVer with upstream synchronization:

- **`Major.Minor`** (`1.6.x`): Directly tracks the upstream [Chromaprint](https://github.com/acoustid/chromaprint) `Major.Minor` release version to ensure algorithm and compatibility parity.
- **`Patch`** (`1.6.0`, `1.6.1`, ...): Represents the `gochromaprint` Go library release version for bug fixes, performance optimizations, and pure Go tooling enhancements built against that upstream version baseline.

---

## Matching Scores

`Segment.Score` measures average differing bits per subfingerprint (0–32); lower is closer. `PublicScore()` scales that error score by 100 and is not a similarity percentage. Like upstream, the matcher may combine similar-scoring segments across a rejected interval; a returned segment's duration does not establish uninterrupted agreement across its extent.

---

## License

MIT License (compatible with Chromaprint).
