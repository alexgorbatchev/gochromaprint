package gochromaprint

import "math/bits"

// CountSetBits returns the number of 1-bits in v.
func CountSetBits(v uint32) int {
	return bits.OnesCount32(v)
}

// HammingDistance returns the number of bit differences between a and b.
func HammingDistance(a, b uint32) int {
	return bits.OnesCount32(a ^ b)
}

// SimHash computes a 32-bit locality-sensitive hash of a raw fingerprint.
func SimHash(data []uint32) uint32 {
	size := len(data)
	if size == 0 {
		return 0
	}

	var v [32]int
	for _, localHash := range data {
		for j := 0; j < 32; j++ {
			v[j] += int((localHash >> j) & 1)
		}
	}

	threshold := size / 2
	var hash uint32
	for i := 0; i < 32; i++ {
		if v[i] > threshold {
			hash |= (1 << i)
		}
	}

	return hash
}

// HashFingerprint is an alias for SimHash.
func HashFingerprint(data []uint32) uint32 {
	return SimHash(data)
}
