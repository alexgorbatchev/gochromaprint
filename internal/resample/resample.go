package resample

import (
	"math"
)

const (
	filterShift           = 15
	windowType            = 9
	kResampleFilterLength = 16
	kResamplePhaseShift   = 8
	kResampleLinear       = 0
	kResampleCutoff       = 0.8
)

// Resampler converts audio sample rates using a polyphase sinc filterbank.
type Resampler struct {
	filterBank           []int16
	filterLength         int
	idealDstIncr         int
	dstIncr              int
	index                int
	frac                 int
	srcIncr              int
	compensationDistance int
	phaseShift           int
	phaseMask            int
	linear               int
}

func bessel(x float64) float64 {
	v := 1.0
	lastv := 0.0
	t := 1.0
	x = x * x / 4.0
	for i := 1.0; v != lastv; i++ {
		lastv = v
		t *= x / (i * i)
		v += t
	}
	return v
}

func buildFilter(filter []int16, factor float64, tapCount, phaseCount, scale, filterType int) {
	center := (tapCount - 1) / 2
	if factor > 1.0 {
		factor = 1.0
	}

	tab := make([]float64, tapCount)
	for ph := 0; ph < phaseCount; ph++ {
		var norm float64
		for i := 0; i < tapCount; i++ {
			x := math.Pi * (float64(i-center) - float64(ph)/float64(phaseCount)) * factor
			var y float64
			if x == 0 {
				y = 1.0
			} else {
				y = math.Sin(x) / x
			}

			switch filterType {
			case 0:
				const d = -0.5
				absX := math.Abs((float64(i-center) - float64(ph)/float64(phaseCount)) * factor)
				if absX < 1.0 {
					y = 1 - 3*absX*absX + 2*absX*absX*absX + d*(-absX*absX+absX*absX*absX)
				} else {
					y = d * (-4 + 8*absX - 5*absX*absX + absX*absX*absX)
				}
			case 1:
				w := 2.0*x/(factor*float64(tapCount)) + math.Pi
				y *= 0.3635819 - 0.4891775*math.Cos(w) + 0.1365995*math.Cos(2*w) - 0.0106411*math.Cos(3*w)
			default:
				w := 2.0 * x / (factor * float64(tapCount) * math.Pi)
				arg := 1.0 - w*w
				if arg < 0 {
					arg = 0
				}
				y *= bessel(float64(filterType) * math.Sqrt(arg))
			}

			tab[i] = y
			norm += y
		}

		for i := 0; i < tapCount; i++ {
			val := math.Round(tab[i] * float64(scale) / norm)
			if val > math.MaxInt16 {
				val = math.MaxInt16
			} else if val < math.MinInt16 {
				val = math.MinInt16
			}
			filter[ph*tapCount+i] = int16(val)
		}
	}
}

// New creates a new Resampler for converting between inRate and outRate.
func New(outRate, inRate int) *Resampler {
	factor := float64(outRate) * kResampleCutoff / float64(inRate)
	if factor > 1.0 {
		factor = 1.0
	}

	phaseShift := kResamplePhaseShift
	phaseCount := 1 << phaseShift
	phaseMask := phaseCount - 1
	filterLength := int(math.Ceil(float64(kResampleFilterLength) / factor))
	if filterLength < 1 {
		filterLength = 1
	}

	filterBank := make([]int16, filterLength*(phaseCount+1))
	buildFilter(filterBank, factor, filterLength, phaseCount, 1<<filterShift, windowType)

	filterBank[filterLength*phaseCount] = filterBank[filterLength-1]
	copy(filterBank[filterLength*phaseCount+1:], filterBank[:filterLength-1])

	return &Resampler{
		filterBank:   filterBank,
		filterLength: filterLength,
		idealDstIncr: inRate * phaseCount,
		dstIncr:      inRate * phaseCount,
		srcIncr:      outRate,
		index:        -phaseCount * ((filterLength - 1) / 2),
		phaseShift:   phaseShift,
		phaseMask:    phaseMask,
		linear:       kResampleLinear,
	}
}

// Resample resamples audio data from src into dst.
// It returns the number of input samples consumed and output samples produced.
func (r *Resampler) Resample(dst []int16, src []int16, updateCtx bool) (consumed int, produced int) {
	srcSize := len(src)
	dstSize := len(dst)
	if srcSize == 0 || dstSize == 0 {
		return 0, 0
	}

	index := r.index
	frac := r.frac
	dstIncrFrac := r.dstIncr % r.srcIncr
	dstIncr := r.dstIncr / r.srcIncr
	compensationDistance := r.compensationDistance

	dstIndex := 0
	for ; dstIndex < dstSize; dstIndex++ {
		filterOffset := r.filterLength * (index & r.phaseMask)
		filter := r.filterBank[filterOffset : filterOffset+r.filterLength]
		sampleIndex := index >> r.phaseShift
		var val int32

		if sampleIndex < 0 {
			for i := 0; i < r.filterLength; i++ {
				idx := sampleIndex + i
				if idx < 0 {
					idx = -idx
				}
				idx %= srcSize
				val += int32(src[idx]) * int32(filter[i])
			}
		} else if sampleIndex+r.filterLength > srcSize {
			break
		} else {
			for i := 0; i < r.filterLength; i++ {
				val += int32(src[sampleIndex+i]) * int32(filter[i])
			}
		}

		val = (val + (1 << (filterShift - 1))) >> filterShift
		if val > math.MaxInt16 {
			val = math.MaxInt16
		} else if val < math.MinInt16 {
			val = math.MinInt16
		}
		dst[dstIndex] = int16(val)

		frac += dstIncrFrac
		index += dstIncr
		if frac >= r.srcIncr {
			frac -= r.srcIncr
			index++
		}

		if dstIndex+1 == compensationDistance {
			compensationDistance = 0
			dstIncrFrac = r.idealDstIncr % r.srcIncr
			dstIncr = r.idealDstIncr / r.srcIncr
		}
	}

	consumed = index
	if consumed < 0 {
		consumed = 0
	}
	consumed >>= r.phaseShift

	if index >= 0 {
		index &= r.phaseMask
	}

	if compensationDistance > 0 {
		compensationDistance -= dstIndex
	}

	if updateCtx {
		r.frac = frac
		r.index = index
		r.dstIncr = dstIncrFrac + r.srcIncr*dstIncr
		r.compensationDistance = compensationDistance
	}

	return consumed, dstIndex
}
