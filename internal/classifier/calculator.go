package classifier

var grayCodes = [4]uint32{0, 1, 3, 2}

// FingerprintCalculator generates 32-bit subfingerprints from normalized chroma features.
type FingerprintCalculator struct {
	classifiers    []Classifier
	maxFilterWidth int
	image          *RollingIntegralImage
	fingerprint    []uint32
}

// NewFingerprintCalculator creates a new calculator for given classifiers.
func NewFingerprintCalculator(classifiers []Classifier) *FingerprintCalculator {
	maxWidth := 0
	for _, c := range classifiers {
		if c.Filter.Width > maxWidth {
			maxWidth = c.Filter.Width
		}
	}

	return &FingerprintCalculator{
		classifiers:    classifiers,
		maxFilterWidth: maxWidth,
		image:          NewRollingIntegralImage(256),
	}
}

// Reset resets the calculator state and clears internal buffers.
func (fc *FingerprintCalculator) Reset() {
	fc.image.Reset()
	fc.fingerprint = fc.fingerprint[:0]
}

// ClearFingerprint clears the accumulated subfingerprints but keeps rolling image state.
func (fc *FingerprintCalculator) ClearFingerprint() {
	fc.fingerprint = fc.fingerprint[:0]
}

// Fingerprint returns the generated raw subfingerprints.
func (fc *FingerprintCalculator) Fingerprint() []uint32 {
	return fc.fingerprint
}

func (fc *FingerprintCalculator) calculateSubfingerprint(offset int) uint32 {
	var bits uint32
	for _, c := range fc.classifiers {
		q := c.Classify(fc.image, offset)
		bits = (bits << 2) | grayCodes[q]
	}
	return bits
}

// Consume receives a normalized 12-element chroma feature vector and updates the fingerprint.
func (fc *FingerprintCalculator) Consume(features []float64) {
	fc.image.AddRow(features)
	if fc.image.NumRows() >= fc.maxFilterWidth {
		sub := fc.calculateSubfingerprint(fc.image.NumRows() - fc.maxFilterWidth)
		fc.fingerprint = append(fc.fingerprint, sub)
	}
}
