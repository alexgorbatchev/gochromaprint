package gochromaprint

import (
	"errors"
	"math"
	"sort"

	"github.com/alexgorbatchev/gochromaprint/internal/classifier"
	"github.com/alexgorbatchev/gochromaprint/internal/filter"
)

const (
	defaultMatchThreshold = 10.0
	alignBits             = 12
)

var (
	// ErrFingerprintTooLong indicates a fingerprint is too long for the matcher.
	ErrFingerprintTooLong = errors.New("fingerprint too long")
)

// Segment represents a matching segment between two fingerprints.
type Segment struct {
	Pos1       int
	Pos2       int
	Duration   int
	Score      float64
	LeftScore  float64
	RightScore float64
}

// PublicScore returns the match score percentage as an integer (0..100).
func (s Segment) PublicScore() int {
	return int(s.Score*100.0 + 0.5)
}

// Merged returns a merged segment with another adjacent segment.
func (s Segment) Merged(other Segment) Segment {
	newDuration := s.Duration + other.Duration
	newScore := (s.Score*float64(s.Duration) + other.Score*float64(other.Duration)) / float64(newDuration)
	return Segment{
		Pos1:       s.Pos1,
		Pos2:       s.Pos2,
		Duration:   newDuration,
		Score:      newScore,
		LeftScore:  s.Score,
		RightScore: other.Score,
	}
}

// Matcher compares two raw fingerprints to find matching segments and calculate similarity.
type Matcher struct {
	config         classifier.Config
	matchThreshold float64
}

// NewMatcher creates a new fingerprint matcher with the given configuration.
func NewMatcher(config classifier.Config) *Matcher {
	return &Matcher{
		config:         config,
		matchThreshold: defaultMatchThreshold,
	}
}

// SetMatchThreshold sets the maximum score threshold for a match.
func (m *Matcher) SetMatchThreshold(t float64) {
	m.matchThreshold = t
}

// MatchThreshold returns the current match threshold.
func (m *Matcher) MatchThreshold() float64 {
	return m.matchThreshold
}

// HashTime returns the time in seconds corresponding to subfingerprint index i.
func (m *Matcher) HashTime(i int) float64 {
	return m.config.ItemDurationSeconds() * float64(i)
}

// HashDuration returns the total duration in seconds up to subfingerprint index i including buffer delay.
func (m *Matcher) HashDuration(i int) float64 {
	return m.HashTime(i) + m.config.DelaySeconds()
}

type alignment struct {
	count uint32
	index int
}

// Match compares two fingerprints and returns matching segments.
func (m *Matcher) Match(fp1, fp2 []uint32) ([]Segment, error) {
	fp1Size := len(fp1)
	fp2Size := len(fp2)

	const hashShift = 32 - alignBits
	const hashMask = uint32(((1 << alignBits) - 1) << hashShift)
	const offsetMask = uint32((1 << (32 - alignBits - 1)) - 1)
	const sourceMask = uint32(1 << (32 - alignBits - 1))

	if uint32(fp1Size+1) >= offsetMask {
		return nil, ErrFingerprintTooLong
	}
	if uint32(fp2Size+1) >= offsetMask {
		return nil, ErrFingerprintTooLong
	}

	if fp1Size == 0 || fp2Size == 0 {
		return []Segment{}, nil
	}

	offsets := make([]uint32, 0, fp1Size+fp2Size)
	for i := 0; i < fp1Size; i++ {
		key := ((fp1[i] >> (32 - alignBits)) << hashShift) | uint32(i)&offsetMask
		offsets = append(offsets, key)
	}
	for i := 0; i < fp2Size; i++ {
		key := ((fp2[i] >> (32 - alignBits)) << hashShift) | (uint32(i) & offsetMask) | sourceMask
		offsets = append(offsets, key)
	}

	sort.Slice(offsets, func(i, j int) bool {
		return offsets[i] < offsets[j]
	})

	histogram := make([]uint32, fp1Size+fp2Size)
	for it := 0; it < len(offsets); it++ {
		hash := offsets[it] & hashMask
		offset1 := offsets[it] & offsetMask
		source1 := offsets[it] & sourceMask
		if source1 != 0 {
			continue
		}

		for it2 := it + 1; it2 < len(offsets); it2++ {
			hash2 := offsets[it2] & hashMask
			if hash != hash2 {
				break
			}
			offset2 := offsets[it2] & offsetMask
			source2 := offsets[it2] & sourceMask
			if source2 != 0 {
				offsetDiff := int(offset1) + fp2Size - int(offset2)
				if offsetDiff >= 0 && offsetDiff < len(histogram) {
					histogram[offsetDiff]++
				}
			}
		}
	}

	var bestAlignments []alignment
	histLen := len(histogram)
	for i := 0; i < histLen; i++ {
		count := histogram[i]
		if count > 1 {
			isPeakLeft := (i == 0) || (histogram[i-1] <= count)
			isPeakRight := (i == histLen-1) || (histogram[i+1] <= count)
			if isPeakLeft && isPeakRight {
				bestAlignments = append(bestAlignments, alignment{count: count, index: i})
			}
		}
	}

	if len(bestAlignments) == 0 {
		return []Segment{}, nil
	}

	sort.Slice(bestAlignments, func(i, j int) bool {
		if bestAlignments[i].count == bestAlignments[j].count {
			return bestAlignments[i].index < bestAlignments[j].index
		}
		return bestAlignments[i].count > bestAlignments[j].count
	})

	var segments []Segment

	for _, item := range bestAlignments {
		offsetDiff := item.index - fp2Size

		offset1 := 0
		if offsetDiff > 0 {
			offset1 = offsetDiff
		}
		offset2 := 0
		if offsetDiff < 0 {
			offset2 = -offsetDiff
		}

		size := fp1Size - offset1
		if fp2Size-offset2 < size {
			size = fp2Size - offset2
		}
		if size <= 0 {
			break
		}

		bitCounts := make([]float64, size)
		for i := 0; i < size; i++ {
			bitCounts[i] = float64(HammingDistance(fp1[offset1+i], fp2[offset2+i]))
		}

		origBitCounts := append([]float64(nil), bitCounts...)
		smoothedBitCounts := filter.GaussianFilter(bitCounts, 8.0, 3)
		grad := filter.Gradient(smoothedBitCounts)
		for i := range grad {
			grad[i] = math.Abs(grad[i])
		}

		var gradientPeaks []int
		for i := 0; i < size; i++ {
			gi := grad[i]
			if i > 0 && i < size-1 && gi > 0.15 && gi >= grad[i-1] && gi >= grad[i+1] {
				if len(gradientPeaks) == 0 || gradientPeaks[len(gradientPeaks)-1]+1 < i {
					gradientPeaks = append(gradientPeaks, i)
				}
			}
		}
		gradientPeaks = append(gradientPeaks, size)

		begin := 0
		for _, end := range gradientPeaks {
			duration := end - begin
			if duration <= 0 {
				continue
			}

			var sum float64
			for j := begin; j < end; j++ {
				sum += origBitCounts[j]
			}
			score := sum / float64(duration)

			if score < m.matchThreshold {
				added := false
				if len(segments) > 0 {
					last := &segments[len(segments)-1]
					if math.Abs(last.Score-score) < 0.7 {
						*last = last.Merged(Segment{
							Pos1:       offset1 + begin,
							Pos2:       offset2 + begin,
							Duration:   duration,
							Score:      score,
							LeftScore:  score,
							RightScore: score,
						})
						added = true
					}
				}
				if !added {
					segments = append(segments, Segment{
						Pos1:       offset1 + begin,
						Pos2:       offset2 + begin,
						Duration:   duration,
						Score:      score,
						LeftScore:  score,
						RightScore: score,
					})
				}
			}
			begin = end
		}

		break
	}

	return segments, nil
}
