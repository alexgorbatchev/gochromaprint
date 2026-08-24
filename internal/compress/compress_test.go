package compress_test

import (
	"bytes"
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/alexgorbatchev/gochromaprint/internal/compress"
)

func TestCompressor(t *testing.T) {
	tests := []struct {
		name     string
		input    []uint32
		expected []byte
	}{
		{
			name:     "OneItemOneBit",
			input:    []uint32{1},
			expected: []byte{0, 0, 0, 1, 1},
		},
		{
			name:     "OneItemThreeBits",
			input:    []uint32{7},
			expected: []byte{0, 0, 0, 1, 73, 0},
		},
		{
			name:     "OneItemOneBitExcept",
			input:    []uint32{1 << 6},
			expected: []byte{0, 0, 0, 1, 7, 0},
		},
		{
			name:     "OneItemOneBitExcept2",
			input:    []uint32{1 << 8},
			expected: []byte{0, 0, 0, 1, 7, 2},
		},
		{
			name:     "TwoItems",
			input:    []uint32{1, 0},
			expected: []byte{0, 0, 0, 2, 65, 0},
		},
		{
			name:     "TwoItemsNoChange",
			input:    []uint32{1, 1},
			expected: []byte{0, 0, 0, 2, 1, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compress.Compress(tt.input, 0)
			if !bytes.Equal(got, tt.expected) {
				t.Fatalf("Compress(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDecompress(t *testing.T) {
	t.Run("Header", func(t *testing.T) {
		raw, err := base64.RawURLEncoding.DecodeString("AQAAEwkjrUmSJQpUHflR9mjSJMdZpcO_Imdw9dCO9Clu4_wQPvhCB01w6xAtXNcAp5RASgDBhDSCGGIAcwA")
		if err != nil {
			t.Fatal(err)
		}
		size, algo, err := compress.DecompressHeader(raw)
		if err != nil {
			t.Fatal(err)
		}
		if size != 19 || algo != 1 {
			t.Fatalf("DecompressHeader got size=%d algo=%d, want 19, 1", size, algo)
		}
	})

	tests := []struct {
		name      string
		data      []byte
		want      []uint32
		wantAlgo  int
		wantError bool
	}{
		{
			name:     "OneItemOneBit",
			data:     []byte{0, 0, 0, 1, 1},
			want:     []uint32{1},
			wantAlgo: 0,
		},
		{
			name:     "OneItemThreeBits",
			data:     []byte{0, 0, 0, 1, 73, 0},
			want:     []uint32{7},
			wantAlgo: 0,
		},
		{
			name:     "OneItemOneBitExcept",
			data:     []byte{0, 0, 0, 1, 7, 0},
			want:     []uint32{1 << 6},
			wantAlgo: 0,
		},
		{
			name:     "OneItemOneBitExcept2",
			data:     []byte{0, 0, 0, 1, 7, 2},
			want:     []uint32{1 << 8},
			wantAlgo: 0,
		},
		{
			name:     "OneItemOneBitExceptWithTrailingData",
			data:     []byte{0, 0, 0, 1, 7, 0, 255, 255, 255, 255},
			want:     []uint32{1 << 6},
			wantAlgo: 0,
		},
		{
			name:     "TwoItems",
			data:     []byte{0, 0, 0, 2, 65, 0},
			want:     []uint32{1, 0},
			wantAlgo: 0,
		},
		{
			name:     "TwoItemsNoChange",
			data:     []byte{0, 0, 0, 2, 1, 0},
			want:     []uint32{1, 1},
			wantAlgo: 0,
		},
		{
			name:      "Invalid1",
			data:      []byte{0, 255, 255, 255},
			wantError: true,
		},
		{
			name:     "Empty",
			data:     []byte{1, 0, 0, 0},
			want:     []uint32{},
			wantAlgo: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, algo, err := compress.Decompress(tt.data)
			if (err != nil) != tt.wantError {
				t.Fatalf("Decompress error = %v, wantError = %v", err, tt.wantError)
			}
			if err == nil {
				if !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("Decompress got %v, want %v", got, tt.want)
				}
				if algo != tt.wantAlgo {
					t.Fatalf("Decompress algo = %d, want %d", algo, tt.wantAlgo)
				}
			}
		})
	}

	t.Run("Long", func(t *testing.T) {
		raw, err := base64.RawURLEncoding.DecodeString("AQAAEwkjrUmSJQpUHflR9mjSJMdZpcO_Imdw9dCO9Clu4_wQPvhCB01w6xAtXNcAp5RASgDBhDSCGGIAcwA")
		if err != nil {
			t.Fatal(err)
		}
		expectedSigned := []int32{
			-587455133, -591649759, -574868448, -576973520, -543396544, 1330439488, 1326360000, 1326355649,
			1191625921, 1192674515, 1194804466, 1195336818, 1165981042, 1165956451, 1157441379, 1157441299,
			1291679571, 1291673457, 1170079601,
		}
		expected := make([]uint32, len(expectedSigned))
		for i, v := range expectedSigned {
			expected[i] = uint32(v)
		}

		got, algo, err := compress.Decompress(raw)
		if err != nil {
			t.Fatal(err)
		}
		if algo != 1 {
			t.Fatalf("algo = %d, want 1", algo)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("got %v, want %v", got, expected)
		}

		// Re-compress and verify
		recompressed := compress.Compress(got, algo)
		if !bytes.Equal(recompressed, raw) {
			t.Fatalf("recompressed = %v, want %v", recompressed, raw)
		}
	})
}
