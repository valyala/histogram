// Package histogram provides building blocks for fast histograms.
package histogram

import (
	"math"
	"sort"
	"sync"

	"github.com/valyala/fastrand"
)

var (
	infNeg = math.Inf(-1)
	infPos = math.Inf(1)
	nan    = math.NaN()
)

// Fast is a fast histogram.
//
// It cannot be used from concurrently running goroutines without
// external synchronization.
type Fast struct {
	max   float64
	min   float64
	count uint64

	a   []float64
	tmp []float64
	rng fastrand.RNG
}

// NewFast returns new fast histogram.
func NewFast() *Fast {
	f := &Fast{}
	f.Reset()
	return f
}

// Reset resets the histogram.
func (f *Fast) Reset() {
	f.max = infNeg
	f.min = infPos
	f.count = 0
	if len(f.a) > 0 {
		f.a = f.a[:0]
		f.tmp = f.tmp[:0]
	} else {
		// Free up memory occupied by unused histogram.
		f.a = nil
		f.tmp = nil
	}
	// Reset rng state in order to get repeatable results
	// for the same sequence of values passed to Fast.Update.
	// See https://github.com/VictoriaMetrics/VictoriaMetrics/issues/1612
	f.rng.Seed(1)
}

// Update updates the f with v.
func (f *Fast) Update(v float64) {
	if v > f.max {
		f.max = v
	}
	if v < f.min {
		f.min = v
	}

	f.count++
	if len(f.a) < maxSamples {
		f.a = append(f.a, v)
		return
	}
	if n := int(f.rng.Uint32n(uint32(f.count))); n < len(f.a) {
		f.a[n] = v
	}
}

const maxSamples = 1000

// Quantile returns the quantile value for the given phi.
func (f *Fast) Quantile(phi float64) float64 {
	f.tmp = append(f.tmp[:0], f.a...)
	sort.Float64s(f.tmp)
	return f.quantile(phi)
}

// Quantiles appends quantile values to dst for the given phis.
func (f *Fast) Quantiles(dst, phis []float64) []float64 {
	f.tmp = append(f.tmp[:0], f.a...)
	sort.Float64s(f.tmp)
	return f.quantiles(dst, phis)
}

func (f *Fast) quantiles(dst, phis []float64) []float64 {
	for _, phi := range phis {
		q := f.quantile(phi)
		dst = append(dst, q)
	}
	return dst
}

func (f *Fast) quantile(phi float64) float64 {
	if len(f.tmp) == 0 || math.IsNaN(phi) {
		return nan
	}
	if phi <= 0 {
		return f.min
	}
	if phi >= 1 {
		return f.max
	}
	idx := uint(phi*float64(len(f.tmp)-1) + 0.5)
	if idx >= uint(len(f.tmp)) {
		idx = uint(len(f.tmp) - 1)
	}
	return f.tmp[idx]
}

// GetFast returns a histogram from a pool.
func GetFast() *Fast {
	v := fastPool.Get()
	if v == nil {
		return NewFast()
	}
	return v.(*Fast)
}

// PutFast puts hf to the pool.
//
// hf cannot be used after this call.
func PutFast(f *Fast) {
	f.Reset()
	fastPool.Put(f)
}

var fastPool sync.Pool

func Quantile(fs []*Fast, phi float64) float64 {
	t := combine(fs)
	return t.quantile(phi)
}

func Quantiles(fs []*Fast, dst, phis []float64) []float64 {
	t := combine(fs)
	return t.quantiles(dst, phis)
}

func combine(fs []*Fast) Fast {
	n := 0
	for _, f := range fs {
		n += len(f.a)
	}

	var t Fast
	t.Reset()
	t.tmp = make([]float64, 0, n)

	for _, f := range fs {
		t.tmp = append(t.tmp, f.a...)
		if t.max < f.max {
			t.max = f.max
		}
		if t.min > f.min {
			t.min = f.min
		}
	}
	sort.Float64s(t.tmp)

	return t
}
