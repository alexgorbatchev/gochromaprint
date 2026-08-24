package chroma_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/alexgorbatchev/gochromaprint/internal/chroma"
)

type vectorCollector struct {
	vectors [][]float64
}

func (c *vectorCollector) Consume(features []float64) {
	v := make([]float64, len(features))
	copy(v, features)
	c.vectors = append(c.vectors, v)
}

func TestChromaNormalA(t *testing.T) {
	col := &vectorCollector{}
	e := chroma.NewExtractor(10, 510, 256, 1000, col)
	if e.Interpolate() {
		t.Fatal("expected Interpolate() to be false initially")
	}
	e.Reset()
	frame := make([]float64, 128)
	frame[113] = 1.0
	e.Consume(frame)

	if len(col.vectors) != 1 {
		t.Fatalf("got %d vectors, want 1", len(col.vectors))
	}
	expected := [12]float64{1.0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for i := 0; i < 12; i++ {
		if math.Abs(col.vectors[0][i]-expected[i]) > 0.0001 {
			t.Fatalf("index %d: got %g, want %g", i, col.vectors[0][i], expected[i])
		}
	}
}

func TestChromaNormalGSharp(t *testing.T) {
	col := &vectorCollector{}
	e := chroma.NewExtractor(10, 510, 256, 1000, col)
	frame := make([]float64, 128)
	frame[112] = 1.0
	e.Consume(frame)

	if len(col.vectors) != 1 {
		t.Fatalf("got %d vectors, want 1", len(col.vectors))
	}
	expected := [12]float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1.0}
	for i := 0; i < 12; i++ {
		if math.Abs(col.vectors[0][i]-expected[i]) > 0.0001 {
			t.Fatalf("index %d: got %g, want %g", i, col.vectors[0][i], expected[i])
		}
	}
}

func TestChromaNormalB(t *testing.T) {
	col := &vectorCollector{}
	e := chroma.NewExtractor(10, 510, 256, 1000, col)
	frame := make([]float64, 128)
	frame[64] = 1.0
	e.Consume(frame)

	if len(col.vectors) != 1 {
		t.Fatalf("got %d vectors, want 1", len(col.vectors))
	}
	expected := [12]float64{0, 0, 1.0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for i := 0; i < 12; i++ {
		if math.Abs(col.vectors[0][i]-expected[i]) > 0.0001 {
			t.Fatalf("index %d: got %g, want %g", i, col.vectors[0][i], expected[i])
		}
	}
}

func TestChromaInterpolatedB(t *testing.T) {
	col := &vectorCollector{}
	e := chroma.NewExtractor(10, 510, 256, 1000, col)
	e.SetInterpolate(true)
	if !e.Interpolate() {
		t.Fatal("expected Interpolate() to be true")
	}
	frame := make([]float64, 128)
	frame[64] = 1.0
	e.Consume(frame)

	if len(col.vectors) != 1 {
		t.Fatalf("got %d vectors, want 1", len(col.vectors))
	}
	expected := [12]float64{0, 0.286905, 0.713095, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for i := 0; i < 12; i++ {
		if math.Abs(col.vectors[0][i]-expected[i]) > 0.0001 {
			t.Fatalf("index %d: got %g, want %g", i, col.vectors[0][i], expected[i])
		}
	}
}

func TestChromaInterpolatedA(t *testing.T) {
	col := &vectorCollector{}
	e := chroma.NewExtractor(10, 510, 256, 1000, col)
	e.SetInterpolate(true)
	frame := make([]float64, 128)
	frame[113] = 1.0
	e.Consume(frame)

	if len(col.vectors) != 1 {
		t.Fatalf("got %d vectors, want 1", len(col.vectors))
	}
	expected := [12]float64{0.555242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.444758}
	for i := 0; i < 12; i++ {
		if math.Abs(col.vectors[0][i]-expected[i]) > 0.0001 {
			t.Fatalf("index %d: got %g, want %g", i, col.vectors[0][i], expected[i])
		}
	}
}

func TestChromaInterpolatedGSharp(t *testing.T) {
	col := &vectorCollector{}
	e := chroma.NewExtractor(10, 510, 256, 1000, col)
	e.SetInterpolate(true)
	frame := make([]float64, 128)
	frame[112] = 1.0
	e.Consume(frame)

	if len(col.vectors) != 1 {
		t.Fatalf("got %d vectors, want 1", len(col.vectors))
	}
	expected := [12]float64{0.401354, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.598646}
	for i := 0; i < 12; i++ {
		if math.Abs(col.vectors[0][i]-expected[i]) > 0.0001 {
			t.Fatalf("index %d: got %g, want %g", i, col.vectors[0][i], expected[i])
		}
	}
}

func TestChromaConsumerFunc(t *testing.T) {
	consumed := false
	fn := chroma.ConsumerFunc(func(features []float64) {
		consumed = true
	})
	fn.Consume([]float64{1, 2, 3})
	if !consumed {
		t.Fatal("expected ConsumerFunc to be called")
	}
}

func TestChromaFilterBlur2(t *testing.T) {
	col := &vectorCollector{}
	filter := chroma.NewFilter([]float64{0.5, 0.5}, col)
	filter.Reset()
	d1 := []float64{0.0, 5.0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	d2 := []float64{1.0, 6.0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	d3 := []float64{2.0, 7.0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	filter.Consume(d1)
	filter.Consume(d2)
	filter.Consume(d3)

	if len(col.vectors) != 2 {
		t.Fatalf("got %d rows, want 2", len(col.vectors))
	}
	if math.Abs(col.vectors[0][0]-0.5) > 1e-6 || math.Abs(col.vectors[1][0]-1.5) > 1e-6 {
		t.Fatalf("col 0 values mismatch: %v", col.vectors)
	}
	if math.Abs(col.vectors[0][1]-5.5) > 1e-6 || math.Abs(col.vectors[1][1]-6.5) > 1e-6 {
		t.Fatalf("col 1 values mismatch: %v", col.vectors)
	}
}

func TestChromaFilterBlur3(t *testing.T) {
	col := &vectorCollector{}
	filter := chroma.NewFilter([]float64{0.5, 0.7, 0.5}, col)
	d1 := []float64{0.0, 5.0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	d2 := []float64{1.0, 6.0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	d3 := []float64{2.0, 7.0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	d4 := []float64{3.0, 8.0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	filter.Consume(d1)
	filter.Consume(d2)
	filter.Consume(d3)
	filter.Consume(d4)

	if len(col.vectors) != 2 {
		t.Fatalf("got %d rows, want 2", len(col.vectors))
	}
	if math.Abs(col.vectors[0][0]-1.7) > 1e-6 || math.Abs(col.vectors[1][0]-3.4) > 1e-6 {
		t.Fatalf("col 0 values mismatch: %v", col.vectors)
	}
	if math.Abs(col.vectors[0][1]-10.2) > 1e-6 || math.Abs(col.vectors[1][1]-11.9) > 1e-6 {
		t.Fatalf("col 1 values mismatch: %v", col.vectors)
	}
}

func TestChromaFilterLargeKernel(t *testing.T) {
	col := &vectorCollector{}
	kernel := make([]float64, 10)
	for i := range kernel {
		kernel[i] = 0.1
	}
	filter := chroma.NewFilter(kernel, col)
	for i := 0; i < 12; i++ {
		row := make([]float64, 12)
		row[0] = float64(i)
		filter.Consume(row)
	}
	if len(col.vectors) != 3 {
		t.Fatalf("got %d rows, want 3", len(col.vectors))
	}
}

func TestChromaResampler(t *testing.T) {
	col := &vectorCollector{}
	resampler := chroma.NewResampler(2, col)

	v1 := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	v2 := []float64{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}
	resampler.Consume(v1)
	if len(col.vectors) != 0 {
		t.Fatalf("expected 0 vectors after 1 consume, got %d", len(col.vectors))
	}
	resampler.Consume(v2)
	if len(col.vectors) != 1 {
		t.Fatalf("expected 1 vector after 2 consumes, got %d", len(col.vectors))
	}
	expected := []float64{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	if !reflect.DeepEqual(col.vectors[0], expected) {
		t.Fatalf("got %v, want %v", col.vectors[0], expected)
	}
}

func TestChromaNormalizer(t *testing.T) {
	col := &vectorCollector{}
	norm := chroma.NewNormalizer(col)
	norm.Reset()

	// Below threshold (norm < 0.01)
	norm.Consume([]float64{0.001, 0.001, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	for _, v := range col.vectors[0] {
		if v != 0 {
			t.Fatalf("expected 0, got %g", v)
		}
	}

	// Above threshold (3, 4 -> norm = 5)
	col.vectors = col.vectors[:0]
	norm.Consume([]float64{3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	if math.Abs(col.vectors[0][0]-0.6) > 1e-6 || math.Abs(col.vectors[0][1]-0.8) > 1e-6 {
		t.Fatalf("expected 0.6, 0.8, got %v", col.vectors[0])
	}
}
