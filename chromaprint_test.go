package gochromaprint_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/alexgorbatchev/gochromaprint"
	"github.com/alexgorbatchev/gochromaprint/internal/audio"
	"github.com/alexgorbatchev/gochromaprint/internal/chroma"
	"github.com/alexgorbatchev/gochromaprint/internal/classifier"
	"github.com/alexgorbatchev/gochromaprint/internal/fft"
)

func loadAudioFile(t *testing.T, filename string) []int16 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to read testdata %s: %v", filename, err)
	}

	samples := make([]int16, len(raw)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(raw[i*2:]))
	}
	return samples
}

func TestBasicImage(t *testing.T) {
	data := loadAudioFile(t, "test_stereo_44100.raw")

	image := classifier.NewImage(12)
	normalizer := chroma.NewNormalizer(chroma.ConsumerFunc(func(features []float64) {
		image.AddRow(features)
	}))
	chromaExt := chroma.NewExtractor(28, 3520, 4096, 11025, normalizer)
	fftCalc := fft.New(4096)
	fftOut := make([]float64, 2049)

	slicer := audio.NewSlicer(4096, 4096-2730, func(b1, b2 []int16) {
		fftCalc.ComputeFromSamples(b1, b2, fftOut)
		chromaExt.Consume(fftOut)
	})

	proc := audio.NewProcessor(11025, audio.ConsumerFunc(func(samples []int16) {
		slicer.Process(samples)
	}))

	if err := proc.Reset(44100, 2); err != nil {
		t.Fatal(err)
	}
	proc.Consume(data)
	proc.Flush()

	expected := [][]float64{
		{0.155444, 0.268618, 0.474445, 0.159887, 0.1761, 0.423511, 0.178933, 0.34433, 0.360958, 0.30421, 0.200217, 0.17072},
		{0.159809, 0.238675, 0.286526, 0.166119, 0.225144, 0.449236, 0.162444, 0.371875, 0.259626, 0.483961, 0.24491, 0.17034},
		{0.156518, 0.271503, 0.256073, 0.152689, 0.174664, 0.52585, 0.141517, 0.253695, 0.293199, 0.332114, 0.442906, 0.170459},
		{0.154183, 0.38592, 0.497451, 0.203884, 0.362608, 0.355691, 0.125349, 0.146766, 0.315143, 0.318133, 0.172547, 0.112769},
		{0.201289, 0.42033, 0.509467, 0.259247, 0.322772, 0.325837, 0.140072, 0.177756, 0.320356, 0.228176, 0.148994, 0.132588},
		{0.187921, 0.302804, 0.46976, 0.302809, 0.183035, 0.228691, 0.206216, 0.35174, 0.308208, 0.233234, 0.316017, 0.243563},
		{0.213539, 0.240346, 0.308664, 0.250704, 0.204879, 0.365022, 0.241966, 0.312579, 0.361886, 0.277293, 0.338944, 0.290351},
		{0.227784, 0.252841, 0.295752, 0.265796, 0.227973, 0.451155, 0.219418, 0.272508, 0.376082, 0.312717, 0.285395, 0.165745},
		{0.168662, 0.180795, 0.264397, 0.225101, 0.562332, 0.33243, 0.236684, 0.199847, 0.409727, 0.247569, 0.21153, 0.147286},
		{0.0491864, 0.0503369, 0.130942, 0.0505802, 0.0694409, 0.0303877, 0.0389852, 0.674067, 0.712933, 0.05762, 0.0245158, 0.0389336},
		{0.0814379, 0.0312366, 0.240546, 0.134609, 0.063374, 0.0466124, 0.0752175, 0.657041, 0.680085, 0.0720311, 0.0249404, 0.0673359},
		{0.139331, 0.0173442, 0.49035, 0.287237, 0.0453947, 0.0873279, 0.15423, 0.447475, 0.621502, 0.127166, 0.0355933, 0.141163},
		{0.115417, 0.0132515, 0.356601, 0.245902, 0.0283943, 0.0588233, 0.117077, 0.499376, 0.715366, 0.100398, 0.0281382, 0.0943482},
		{0.047297, 0.0065354, 0.181074, 0.121455, 0.0135504, 0.030693, 0.0613105, 0.631705, 0.73548, 0.0550565, 0.0128093, 0.0460393},
	}

	if image.NumRows() != len(expected) {
		t.Fatalf("rows = %d, want %d", image.NumRows(), len(expected))
	}
	for y := 0; y < len(expected); y++ {
		row := image.Row(y)
		for x := 0; x < 12; x++ {
			if math.Abs(row[x]-expected[y][x]) > 1e-2 {
				t.Fatalf("row %d col %d: got %g, want %g", y, x, row[x], expected[y][x])
			}
		}
	}
}

func TestAllAlgorithmsEndToEnd(t *testing.T) {
	data := loadAudioFile(t, "test_stereo_44100.raw")

	algos := []gochromaprint.Algorithm{
		gochromaprint.AlgorithmTest1,
		gochromaprint.AlgorithmTest2,
		gochromaprint.AlgorithmTest3,
		gochromaprint.AlgorithmTest4,
		gochromaprint.AlgorithmTest5,
	}

	for _, algo := range algos {
		t.Run(fmt.Sprintf("Algo_%d", algo), func(t *testing.T) {
			ctx := gochromaprint.New(algo)
			if ctx.Algorithm() != algo {
				t.Fatalf("Algorithm = %d, want %d", ctx.Algorithm(), algo)
			}
			if ctx.Config().SampleRate != 11025 {
				t.Fatalf("Config.SampleRate = %d, want 11025", ctx.Config().SampleRate)
			}
			if err := ctx.Start(44100, 1); err != nil {
				t.Fatalf("Start failed: %v", err)
			}
			if err := ctx.Feed(data); err != nil {
				t.Fatalf("Feed failed: %v", err)
			}
			if err := ctx.Finish(); err != nil {
				t.Fatalf("Finish failed: %v", err)
			}

			fp, err := ctx.Fingerprint()
			if err != nil {
				t.Fatalf("Fingerprint failed: %v", err)
			}
			if len(fp) == 0 {
				t.Fatal("expected non-empty fingerprint")
			}

			hash, err := ctx.FingerprintHash()
			if err != nil {
				t.Fatalf("FingerprintHash failed: %v", err)
			}
			if hash == 0 {
				t.Fatal("expected non-zero hash")
			}

			raw, err := ctx.RawFingerprint()
			if err != nil {
				t.Fatalf("RawFingerprint failed: %v", err)
			}
			if len(raw) == 0 {
				t.Fatal("expected non-empty raw fingerprint")
			}
		})
	}
}

func TestAPITestFp(t *testing.T) {
	data := loadAudioFile(t, "test_stereo_44100.raw")

	ctx := gochromaprint.New(gochromaprint.AlgorithmTest2)
	if ctx.NumChannels() != 1 {
		t.Fatalf("NumChannels = %d, want 1", ctx.NumChannels())
	}
	if ctx.SampleRate() != 11025 {
		t.Fatalf("SampleRate = %d, want 11025", ctx.SampleRate())
	}

	if err := ctx.Start(44100, 1); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Feed(data); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Finish(); err != nil {
		t.Fatal(err)
	}

	fp, err := ctx.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	hash, err := ctx.FingerprintHash()
	if err != nil {
		t.Fatal(err)
	}

	const wantFp = "AQAAC0kkZUqYREkUnFAXHk8uuMZl6EfO4zu-4ABKFGESWIIMEQE"
	const wantHash = uint32(3732003127)

	if fp != wantFp {
		t.Fatalf("Fingerprint = %q, want %q", fp, wantFp)
	}
	if hash != wantHash {
		t.Fatalf("FingerprintHash = %d, want %d", hash, wantHash)
	}
}

func TestAPITest2SilenceFp(t *testing.T) {
	zeroes := make([]int16, 1024)

	ctx := gochromaprint.New(gochromaprint.AlgorithmTest2)
	if err := ctx.Start(44100, 1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 130; i++ {
		if err := ctx.Feed(zeroes); err != nil {
			t.Fatal(err)
		}
	}
	if err := ctx.Finish(); err != nil {
		t.Fatal(err)
	}

	fp, err := ctx.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	hash, err := ctx.FingerprintHash()
	if err != nil {
		t.Fatal(err)
	}

	const wantFp = "AQAAA0mUaEkSRZEGAA"
	const wantHash = uint32(627964279)

	if fp != wantFp {
		t.Fatalf("Fingerprint = %q, want %q", fp, wantFp)
	}
	if hash != wantHash {
		t.Fatalf("FingerprintHash = %d, want %d", hash, wantHash)
	}
}

func TestAPITest2SilenceRawFp(t *testing.T) {
	zeroes := make([]int16, 1024)

	ctx := gochromaprint.New(gochromaprint.AlgorithmTest2)
	if err := ctx.Start(44100, 1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 130; i++ {
		if err := ctx.Feed(zeroes); err != nil {
			t.Fatal(err)
		}
	}
	if err := ctx.Finish(); err != nil {
		t.Fatal(err)
	}

	raw, err := ctx.RawFingerprint()
	if err != nil {
		t.Fatal(err)
	}

	if len(raw) != 3 {
		t.Fatalf("len(raw) = %d, want 3", len(raw))
	}
	for i, v := range raw {
		if v != 627964279 {
			t.Fatalf("raw[%d] = %d, want 627964279", i, v)
		}
	}
}

func TestEncodeDecodeFingerprint(t *testing.T) {
	raw := []uint32{1, 0}
	expectedBinary := []byte{55, 0, 0, 2, 65, 0}
	const expectedBase64 = "NwAAAkEA"

	// Binary encode
	encBin, err := gochromaprint.EncodeFingerprint(raw, 55, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encBin, expectedBinary) {
		t.Fatalf("EncodeFingerprint(bin) = %v, want %v", encBin, expectedBinary)
	}

	// Base64 encode
	encB64, err := gochromaprint.EncodeFingerprint(raw, 55, true)
	if err != nil {
		t.Fatal(err)
	}
	if string(encB64) != expectedBase64 {
		t.Fatalf("EncodeFingerprint(b64) = %q, want %q", string(encB64), expectedBase64)
	}

	// Binary decode
	decFp, algo, err := gochromaprint.DecodeFingerprint(expectedBinary, false)
	if err != nil {
		t.Fatal(err)
	}
	if algo != 55 || !reflect.DeepEqual(decFp, raw) {
		t.Fatalf("DecodeFingerprint(bin) = %v, %v, want %v, 55", decFp, algo, raw)
	}

	// Text decode
	const textData = "AQAAEwkjrUmSJQpUHflR9mjSJMdZpcO_Imdw9dCO9Clu4_wQPvhCB01w6xAtXNcAp5RASgDBhDSCGGIAcwA"
	decFp2, algo2, err := gochromaprint.DecodeFingerprint([]byte(textData), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(decFp2) != 19 || algo2 != 1 {
		t.Fatalf("len = %d, algo = %d, want 19, 1", len(decFp2), algo2)
	}
	v0 := int32(-587455133)
	v1 := int32(-591649759)
	if decFp2[0] != uint32(v0) || decFp2[1] != uint32(v1) {
		t.Fatalf("unexpected decFp2 values: %v, %v", decFp2[0], decFp2[1])
	}
}

func TestDecodeFingerprintOutOfRange(t *testing.T) {
	const encoded = "AQABz9GSJEkkKYmS4Tp6xvjx-tBpXMfRpUP1w3yGHmUPcWTwhnhgiYNeeBcu5HhbxA2PZuEFnSd-9FECq-gJPbtQ_fiLfkf3wvnRl9AO__jxCj7-QjzyoFek4xcmPnA2-qjUDz9yTsR56PFRJp4GP8ePLj2qY-NRfkL2oXEx6cFThPCh7_iRVnzQbX9wHc0UT0I_5Duao1KVGOsPlpdQLcqFHxf5oqfRFVpySvjR_Ch7_NCO87h05ChL7Do65vDIFC2H8zhyKnhqXJ6GdsuLw4c276h-TGEcfAfjmWgeYzy6HT6u5UhvhCceZSp0PD5iZzka9Zh05NOEZmsgxZTx50ifa8hxpRv0E1vHKciFHk0cFY0moRKzabgbDb3gH_2RH8k3lGuEKc_x3_hRPehX5ODRfDX6JCtOIVyOMFyP-4XyHWkaF9fRt0VzD7sOJlUkHX2Q9Ed-9PnR8EpwijfCJJFi9PmhLySPhmOC-sIZiI8m_ML3INJ3pI6UknCNnkHzyNDEpchz400d9IN99PiSo9kyDf0xXTrS_HCC79D64Tx4o1YSIReeCKH84Q0ORT7CNMKPmQma62inD7xoodlEJNOJXEHzo3qCMEn4omWg_UPKqBN69EYz5kPyDrkeCb2OKd2i4RJ6PB3CQ5Wo5shTMUH7Hf6IHsd5NNcQnjFGo8oP__DJQz8OJk9s5BROotERSmSOjyaUv0LqsziV4zmaK5jIFLXAxEk0xIfeIcyFqiEVXL_hGLnQUOXRJ8SHSLnRr_jx5IU35CH0N0gfHtdQ_Wj0ZniT7ENZXPDRrijZIMxn7Dkqo0nIoG_QVDsO_mi6yOibDJqyHf6DSiX2fdAyVEyCXDyqhD2aRyf2H09EMJkchE-g60GYK3grNHHwHE9I9DrSMoJ2ZMd7_Ch7WEftI24DqTrCEteio9ePLxIVIj-uW1CjK6goVoYnhYZuuElwNDfyIT4e4lGWKWhEyUF5XKic47xQ0ZygO0XXhAgTMzP-FN_x4_lQiynCxBlxPgivIEyNL0e64zu-hCR-fMzBieLxIzy0MEZ-MNcc-Bqc53gYUfCPh0c6MmFI6DWaijk-K8g_PNXRhKjTD72UBSXzgT_-YIx27EsUJ6iThkJTZUd-aM_R03iaCk-CUwITPUcPVVfwLA8m0UbTrLgSHR3-Y8-DUsKVB6EkJWh-_HgPTv3AZs_R23CcBfWRM1B1hB1K63iipLg9XETcZ0I3jynO8PiF64YtH1klhDvxhThXzFex3kG1HPZu9Dmu47mh5UGtHKp1POjyUMWfDJo6of8Gtgvy5EDfo1bgnWiX6wCGhEEAKADJIUAbILJzTAgDBDBCAgQQMEQQJABBwgJgBBMAAGaEAMwAh5wAikDkhQBGCIGkAgQASghihAmlEBDEMCOUAYYJ4pRVwhAEAiXEYWAUMchQBpQVhkwhgDUMAQAoMkgxJRUwjgFBBRBCEGM0BgBAAIhQRBJkkALCCAUMQIAIIQBQRDGDBAECAAGY4MYQIhSgBCEkKAHEggG8cAIYJIgUwJAHpCHAMAaEQEYo6IgAADgEAUIIBAQcMEIgYBkECEABCEYKAYYcA4oowBBABAAvBJJKMGIAEAYwA5QADAAGCINMaAKQJcQYjCCzxBBACAcGMM4II8IQKIghgFgABTOEUAegMQQ4wYCBSjgAiAFCEGCAYwZIIAA0QChAhBROUAQAEo4YR4AQhhhliAAMEAQUBtooAg5xjCADgCGAOCQIQgQghigwQAikBBAEAA"
	raw, algo, err := gochromaprint.DecodeFingerprint([]byte(encoded), true)
	if err != nil {
		t.Fatalf("DecodeFingerprint failed: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected non-empty raw fingerprint")
	}
	if algo != 1 {
		t.Fatalf("algo = %d, want 1", algo)
	}
}

func TestDecodeFingerprintEmpty(t *testing.T) {
	const encoded = "AQAAAA"
	raw, algo, err := gochromaprint.DecodeFingerprint([]byte(encoded), true)
	if err != nil {
		t.Fatalf("DecodeFingerprint failed: %v", err)
	}
	if len(raw) != 0 {
		t.Fatalf("expected empty raw fingerprint, got %d", len(raw))
	}
	if algo != 1 {
		t.Fatalf("algo = %d, want 1", algo)
	}
}

func TestDecodeFingerprintCorrupted(t *testing.T) {
	corrupted := []string{"null", "coustid Fingerprint", "AQ", ""}
	for _, s := range corrupted {
		_, _, err := gochromaprint.DecodeFingerprint([]byte(s), true)
		if err == nil {
			t.Fatalf("expected error for %q, got nil", s)
		}
	}
}

func TestDecodeFingerprintHeaders(t *testing.T) {
	binaryHeader := []byte{55, 0, 0, 2, 65, 0}
	sz, algo, err := gochromaprint.DecodeFingerprintHeader(binaryHeader, false)
	if err != nil {
		t.Fatal(err)
	}
	if sz != 2 || algo != 55 {
		t.Fatalf("sz = %d, algo = %d, want 2, 55", sz, algo)
	}

	const textData = "AQAAEwkjrUmSJQpUHflR9mjSJMdZpcO_Imdw9dCO9Clu4_wQPvhCB01w6xAtXNcAp5RASgDBhDSCGGIAcwA"
	sz2, algo2, err := gochromaprint.DecodeFingerprintHeader([]byte(textData), true)
	if err != nil {
		t.Fatal(err)
	}
	if sz2 != 19 || algo2 != 1 {
		t.Fatalf("sz2 = %d, algo2 = %d, want 19, 1", sz2, algo2)
	}

	// Invalid header tests
	if _, _, err := gochromaprint.DecodeFingerprintHeader([]byte{1, 2}, false); err == nil {
		t.Fatal("expected error for short header")
	}
	if _, _, err := gochromaprint.DecodeFingerprintHeader([]byte("invalid!!"), true); err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestContextAlgorithmMetadata(t *testing.T) {
	ctx := gochromaprint.New(gochromaprint.AlgorithmTest2)
	if ctx.ItemDuration() != 1365 {
		t.Fatalf("ItemDuration = %d, want 1365", ctx.ItemDuration())
	}
	if ctx.ItemDurationMs() != 123 {
		t.Fatalf("ItemDurationMs = %d, want 123", ctx.ItemDurationMs())
	}
	if ctx.Delay() != 28666 {
		t.Fatalf("Delay = %d, want 28666", ctx.Delay())
	}
	if ctx.DelayMs() != 2600 {
		t.Fatalf("DelayMs = %d, want 2600", ctx.DelayMs())
	}
}

func TestContextLifecycleErrors(t *testing.T) {
	ctx := gochromaprint.New(gochromaprint.AlgorithmTest2)
	// Feeding before start
	if err := ctx.Feed([]int16{1, 2, 3}); err != gochromaprint.ErrContextNotStarted {
		t.Fatalf("expected ErrContextNotStarted, got %v", err)
	}
	if err := ctx.Finish(); err != gochromaprint.ErrContextNotStarted {
		t.Fatalf("expected ErrContextNotStarted, got %v", err)
	}

	// Invalid options
	if err := ctx.SetOption("invalid_opt", 123); err != gochromaprint.ErrUnknownOption {
		t.Fatalf("expected ErrUnknownOption, got %v", err)
	}

	// Silence threshold option
	ctx4 := gochromaprint.New(gochromaprint.AlgorithmTest4)
	if err := ctx4.SetOption("silence_threshold", 80); err != nil {
		t.Fatalf("unexpected error setting silence_threshold: %v", err)
	}

	// Unknown algorithm fallback
	ctxFallback := gochromaprint.New(gochromaprint.Algorithm(999))
	if ctxFallback.Algorithm() != gochromaprint.AlgorithmDefault {
		t.Fatalf("expected default algorithm fallback, got %d", ctxFallback.Algorithm())
	}
}

func TestContextClearAndContinue(t *testing.T) {
	data := loadAudioFile(t, "test_stereo_44100.raw")
	ctx := gochromaprint.New(gochromaprint.AlgorithmTest2)
	if err := ctx.Start(44100, 2); err != nil {
		t.Fatal(err)
	}

	// Feed first half
	half := len(data) / 2
	if err := ctx.Feed(data[:half]); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Finish(); err != nil {
		t.Fatal(err)
	}
	fp1, err := ctx.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if len(fp1) == 0 {
		t.Fatal("expected non-empty fp1")
	}

	// Clear and feed second half
	ctx.Clear()
	if ctx.RawFingerprintSize() != 0 {
		t.Fatalf("expected size 0 after Clear, got %d", ctx.RawFingerprintSize())
	}

	if err := ctx.Feed(data[half:]); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Finish(); err != nil {
		t.Fatal(err)
	}
	fp2, err := ctx.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if len(fp2) == 0 {
		t.Fatal("expected non-empty fp2")
	}
}
