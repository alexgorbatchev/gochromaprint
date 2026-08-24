package filter

// Gradient computes central differences for interior points and forward/backward differences at edges.
func Gradient(input []float64) []float64 {
	size := len(input)
	if size == 0 {
		return nil
	}
	if size == 1 {
		return []float64{0.0}
	}

	out := make([]float64, size)
	out[0] = input[1] - input[0]
	if size == 2 {
		out[1] = input[1] - input[0]
		return out
	}

	for i := 1; i < size-1; i++ {
		out[i] = (input[i+1] - input[i-1]) / 2.0
	}
	out[size-1] = input[size-1] - input[size-2]
	return out
}
