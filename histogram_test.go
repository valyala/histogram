package histogram

import (
	"math"
	"testing"
)

func TestFastUnderflow(t *testing.T) {
	f := GetFast()
	defer PutFast(f)

	q := f.Quantile(0.5)
	if !math.IsNaN(q) {
		t.Fatalf("unexpected quantile for empty histogram; got %v; want %v", q, nan)
	}

	for i := 0; i < maxSamples; i++ {
		f.Update(float64(i))
	}
	qs := f.Quantiles(nil, []float64{0, 0.5, 1})
	if qs[0] != 0 {
		t.Fatalf("unexpected quantile value for phi=0; got %v; want %v", qs[0], 0)
	}
	if qs[1] != maxSamples/2 {
		t.Fatalf("unexpected quantile value for phi=0.5; got %v; want %v", qs[1], maxSamples/2)
	}
	if qs[2] != maxSamples-1 {
		t.Fatalf("unexpected quantile value for phi=1; got %v; want %v", qs[2], maxSamples-1)
	}
}

func TestFastOverflow(t *testing.T) {
	f := GetFast()
	defer PutFast(f)

	for i := 0; i < maxSamples*10; i++ {
		f.Update(float64(i))
	}
	qs := f.Quantiles(nil, []float64{0, 0.5, 0.9999, 1})
	if qs[0] != 0 {
		t.Fatalf("unexpected quantile value for phi=0; got %v; want %v", qs[0], 0)
	}

	median := float64(maxSamples*10-1) / 2
	if qs[1] < median*0.9 || qs[1] > median*1.1 {
		t.Fatalf("unexpected quantile value for phi=0.5; got %v; want %v", qs[1], median)
	}
	if qs[2] < maxSamples*10*0.9 {
		t.Fatalf("unexpected quantile value for phi=0.9999; got %v; want %v", qs[2], maxSamples*10*0.9)
	}
	if qs[3] != maxSamples*10-1 {
		t.Fatalf("unexpected quantile value for phi=1; got %v; want %v", qs[3], maxSamples*10-1)
	}

	q := f.Quantile(nan)
	if !math.IsNaN(q) {
		t.Fatalf("unexpected value for phi=NaN; got %v; want %v", q, nan)
	}
}

func TestFastRepeatableResults(t *testing.T) {
	f := GetFast()
	defer PutFast(f)

	for i := 0; i < maxSamples*10; i++ {
		f.Update(float64(i))
	}
	q1 := f.Quantile(0.95)

	for j := 0; j < 10; j++ {
		f.Reset()
		for i := 0; i < maxSamples*10; i++ {
			f.Update(float64(i))
		}
		q2 := f.Quantile(0.95)
		if q2 != q1 {
			t.Fatalf("unexpected quantile value; got %g; want %g", q2, q1)
		}
	}
}

func TestCombine(t *testing.T) {
	f1 := GetFast()
	defer PutFast(f1)

	f2 := GetFast()
	defer PutFast(f2)

	for i := 0; i < 10000; i++ {
		f1.Update(float64(i))
	}
	for i := 10000; i < 20000; i++ {
		f2.Update(float64(i))
	}

	q50 := Quantile([]*Fast{f1, f2}, 0.5)
	if q50 < 9000 || q50 > 11000 {
		t.Fatal(q50)
	}
	qs := Quantiles([]*Fast{f1, f2}, nil, []float64{0, 0.5, 1})
	if len(qs) != 3 {
		t.Fatal(len(qs))
	}
	if qs[0] != 0 || qs[1] != q50 || qs[2] != 19999 {
		t.Fatal(qs)
	}
}
