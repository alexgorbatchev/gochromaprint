package compress

import (
	"errors"
	"fmt"

	"github.com/alexgorbatchev/gochromaprint/internal/bitpack"
)

const (
	maxNormalValue = 7
	normalBits     = 3
	exceptionBits  = 5
)

var (
	// ErrTooShort indicates the compressed fingerprint is too short.
	ErrTooShort = errors.New("invalid fingerprint: data too short")
	// ErrInvalidData indicates the fingerprint payload is corrupted.
	ErrInvalidData = errors.New("invalid fingerprint: corrupted data")
)

// Compressor compresses raw 32-bit uint32 fingerprints.
type Compressor struct {
	normalBits      []byte
	exceptionalBits []byte
}

// NewCompressor creates a new Compressor.
func NewCompressor() *Compressor {
	return &Compressor{}
}

func (c *Compressor) processSubfingerprint(x uint32) {
	bit := 1
	lastBit := 0
	for x != 0 {
		if (x & 1) != 0 {
			val := bit - lastBit
			if val >= maxNormalValue {
				c.normalBits = append(c.normalBits, maxNormalValue)
				c.exceptionalBits = append(c.exceptionalBits, byte(val-maxNormalValue))
			} else {
				c.normalBits = append(c.normalBits, byte(val))
			}
			lastBit = bit
		}
		x >>= 1
		bit++
	}
	c.normalBits = append(c.normalBits, 0)
}

// Compress compresses a slice of subfingerprints with the specified algorithm tag.
func (c *Compressor) Compress(data []uint32, algorithm int) []byte {
	size := len(data)
	c.normalBits = c.normalBits[:0]
	c.exceptionalBits = c.exceptionalBits[:0]

	if size > 0 {
		c.processSubfingerprint(data[0])
		for i := 1; i < size; i++ {
			c.processSubfingerprint(data[i] ^ data[i-1])
		}
	}

	packed3 := bitpack.PackInt3(c.normalBits)
	packed5 := bitpack.PackInt5(c.exceptionalBits)

	out := make([]byte, 4+len(packed3)+len(packed5))
	out[0] = byte(algorithm & 0xff)
	out[1] = byte((size >> 16) & 0xff)
	out[2] = byte((size >> 8) & 0xff)
	out[3] = byte(size & 0xff)

	copy(out[4:], packed3)
	copy(out[4+len(packed3):], packed5)
	return out
}

// Compress is a helper function to compress raw fingerprints.
func Compress(data []uint32, algorithm int) []byte {
	c := NewCompressor()
	return c.Compress(data, algorithm)
}

// DecompressHeader extracts the raw size and algorithm from a compressed fingerprint.
func DecompressHeader(input []byte) (size int, algorithm int, err error) {
	if len(input) < 4 {
		return 0, 0, ErrTooShort
	}
	algorithm = int(input[0])
	size = (int(input[1]) << 16) | (int(input[2]) << 8) | int(input[3])
	return size, algorithm, nil
}

// Decompress decompresses a binary compressed fingerprint into raw uint32 subfingerprints.
func Decompress(input []byte) ([]uint32, int, error) {
	size, algorithm, err := DecompressHeader(input)
	if err != nil {
		return nil, 0, err
	}
	if size == 0 {
		return []uint32{}, algorithm, nil
	}

	offset := 4
	if len(input) <= offset {
		return nil, algorithm, fmt.Errorf("%w: missing body", ErrTooShort)
	}

	bits := bitpack.UnpackInt3(input[offset:])

	foundValues := 0
	numExceptionalBits := 0
	bitsLen := 0

	for i, b := range bits {
		if b == 0 {
			foundValues++
			if foundValues == size {
				bitsLen = i + 1
				break
			}
		} else if b == maxNormalValue {
			numExceptionalBits++
		}
	}

	if foundValues != size {
		return nil, algorithm, fmt.Errorf("%w: not enough normal bits", ErrTooShort)
	}

	bits = bits[:bitsLen]
	offset += bitpack.PackedInt3Size(bitsLen)

	if len(input) < offset+bitpack.PackedInt5Size(numExceptionalBits) {
		return nil, algorithm, fmt.Errorf("%w: not enough exceptional bits", ErrTooShort)
	}

	if numExceptionalBits > 0 {
		exceptionalBytes := bitpack.PackedInt5Size(numExceptionalBits)
		exceptionalBits := bitpack.UnpackInt5(input[offset : offset+exceptionalBytes])
		j := 0
		for i := 0; i < len(bits); i++ {
			if bits[i] == maxNormalValue {
				if j >= len(exceptionalBits) {
					return nil, algorithm, fmt.Errorf("%w: exceptional bit index out of range", ErrInvalidData)
				}
				bits[i] += exceptionalBits[j]
				j++
			}
		}
	}

	output := make([]uint32, size)
	itemIdx := 0
	lastBit := 0
	var value uint32

	for _, bit := range bits {
		if bit == 0 {
			output[itemIdx] = value
			lastBit = 0
			itemIdx++
		} else {
			lastBit += int(bit)
			if lastBit > 32 {
				return nil, algorithm, fmt.Errorf("%w: bit shift %d out of 32-bit range", ErrInvalidData, lastBit)
			}
			value ^= 1 << (lastBit - 1)
		}
	}

	return output, algorithm, nil
}
