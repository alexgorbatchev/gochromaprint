package bitpack_test

import (
	"reflect"
	"testing"

	"github.com/alexgorbatchev/gochromaprint/internal/bitpack"
)

func TestPackUnpackInt3(t *testing.T) {
	for length := 0; length <= 25; length++ {
		src := make([]byte, length)
		for i := 0; i < length; i++ {
			src[i] = byte(i % 8)
		}
		packed := bitpack.PackInt3(src)
		wantPackedLen := bitpack.PackedInt3Size(length)
		if len(packed) != wantPackedLen {
			t.Fatalf("length %d: packed len = %d, want %d", length, len(packed), wantPackedLen)
		}

		unpacked := bitpack.UnpackInt3(packed)
		if len(unpacked) < length {
			t.Fatalf("length %d: unpacked len = %d, want at least %d", length, len(unpacked), length)
		}
		if !reflect.DeepEqual(src, unpacked[:length]) {
			t.Fatalf("length %d: got %v, want %v", length, unpacked[:length], src)
		}
	}
}

func TestPackUnpackInt5(t *testing.T) {
	for length := 0; length <= 25; length++ {
		src := make([]byte, length)
		for i := 0; i < length; i++ {
			src[i] = byte(i % 32)
		}
		packed := bitpack.PackInt5(src)
		wantPackedLen := bitpack.PackedInt5Size(length)
		if len(packed) != wantPackedLen {
			t.Fatalf("length %d: packed len = %d, want %d", length, len(packed), wantPackedLen)
		}

		unpacked := bitpack.UnpackInt5(packed)
		if len(unpacked) < length {
			t.Fatalf("length %d: unpacked len = %d, want at least %d", length, len(unpacked), length)
		}
		if !reflect.DeepEqual(src, unpacked[:length]) {
			t.Fatalf("length %d: got %v, want %v", length, unpacked[:length], src)
		}
	}
}
