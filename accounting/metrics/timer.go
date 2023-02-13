package metrics

import (
	"time"

	"github.com/couchbase/query/util"
)

// Timers capture the duration and rate of events.
type Timer interface {
	Count() int64
	Max() int64
	Mean() float64
	Min() int64
	Percentile(float64) float64
	Percentiles([]float64) []float64
	Rate1() float64
	Rate5() float64
	Rate15() float64
	RateMean() float64
	StdDev() float64
	Sum() int64
	Time(func())
	Update(time.Duration)
	UpdateSince(time.Time)
	UpdateWithTimestamp(time.Time, time.Duration)
	Variance() float64
}

// GetOrRegisterTimer returns an existing Timer or constructs and registers a
// new StandardTimer.
func GetOrRegisterTimer(name string, r Registry) Timer {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewTimer).(Timer)
}

// NewRegisteredTimer constructs and registers a new StandardTimer.
func NewRegisteredTimer(name string, r Registry) Timer {
	c := NewTimer()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// NewTimer constructs a new StandardTimer using an exponentially-decaying
// sample with the same reservoir size and alpha as UNIX load averages.
func NewTimer() Timer {
	return &StandardTimer{
		histogram: NewHistogram(NewExpDecaySample(1028, 0.015)),
		meter:     NewMeter(),
	}
}

// StandardTimer is the standard implementation of a Timer and uses a Histogram
// and Meter.
type StandardTimer struct {
	histogram Histogram
	meter     Meter
}

// Count returns the number of events recorded.
func (t *StandardTimer) Count() int64 {
	return t.histogram.Count()
}

// Max returns the maximum value in the sample.
func (t *StandardTimer) Max() int64 {
	return t.histogram.Max()
}

// Mean returns the mean of the values in the sample.
func (t *StandardTimer) Mean() float64 {
	return t.histogram.Mean()
}

// Min returns the minimum value in the sample.
func (t *StandardTimer) Min() int64 {
	return t.histogram.Min()
}

// Percentile returns an arbitrary percentile of the values in the sample.
func (t *StandardTimer) Percentile(p float64) float64 {
	return t.histogram.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of the values in the
// sample.
func (t *StandardTimer) Percentiles(ps []float64) []float64 {
	return t.histogram.Percentiles(ps)
}

// Rate1 returns the one-minute moving average rate of events per second.
func (t *StandardTimer) Rate1() float64 {
	return t.meter.Rate1()
}

// Rate5 returns the five-minute moving average rate of events per second.
func (t *StandardTimer) Rate5() float64 {
	return t.meter.Rate5()
}

// Rate15 returns the fifteen-minute moving average rate of events per second.
func (t *StandardTimer) Rate15() float64 {
	return t.meter.Rate15()
}

// RateMean returns the meter's mean rate of events per second.
func (t *StandardTimer) RateMean() float64 {
	return t.meter.RateMean()
}

// StdDev returns the standard deviation of the values in the sample.
func (t *StandardTimer) StdDev() float64 {
	return t.histogram.StdDev()
}

// Sum returns the sum in the sample.
func (t *StandardTimer) Sum() int64 {
	return t.histogram.Sum()
}

// Record the duration of the execution of the given function.
func (t *StandardTimer) Time(f func()) {
	ts := util.Now()
	f()
	t.Update(util.Now().Sub(ts))
}

// Record the duration of an event.
func (t *StandardTimer) Update(d time.Duration) {
	t.histogram.Update(int64(d))
	t.meter.Mark(1)
}

// Record the duration of an event that started at a time and ends now.
func (t *StandardTimer) UpdateSince(ts time.Time) {
	t.histogram.Update(int64(time.Since(ts)))
	t.meter.Mark(1)
}

// Record the duration of an event at a specific point in time
func (t *StandardTimer) UpdateWithTimestamp(w time.Time, d time.Duration) {
	t.histogram.UpdateWithTimestamp(w, int64(d))
	t.meter.Mark(1)
}

// Variance returns the variance of the values in the sample.
func (t *StandardTimer) Variance() float64 {
	return t.histogram.Variance()
}
