# gochromaprint

[![Go Reference](https://pkg.go.dev/badge/github.com/alexgorbatchev/gochromaprint.svg)](https://pkg.go.dev/github.com/alexgorbatchev/gochromaprint)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexgorbatchev/gochromaprint)](https://goreportcard.com/report/github.com/alexgorbatchev/gochromaprint)

A pure Go port of [Chromaprint](https://acoustid.org/chromaprint), the audio fingerprinting library used by [AcoustID](https://acoustid.org).

- **100% Pure Go**: Zero CGO, zero external audio processing dependencies.
- **Bit-for-bit Compatible**: Exact AcoustID / Chromaprint algorithm implementations (Test1–Test5, including default Algorithm 2).
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

`fpcalc` is a **100% drop-in replacement** for the upstream C++ `fpcalc` utility, accepting identical flags and arguments.

#### Pre-built Binaries (GitHub Releases)

Download pre-built binaries for macOS (Darwin), Linux, or Windows (amd64 / arm64) directly from the [GitHub Releases](https://github.com/alexgorbatchev/gochromaprint/releases) page:

```bash
# Using GitHub CLI
gh release download -R alexgorbatchev/gochromaprint --pattern "*darwin_arm64.tar.gz"

# Or install / update via Go toolchain
go install github.com/alexgorbatchev/gochromaprint/cmd/fpcalc@latest
```

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
    fmt.Printf("Match: Pos1=%d, Pos2=%d, Duration=%d, Score=%.2f (PublicScore=%d%%)\n",
        seg.Pos1, seg.Pos2, seg.Duration, seg.Score, seg.PublicScore())
}
```

---

## `fpcalc` CLI

`fpcalc` is a **drop-in replacement** for the official Chromaprint `fpcalc` tool with support for single-dash (`-rate`, `-channels`, `-length`) and double-dash (`--rate`, `--json`) arguments, WAV containers, raw PCM streams, and automatic self-updates:

```bash
# Generate fingerprint from a WAV or raw PCM audio file
fpcalc audio.wav
fpcalc -c 2 -r 44100 audio.raw

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
| `AlgorithmTest3` (2) | Algorithm version 3 (interpolated pitch notes) |
| `AlgorithmTest4` (3) | Algorithm version 4 (automatic leading silence removal) |
| `AlgorithmTest5` (4) | Algorithm version 5 (2x higher temporal resolution) |

---

## Versioning Policy

`gochromaprint` version numbers follow SemVer with upstream synchronization:

- **`Major.Minor`** (`1.6.x`): Directly tracks the upstream [Chromaprint](https://github.com/acoustid/chromaprint) `Major.Minor` release version to ensure algorithm and compatibility parity.
- **`Patch`** (`1.6.0`, `1.6.1`, ...): Represents the `gochromaprint` Go library release version for bug fixes, performance optimizations, and pure Go tooling enhancements built against that upstream version baseline.

---

## Development

Use `just` to run recipes:

```bash
just build    # Build bin/fpcalc
just test     # Run all unit tests with race detector
just lint     # Check hygiene, vet, and linter
```

---

## License

MIT License (compatible with Chromaprint).
