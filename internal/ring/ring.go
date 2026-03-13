package ring

import "math"

type FloatRing struct {
	vals []float64
	idx  int
	full bool
}

func NewFloatRing(size int) *FloatRing {
	if size <= 0 {
		size = 1
	}
	return &FloatRing{
		vals: make([]float64, size),
	}
}

func (r *FloatRing) Add(v float64) {
	r.vals[r.idx] = v
	r.idx = (r.idx + 1) % len(r.vals)
	if r.idx == 0 {
		r.full = true
	}
}

func (r *FloatRing) Len() int {
	if r.full {
		return len(r.vals)
	}
	return r.idx
}

func (r *FloatRing) Cap() int {
	return len(r.vals)
}

func (r *FloatRing) Values() []float64 {
	if !r.full {
		out := make([]float64, r.idx)
		copy(out, r.vals[:r.idx])
		return out
	}

	out := make([]float64, len(r.vals))
	copy(out, r.vals[r.idx:])
	copy(out[len(r.vals)-r.idx:], r.vals[:r.idx])
	return out
}

func (r *FloatRing) Last() (float64, bool) {
	if !r.full && r.idx == 0 {
		return 0, false
	}
	p := r.idx - 1
	if p < 0 {
		p = len(r.vals) - 1
	}
	return r.vals[p], true
}

func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

func Std(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := Mean(xs)
	ss := 0.0
	for _, x := range xs {
		d := x - m
		ss += d * d
	}
	return math.Sqrt(ss / float64(len(xs)))
}

func SumAbs(xs []float64) float64 {
	sum := 0.0
	for _, x := range xs {
		if x < 0 {
			sum -= x
		} else {
			sum += x
		}
	}
	return sum
}
