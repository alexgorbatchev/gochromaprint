---
created_on: 2026-10-18 16:32
last_modified: 2026-10-18 16:36
status: current
---

# gochromaprint

The library and bundled `fpcalc` target functional compatibility with the original C++ Chromaprint implementation. Verify behavior against upstream source and native fixtures before claiming parity. The Go API is not a C ABI replacement.

## Checks

- `just build` builds `bin/fpcalc`.
- `just test` runs tests with the race detector. CLI integration tests require FFmpeg and ffprobe on PATH.
- `just lint` checks module hygiene, vet, and golangci-lint.
- `CHROMAPRINT_NATIVE_FPCALC=/absolute/path/to/native/fpcalc go test ./cmd/fpcalc -run TestNativeParity -count=1` compares the CLI with an independently installed native executable.

## Compatibility

- Preserve native input/output contracts when they conflict with general CLI presentation conventions. `-format` specifies the input demuxer; output uses `-json`, `-text`, or `-plain`.
- Preserve the pure Go PCM library. Keep media-container decoding in the CLI.
- FFmpeg decoding must use the upstream compatible SWR resampler settings. Default FFmpeg resampling changes fingerprints.
- Algorithm 3 has zero frame overlap. Upstream leaves pitch interpolation disabled in the fingerprint pipeline despite its configuration flag.
- Native construction with an unknown algorithm preserves its tag but processes with Test1. The silence threshold option only exists for AlgorithmTest4. Header-only Base64 decoding consumes six characters, independent of the body.
- Add behavioral tests whenever runtime behavior changes; require 90% coverage for changed code, excluding scripts. Verify that regression tests fail with the corresponding fix disabled.
- Never publish releases, tags, packages, or deployments without explicit user authorization.
