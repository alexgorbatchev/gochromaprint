package bitpack

// PackedInt3Size returns the number of bytes required to pack `size` 3-bit integers.
func PackedInt3Size(size int) int {
	return (size*3 + 7) / 8
}

// UnpackedInt3Size returns the number of 3-bit integers that can be extracted from `byteCount` bytes.
func UnpackedInt3Size(byteCount int) int {
	return byteCount * 8 / 3
}

// PackInt3 packs a slice of 3-bit integers (values 0..7) into a byte slice.
func PackInt3(src []byte) []byte {
	size := len(src)
	dst := make([]byte, PackedInt3Size(size))
	destIdx := 0
	srcIdx := 0

	for size >= 8 {
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		s4, s5, s6, s7 := src[srcIdx+4], src[srcIdx+5], src[srcIdx+6], src[srcIdx+7]
		dst[destIdx] = (s0 & 0x07) | ((s1 & 0x07) << 3) | ((s2 & 0x03) << 6)
		dst[destIdx+1] = ((s2 & 0x04) >> 2) | ((s3 & 0x07) << 1) | ((s4 & 0x07) << 4) | ((s5 & 0x01) << 7)
		dst[destIdx+2] = ((s5 & 0x06) >> 1) | ((s6 & 0x07) << 2) | ((s7 & 0x07) << 5)
		srcIdx += 8
		destIdx += 3
		size -= 8
	}

	switch size {
	case 7:
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		s4, s5, s6 := src[srcIdx+4], src[srcIdx+5], src[srcIdx+6]
		dst[destIdx] = (s0 & 0x07) | ((s1 & 0x07) << 3) | ((s2 & 0x03) << 6)
		dst[destIdx+1] = ((s2 & 0x04) >> 2) | ((s3 & 0x07) << 1) | ((s4 & 0x07) << 4) | ((s5 & 0x01) << 7)
		dst[destIdx+2] = ((s5 & 0x06) >> 1) | ((s6 & 0x07) << 2)
	case 6:
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		s4, s5 := src[srcIdx+4], src[srcIdx+5]
		dst[destIdx] = (s0 & 0x07) | ((s1 & 0x07) << 3) | ((s2 & 0x03) << 6)
		dst[destIdx+1] = ((s2 & 0x04) >> 2) | ((s3 & 0x07) << 1) | ((s4 & 0x07) << 4) | ((s5 & 0x01) << 7)
		dst[destIdx+2] = (s5 & 0x06) >> 1
	case 5:
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		s4 := src[srcIdx+4]
		dst[destIdx] = (s0 & 0x07) | ((s1 & 0x07) << 3) | ((s2 & 0x03) << 6)
		dst[destIdx+1] = ((s2 & 0x04) >> 2) | ((s3 & 0x07) << 1) | ((s4 & 0x07) << 4)
	case 4:
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		dst[destIdx] = (s0 & 0x07) | ((s1 & 0x07) << 3) | ((s2 & 0x03) << 6)
		dst[destIdx+1] = ((s2 & 0x04) >> 2) | ((s3 & 0x07) << 1)
	case 3:
		s0, s1, s2 := src[srcIdx], src[srcIdx+1], src[srcIdx+2]
		dst[destIdx] = (s0 & 0x07) | ((s1 & 0x07) << 3) | ((s2 & 0x03) << 6)
		dst[destIdx+1] = (s2 & 0x04) >> 2
	case 2:
		s0, s1 := src[srcIdx], src[srcIdx+1]
		dst[destIdx] = (s0 & 0x07) | ((s1 & 0x07) << 3)
	case 1:
		s0 := src[srcIdx]
		dst[destIdx] = s0 & 0x07
	}

	return dst
}

// UnpackInt3 unpacks a slice of bytes into 3-bit integers.
func UnpackInt3(src []byte) []byte {
	size := len(src)
	dst := make([]byte, UnpackedInt3Size(size))
	destIdx := 0
	srcIdx := 0

	for size >= 3 {
		s0, s1, s2 := src[srcIdx], src[srcIdx+1], src[srcIdx+2]
		dst[destIdx] = s0 & 0x07
		dst[destIdx+1] = (s0 & 0x38) >> 3
		dst[destIdx+2] = ((s0 & 0xc0) >> 6) | ((s1 & 0x01) << 2)
		dst[destIdx+3] = (s1 & 0x0e) >> 1
		dst[destIdx+4] = (s1 & 0x70) >> 4
		dst[destIdx+5] = ((s1 & 0x80) >> 7) | ((s2 & 0x03) << 1)
		dst[destIdx+6] = (s2 & 0x1c) >> 2
		dst[destIdx+7] = (s2 & 0xe0) >> 5
		srcIdx += 3
		destIdx += 8
		size -= 3
	}

	switch size {
	case 2:
		s0, s1 := src[srcIdx], src[srcIdx+1]
		dst[destIdx] = s0 & 0x07
		dst[destIdx+1] = (s0 & 0x38) >> 3
		dst[destIdx+2] = ((s0 & 0xc0) >> 6) | ((s1 & 0x01) << 2)
		dst[destIdx+3] = (s1 & 0x0e) >> 1
		dst[destIdx+4] = (s1 & 0x70) >> 4
	case 1:
		s0 := src[srcIdx]
		dst[destIdx] = s0 & 0x07
		dst[destIdx+1] = (s0 & 0x38) >> 3
	}

	return dst
}

// PackedInt5Size returns the number of bytes required to pack `size` 5-bit integers.
func PackedInt5Size(size int) int {
	return (size*5 + 7) / 8
}

// UnpackedInt5Size returns the number of 5-bit integers that can be extracted from `byteCount` bytes.
func UnpackedInt5Size(byteCount int) int {
	return byteCount * 8 / 5
}

// PackInt5 packs a slice of 5-bit integers (values 0..31) into a byte slice.
func PackInt5(src []byte) []byte {
	size := len(src)
	dst := make([]byte, PackedInt5Size(size))
	destIdx := 0
	srcIdx := 0

	for size >= 8 {
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		s4, s5, s6, s7 := src[srcIdx+4], src[srcIdx+5], src[srcIdx+6], src[srcIdx+7]
		dst[destIdx] = (s0 & 0x1f) | ((s1 & 0x07) << 5)
		dst[destIdx+1] = ((s1 & 0x18) >> 3) | ((s2 & 0x1f) << 2) | ((s3 & 0x01) << 7)
		dst[destIdx+2] = ((s3 & 0x1e) >> 1) | ((s4 & 0x0f) << 4)
		dst[destIdx+3] = ((s4 & 0x10) >> 4) | ((s5 & 0x1f) << 1) | ((s6 & 0x03) << 6)
		dst[destIdx+4] = ((s6 & 0x1c) >> 2) | ((s7 & 0x1f) << 3)
		srcIdx += 8
		destIdx += 5
		size -= 8
	}

	switch size {
	case 7:
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		s4, s5, s6 := src[srcIdx+4], src[srcIdx+5], src[srcIdx+6]
		dst[destIdx] = (s0 & 0x1f) | ((s1 & 0x07) << 5)
		dst[destIdx+1] = ((s1 & 0x18) >> 3) | ((s2 & 0x1f) << 2) | ((s3 & 0x01) << 7)
		dst[destIdx+2] = ((s3 & 0x1e) >> 1) | ((s4 & 0x0f) << 4)
		dst[destIdx+3] = ((s4 & 0x10) >> 4) | ((s5 & 0x1f) << 1) | ((s6 & 0x03) << 6)
		dst[destIdx+4] = (s6 & 0x1c) >> 2
	case 6:
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		s4, s5 := src[srcIdx+4], src[srcIdx+5]
		dst[destIdx] = (s0 & 0x1f) | ((s1 & 0x07) << 5)
		dst[destIdx+1] = ((s1 & 0x18) >> 3) | ((s2 & 0x1f) << 2) | ((s3 & 0x01) << 7)
		dst[destIdx+2] = ((s3 & 0x1e) >> 1) | ((s4 & 0x0f) << 4)
		dst[destIdx+3] = ((s4 & 0x10) >> 4) | ((s5 & 0x1f) << 1)
	case 5:
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		s4 := src[srcIdx+4]
		dst[destIdx] = (s0 & 0x1f) | ((s1 & 0x07) << 5)
		dst[destIdx+1] = ((s1 & 0x18) >> 3) | ((s2 & 0x1f) << 2) | ((s3 & 0x01) << 7)
		dst[destIdx+2] = ((s3 & 0x1e) >> 1) | ((s4 & 0x0f) << 4)
		dst[destIdx+3] = (s4 & 0x10) >> 4
	case 4:
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		dst[destIdx] = (s0 & 0x1f) | ((s1 & 0x07) << 5)
		dst[destIdx+1] = ((s1 & 0x18) >> 3) | ((s2 & 0x1f) << 2) | ((s3 & 0x01) << 7)
		dst[destIdx+2] = (s3 & 0x1e) >> 1
	case 3:
		s0, s1, s2 := src[srcIdx], src[srcIdx+1], src[srcIdx+2]
		dst[destIdx] = (s0 & 0x1f) | ((s1 & 0x07) << 5)
		dst[destIdx+1] = ((s1 & 0x18) >> 3) | ((s2 & 0x1f) << 2)
	case 2:
		s0, s1 := src[srcIdx], src[srcIdx+1]
		dst[destIdx] = (s0 & 0x1f) | ((s1 & 0x07) << 5)
		dst[destIdx+1] = (s1 & 0x18) >> 3
	case 1:
		s0 := src[srcIdx]
		dst[destIdx] = s0 & 0x1f
	}

	return dst
}

// UnpackInt5 unpacks a slice of bytes into 5-bit integers.
func UnpackInt5(src []byte) []byte {
	size := len(src)
	dst := make([]byte, UnpackedInt5Size(size))
	destIdx := 0
	srcIdx := 0

	for size >= 5 {
		s0, s1, s2, s3, s4 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3], src[srcIdx+4]
		dst[destIdx] = s0 & 0x1f
		dst[destIdx+1] = ((s0 & 0xe0) >> 5) | ((s1 & 0x03) << 3)
		dst[destIdx+2] = (s1 & 0x7c) >> 2
		dst[destIdx+3] = ((s1 & 0x80) >> 7) | ((s2 & 0x0f) << 1)
		dst[destIdx+4] = ((s2 & 0xf0) >> 4) | ((s3 & 0x01) << 4)
		dst[destIdx+5] = (s3 & 0x3e) >> 1
		dst[destIdx+6] = ((s3 & 0xc0) >> 6) | ((s4 & 0x07) << 2)
		dst[destIdx+7] = (s4 & 0xf8) >> 3
		srcIdx += 5
		destIdx += 8
		size -= 5
	}

	switch size {
	case 4:
		s0, s1, s2, s3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]
		dst[destIdx] = s0 & 0x1f
		dst[destIdx+1] = ((s0 & 0xe0) >> 5) | ((s1 & 0x03) << 3)
		dst[destIdx+2] = (s1 & 0x7c) >> 2
		dst[destIdx+3] = ((s1 & 0x80) >> 7) | ((s2 & 0x0f) << 1)
		dst[destIdx+4] = ((s2 & 0xf0) >> 4) | ((s3 & 0x01) << 4)
		dst[destIdx+5] = (s3 & 0x3e) >> 1
	case 3:
		s0, s1, s2 := src[srcIdx], src[srcIdx+1], src[srcIdx+2]
		dst[destIdx] = s0 & 0x1f
		dst[destIdx+1] = ((s0 & 0xe0) >> 5) | ((s1 & 0x03) << 3)
		dst[destIdx+2] = (s1 & 0x7c) >> 2
		dst[destIdx+3] = ((s1 & 0x80) >> 7) | ((s2 & 0x0f) << 1)
	case 2:
		s0, s1 := src[srcIdx], src[srcIdx+1]
		dst[destIdx] = s0 & 0x1f
		dst[destIdx+1] = ((s0 & 0xe0) >> 5) | ((s1 & 0x03) << 3)
		dst[destIdx+2] = (s1 & 0x7c) >> 2
	case 1:
		s0 := src[srcIdx]
		dst[destIdx] = s0 & 0x1f
	}

	return dst
}
