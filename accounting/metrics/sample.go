package metrics

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const rescaleThreshold = time.Hour

// Samples maintain a statistically-significant selection of values from
// a stream.
type Sample interface {
	Clear()
	Count() int64
	Max() int64
	Mean() float64
	Min() int64
	Percentile(float64) float64
	Percentiles([]float64) []float64
	Size() int
	StdDev() float64
	Sum() int64
	Update(int64)
	Values() []int64
	Variance() float64
}

// ExpDecaySample is an exponentially-decaying sample using a forward-decaying
// priority reservoir.  See Cormode et al's "Forward Decay: A Practical Time
// Decay Model for Streaming Systems".
//
// <http://www.research.att.com/people/Cormode_Graham/library/publications/CormodeShkapenyukSrivastavaXu09.pdf>
type samples struct {
	mutex         sync.RWMutex
	reservoirSize int
	random        *rand.Rand
	values        expDecaySampleHeap
}

const _RESERVOIRS = 8

type ExpDecaySample struct {
	alpha         float64
	count         int64
	next          uint64
	mutex         sync.RWMutex
	reservoirSize int
	t0, t1        time.Time
	values        expDecaySampleHeap
	reservoirs    [_RESERVOIRS]samples
}

// NewExpDecaySample constructs a new exponentially-decaying sample with the
// given reservoir size and alpha.
func NewExpDecaySample(reservoirSize int, alpha float64) Sample {
	rSz := reservoirSize / _RESERVOIRS
	if rSz*_RESERVOIRS != reservoirSize {
		rSz++
		reservoirSize = rSz * _RESERVOIRS
	}
	s := &ExpDecaySample{
		alpha:         alpha,
		reservoirSize: reservoirSize,
		t0:            time.Now(),
	}
	s.t1 = s.t0.Add(rescaleThreshold)

	s.values.s = make([]expDecaySample, reservoirSize, reservoirSize)
	s.values.Clear()
	start := 0
	for r, _ := range s.reservoirs {
		s.reservoirs[r].values.s = s.values.s[start : start : start+rSz]
		s.reservoirs[r].random = rand.New(rand.NewSource(s.t0.UnixNano() + int64(r)))
		s.reservoirs[r].reservoirSize = rSz
		start += rSz
	}
	return s
}

// Clear clears all samples.
func (s *ExpDecaySample) Clear() {
	s.mutex.Lock()
	s.count = 0
	s.t0 = time.Now()
	s.t1 = s.t0.Add(rescaleThreshold)
	s.values.Clear()
	s.mutex.Unlock()
}

// Count returns the number of samples recorded, which may exceed the
// reservoir size.
func (s *ExpDecaySample) Count() int64 {
	return atomic.LoadInt64(&s.count)
}

// Max returns the maximum value in the sample, which may not be the maximum
// value ever to be part of the sample.
func (s *ExpDecaySample) Max() int64 {
	return SampleMax(s.Values())
}

// Mean returns the mean of the values in the sample.
func (s *ExpDecaySample) Mean() float64 {
	return SampleMean(s.Values())
}

// Min returns the minimum value in the sample, which may not be the minimum
// value ever to be part of the sample.
func (s *ExpDecaySample) Min() int64 {
	return SampleMin(s.Values())
}

// Percentile returns an arbitrary percentile of values in the sample.
func (s *ExpDecaySample) Percentile(p float64) float64 {
	return SamplePercentile(s.Values(), p)
}

// Percentiles returns a slice of arbitrary percentiles of values in the
// sample.
func (s *ExpDecaySample) Percentiles(ps []float64) []float64 {
	return SamplePercentiles(s.Values(), ps)
}

// Size returns the size of the sample, which is at most the reservoir size.
func (s *ExpDecaySample) Size() int {
	sz := 0
	for r, _ := range s.reservoirs {
		s.reservoirs[r].mutex.RLock()
		sz += len(s.reservoirs[r].values.s)
		s.reservoirs[r].mutex.RUnlock()
	}
	return sz
}

// StdDev returns the standard deviation of the values in the sample.
func (s *ExpDecaySample) StdDev() float64 {
	return SampleStdDev(s.Values())
}

// Sum returns the sum of the values in the sample.
func (s *ExpDecaySample) Sum() int64 {
	return SampleSum(s.Values())
}

// Update samples a new value.
func (s *ExpDecaySample) Update(v int64) {
	s.update(time.Now(), v)
}

// Values returns a copy of the values in the sample.
func (s *ExpDecaySample) Values() []int64 {
	s.mutex.RLock()
	vals := s.values.Values()
	values := make([]int64, 0, cap(vals))
	for _, v := range vals {
		if v.inUse {
			values = append(values, v.v)
		}
	}
	s.mutex.RUnlock()
	return values
}

// Variance returns the variance of the values in the sample.
func (s *ExpDecaySample) Variance() float64 {
	return SampleVariance(s.Values())
}

// update samples a new value at a particular timestamp.  This is a method all
// its own to facilitate testing.
func (s *ExpDecaySample) update(t time.Time, v int64) {

	// no clearing for now
	s.mutex.RLock()
	rescale := t.After(s.t1)
	atomic.AddInt64(&s.count, 1)

	// choose and amend reservoir
	next := atomic.AddUint64(&s.next, 1) % _RESERVOIRS
	s.reservoirs[next].mutex.Lock()
	ed := expDecaySample{
		k:     math.Exp(t.Sub(s.t0).Seconds()*s.alpha) / s.reservoirs[next].random.Float64(),
		v:     v,
		inUse: true,
	}
	if s.reservoirs[next].values.Size() == s.reservoirs[next].reservoirSize {
		s.reservoirs[next].values.Pop()
	}
	s.reservoirs[next].values.Push(ed)
	s.reservoirs[next].mutex.Unlock()
	s.mutex.RUnlock()

	// numbers getting high - choose a new landmark
	if rescale {
		s.mutex.Lock()
		t0 := s.t0
		s.t0 = t
		s.t1 = s.t0.Add(rescaleThreshold)
		newLandmark := math.Exp(-s.alpha * s.t0.Sub(t0).Seconds())
		for r, _ := range s.values.s {
			if s.values.s[r].inUse {
				s.values.s[r].k = s.values.s[r].k * newLandmark
			}
		}
		s.mutex.Unlock()
	}
}

// SampleMax returns the maximum value of the slice of int64.
func SampleMax(values []int64) int64 {
	if 0 == len(values) {
		return 0
	}
	var max int64 = math.MinInt64
	for _, v := range values {
		if max < v {
			max = v
		}
	}
	return max
}

// SampleMean returns the mean value of the slice of int64.
func SampleMean(values []int64) float64 {
	if 0 == len(values) {
		return 0.0
	}
	return float64(SampleSum(values)) / float64(len(values))
}

// SampleMin returns the minimum value of the slice of int64.
func SampleMin(values []int64) int64 {
	if 0 == len(values) {
		return 0
	}
	var min int64 = math.MaxInt64
	for _, v := range values {
		if min > v {
			min = v
		}
	}
	return min
}

// SamplePercentiles returns an arbitrary percentile of the slice of int64.
func SamplePercentile(values int64Slice, p float64) float64 {
	return SamplePercentiles(values, []float64{p})[0]
}

// SamplePercentiles returns a slice of arbitrary percentiles of the slice of
// int64.
func SamplePercentiles(values int64Slice, ps []float64) []float64 {
	scores := make([]float64, len(ps))
	size := len(values)
	if size > 0 {
		sort.Sort(values)
		for i, p := range ps {
			pos := p * float64(size+1)
			if pos < 1.0 {
				scores[i] = float64(values[0])
			} else if pos >= float64(size) {
				scores[i] = float64(values[size-1])
			} else {
				lower := float64(values[int(pos)-1])
				upper := float64(values[int(pos)])
				scores[i] = lower + (pos-math.Floor(pos))*(upper-lower)
			}
		}
	}
	return scores
}

// SampleStdDev returns the standard deviation of the slice of int64.
func SampleStdDev(values []int64) float64 {
	return math.Sqrt(SampleVariance(values))
}

// SampleSum returns the sum of the slice of int64.
func SampleSum(values []int64) int64 {
	var sum int64
	for _, v := range values {
		sum += v
	}
	return sum
}

// SampleVariance returns the variance of the slice of int64.
func SampleVariance(values []int64) float64 {
	if 0 == len(values) {
		return 0.0
	}
	m := SampleMean(values)
	var sum float64
	for _, v := range values {
		d := float64(v) - m
		sum += d * d
	}
	return sum / float64(len(values))
}

// expDecaySample represents an individual sample in a heap.
type expDecaySample struct {
	k     float64
	v     int64
	inUse bool
}

// expDecaySampleHeap is a min-heap of expDecaySamples.
// The internal implementation is copied from the standard library's container/heap
type expDecaySampleHeap struct {
	s []expDecaySample
}

func (h *expDecaySampleHeap) Clear() {
	for r := 0; r < cap(h.s); r++ {
		h.s[r].inUse = false
	}
}

func (h *expDecaySampleHeap) Push(s expDecaySample) {
	n := len(h.s)
	h.s = h.s[0 : n+1]
	h.s[n] = s
	h.up(n)
}

func (h *expDecaySampleHeap) Pop() expDecaySample {
	n := len(h.s) - 1
	h.s[0], h.s[n] = h.s[n], h.s[0]
	h.down(0, n)

	n = len(h.s)
	s := h.s[n-1]
	h.s[n-1].inUse = false
	h.s = h.s[0 : n-1]
	return s
}

func (h *expDecaySampleHeap) Size() int {
	return len(h.s)
}

func (h *expDecaySampleHeap) Values() []expDecaySample {
	return h.s
}

func (h *expDecaySampleHeap) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !(h.s[j].k < h.s[i].k) {
			break
		}
		h.s[i], h.s[j] = h.s[j], h.s[i]
		j = i
	}
}

func (h *expDecaySampleHeap) down(i, n int) {
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && !(h.s[j1].k < h.s[j2].k) {
			j = j2 // = 2*i + 2  // right child
		}
		if !(h.s[j].k < h.s[i].k) {
			break
		}
		h.s[i], h.s[j] = h.s[j], h.s[i]
		i = j
	}
}

type int64Slice []int64

func (p int64Slice) Len() int           { return len(p) }
func (p int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
