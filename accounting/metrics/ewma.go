package metrics

import (
	"math"
	"sync"
	"sync/atomic"
)

const _TICK_FREQUENCY = 5e9

// EWMAs continuously calculate an exponentially-weighted moving average
// based on an outside source of clock ticks.
type EWMA interface {
	Rate() float64
	Tick()
	Update(int64)
}

// NewEWMA constructs a new EWMA with the given alpha.
func NewEWMA(alpha float64) EWMA {
	return &StandardEWMA{alpha: alpha}
}

// NewEWMA1 constructs a new EWMA for a one-minute moving average.
func NewEWMA1() EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/1))
}

// NewEWMA5 constructs a new EWMA for a five-minute moving average.
func NewEWMA5() EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/5))
}

// NewEWMA15 constructs a new EWMA for a fifteen-minute moving average.
func NewEWMA15() EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/15))
}

// StandardEWMA is the standard implementation of an EWMA and tracks the number
// of uncounted events and processes them on each tick.  It uses the
// sync/atomic package to manage uncounted events.
type StandardEWMA struct {
	uncounted int64 // /!\ this should be the first member to ensure 64-bit alignment
	alpha     float64
	rate      float64
	init      bool
	mutex     sync.Mutex
}

// Rate returns the moving average rate of events per second.
func (a *StandardEWMA) Rate() float64 {
	a.mutex.Lock()
	r := a.rate * float64(1e9)
	a.mutex.Unlock()
	return r
}

// Tick ticks the clock to update the moving average.  It assumes it is called
// every five seconds.
func (a *StandardEWMA) Tick() {
	count := atomic.LoadInt64(&a.uncounted)
	atomic.AddInt64(&a.uncounted, -count)
	instantRate := float64(count) / float64(_TICK_FREQUENCY)
	a.mutex.Lock()
	if a.init {
		a.rate += a.alpha * (instantRate - a.rate)
	} else {
		a.init = true
		a.rate = instantRate
	}
	a.mutex.Unlock()
}

// Update adds n uncounted events.
func (a *StandardEWMA) Update(n int64) {
	atomic.AddInt64(&a.uncounted, n)
}
