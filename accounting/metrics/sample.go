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
	UpdateWithTimestamp(time.Time, int64)
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
const _MIN_RESERVOIR_SIZE = 2
const _MAX_CACHED_BUFFER = 2

type ExpDecaySample struct {
	alpha            float64
	count            int64
	next             uint64
	mutex            sync.RWMutex
	reservoirSize    int
	t0, t1           time.Time
	percentileBuffer [][]int64
	reservoirs       [_RESERVOIRS]samples
}

// NewExpDecaySample constructs a new exponentially-decaying sample with the
// given reservoir size and alpha.
func NewExpDecaySample(reservoirSize int, alpha float64) Sample {
	rSz := reservoirSize / _RESERVOIRS
	if rSz < _MIN_RESERVOIR_SIZE {
		rSz = _MIN_RESERVOIR_SIZE
	}
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

	start := 0
	for r, _ := range s.reservoirs {
		s.reservoirs[r].values.s = make([]expDecaySample, rSz, rSz)
		s.reservoirs[r].values.count = 0
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
	for i := range s.reservoirs {
		s.reservoirs[i].values.count = 0
	}
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
	var max int64 = math.MinInt64
	set := false

	s.scan(func(v expDecaySample) {
		if max < v.v {
			max = v.v
			set = true
		}
	})
	if !set {
		return 0
	}
	return max
}

// Mean returns the mean of the values in the sample.
func (s *ExpDecaySample) Mean() float64 {
	var sum int64 = 0
	l := 0
	s.scan(func(v expDecaySample) {
		sum += v.v
		l++
	})
	if 0 == l {
		return 0.0
	}
	return float64(sum) / float64(l)
}

// Min returns the minimum value in the sample, which may not be the minimum
// value ever to be part of the sample.
func (s *ExpDecaySample) Min() int64 {
	var min int64 = math.MaxInt64
	set := false

	s.scan(func(v expDecaySample) {
		if min > v.v {
			min = v.v
			set = true
		}
	})
	if !set {
		return 0
	}
	return min
}

// Percentile returns an arbitrary percentile of values in the sample.
func (s *ExpDecaySample) Percentile(p float64) float64 {
	return s.Percentiles([]float64{p})[0]
}

// Percentiles returns a slice of arbitrary percentiles of values in the
// sample.
func (s *ExpDecaySample) Percentiles(ps []float64) []float64 {
	var vals []int64

	s.mutex.Lock()
	l := len(s.percentileBuffer)
	if l > 0 {
		vals = s.percentileBuffer[l-1]
		s.percentileBuffer = s.percentileBuffer[:l-1]
		vals = vals[:0]
	} else {
		vals = make([]int64, 0, s.reservoirSize)
	}
	s.mutex.Unlock()
	s.scan(func(v expDecaySample) {
		vals = append(vals, v.v)
	})
	values := int64Slice(vals)
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
	s.mutex.Lock()
	if len(s.percentileBuffer) < _MAX_CACHED_BUFFER {
		vals = vals[:0]
		s.percentileBuffer = append(s.percentileBuffer, vals)
	}
	s.mutex.Unlock()
	return scores
}

// Size returns the size of the sample, which is at most the reservoir size.
func (s *ExpDecaySample) Size() int {
	sz := 0
	for r, _ := range s.reservoirs {
		s.reservoirs[r].mutex.RLock()
		sz += int(s.reservoirs[r].values.count)
		s.reservoirs[r].mutex.RUnlock()
	}
	return sz
}

// StdDev returns the standard deviation of the values in the sample.
func (s *ExpDecaySample) StdDev() float64 {
	return math.Sqrt(s.Variance())
}

// Sum returns the sum of the values in the sample.
func (s *ExpDecaySample) Sum() int64 {
	var sum int64 = 0
	s.scan(func(v expDecaySample) {
		sum += v.v
	})
	return sum
}

// Update samples a new value.
func (s *ExpDecaySample) Update(v int64) {
	s.UpdateWithTimestamp(time.Now(), v)
}

// Values returns a copy of the values in the sample.
func (s *ExpDecaySample) Values() []int64 {
	values := make([]int64, 0, s.reservoirSize)
	s.scan(func(v expDecaySample) {
		values = append(values, v.v)
	})
	return values
}

// Variance returns the variance of the values in the sample.
func (s *ExpDecaySample) Variance() float64 {
	var m float64
	l := 0
	s.scan(func(v expDecaySample) {
		m += float64(v.v)
		l++
	})
	if 0 == l {
		return 0.0
	}
	m /= float64(l)
	var sum float64
	s.scan(func(v expDecaySample) {
		d := float64(v.v) - m
		sum += d * d
	})
	return float64(sum) / float64(l)
}

// update samples a new value at a particular timestamp.  This is a method all
// its own to facilitate testing.
func (s *ExpDecaySample) UpdateWithTimestamp(t time.Time, v int64) {

	// no clearing for now
	s.mutex.RLock()
	rescale := t.After(s.t1)
	atomic.AddInt64(&s.count, 1)

	// choose and amend reservoir
	next := atomic.AddUint64(&s.next, 1) % _RESERVOIRS
	ed := expDecaySample{
		k: math.Exp(t.Sub(s.t0).Seconds()*s.alpha) / s.reservoirs[next].random.Float64(),
		v: v,
	}

	s.reservoirs[next].mutex.Lock()
	if s.reservoirs[next].values.count == s.reservoirs[next].reservoirSize {
		s.reservoirs[next].values.rotate(ed)
	} else {
		s.reservoirs[next].values.add(ed)
	}
	s.reservoirs[next].mutex.Unlock()
	s.mutex.RUnlock()

	// numbers getting high - choose a new landmark
	if rescale {
		s.mutex.Lock()

		// somebody may have already done it
		if t.After(s.t1) {
			t0 := s.t0
			s.t0 = t
			s.t1 = s.t0.Add(rescaleThreshold)
			newLandmark := math.Exp(-s.alpha * s.t0.Sub(t0).Seconds())
			for r, _ := range s.reservoirs {
				s.reservoirs[r].mutex.RLock()
				for i := 0; i <= s.reservoirs[r].values.count; i++ {
					s.reservoirs[r].values.s[i].k = s.reservoirs[r].values.s[i].k * newLandmark
				}
				s.reservoirs[r].mutex.RUnlock()
			}
		}
		s.mutex.Unlock()
	}
}

func (s *ExpDecaySample) scan(f func(v expDecaySample)) {
	s.mutex.RLock()
	for r, _ := range s.reservoirs {
		s.reservoirs[r].mutex.RLock()
		for i := 0; i < s.reservoirs[r].values.count; i++ {
			f(s.reservoirs[r].values.s[i])
		}
		s.reservoirs[r].mutex.RUnlock()
	}
	s.mutex.RUnlock()
}

// expDecaySample represents an individual sample in a heap.
type expDecaySample struct {
	k float64
	v int64
}

// expDecaySampleHeap is a min-heap of expDecaySamples.
// The internal implementation is copied from the standard library's container/heap
type expDecaySampleHeap struct {
	s     []expDecaySample
	count int
}

func (h *expDecaySampleHeap) add(s expDecaySample) {
	h.s[h.count] = s
	h.up(h.count)
	h.count++
}

func (h *expDecaySampleHeap) rotate(s expDecaySample) {
	n := len(h.s) - 1
	h.s[0], h.s[n] = h.s[n], h.s[0]
	h.down(0, n)

	h.s[n] = s
	h.up(n)
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
