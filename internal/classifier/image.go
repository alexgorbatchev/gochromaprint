package classifier

// RollingIntegralImage maintains a rolling integral image of features over time.
type RollingIntegralImage struct {
	maxRows    int
	numColumns int
	numRows    int
	data       []float64
}

// NewRollingIntegralImage creates a rolling integral image with capacity for maxRows.
func NewRollingIntegralImage(maxRows int) *RollingIntegralImage {
	return &RollingIntegralImage{
		maxRows: maxRows + 1,
	}
}

// NumColumns returns the number of columns in the image.
func (img *RollingIntegralImage) NumColumns() int {
	return img.numColumns
}

// NumRows returns the total number of rows added to the image.
func (img *RollingIntegralImage) NumRows() int {
	return img.numRows
}

// Reset clears the integral image state.
func (img *RollingIntegralImage) Reset() {
	img.data = img.data[:0]
	img.numRows = 0
	img.numColumns = 0
}

func (img *RollingIntegralImage) getRowOffset(r int) int {
	return (r % img.maxRows) * img.numColumns
}

// AddRow adds a new row of features to the integral image.
func (img *RollingIntegralImage) AddRow(row []float64) {
	size := len(row)
	if img.numColumns == 0 {
		img.numColumns = size
		img.data = make([]float64, img.maxRows*img.numColumns)
	}

	currOffset := img.getRowOffset(img.numRows)

	// Partial sum of current row
	var sum float64
	for i := 0; i < size; i++ {
		sum += row[i]
		img.data[currOffset+i] = sum
	}

	// Add previous row integral sum if not the first row
	if img.numRows > 0 {
		lastOffset := img.getRowOffset(img.numRows - 1)
		for i := 0; i < size; i++ {
			img.data[currOffset+i] += img.data[lastOffset+i]
		}
	}

	img.numRows++
}

// Area calculates the sum of the integral image in rectangular region [r1, r2) x [c1, c2).
func (img *RollingIntegralImage) Area(r1, c1, r2, c2 int) float64 {
	if r1 == r2 || c1 == c2 {
		return 0.0
	}

	if r1 == 0 {
		row2Offset := img.getRowOffset(r2 - 1)
		if c1 == 0 {
			return img.data[row2Offset+c2-1]
		}
		return img.data[row2Offset+c2-1] - img.data[row2Offset+c1-1]
	}

	row1Offset := img.getRowOffset(r1 - 1)
	row2Offset := img.getRowOffset(r2 - 1)
	if c1 == 0 {
		return img.data[row2Offset+c2-1] - img.data[row1Offset+c2-1]
	}
	return img.data[row2Offset+c2-1] - img.data[row1Offset+c2-1] - img.data[row2Offset+c1-1] + img.data[row1Offset+c1-1]
}

// Image represents a simple 2D matrix of floating point values.
type Image struct {
	Columns int
	Data    []float64
}

// NewImage creates an image with given columns.
func NewImage(columns int) *Image {
	return &Image{
		Columns: columns,
	}
}

// NumRows returns the number of rows in the image.
func (img *Image) NumRows() int {
	if img.Columns == 0 {
		return 0
	}
	return len(img.Data) / img.Columns
}

// AddRow appends a row of features to the image.
func (img *Image) AddRow(row []float64) {
	img.Data = append(img.Data, row...)
}

// Row returns the slice of values at row index i.
func (img *Image) Row(i int) []float64 {
	offset := i * img.Columns
	return img.Data[offset : offset+img.Columns]
}
