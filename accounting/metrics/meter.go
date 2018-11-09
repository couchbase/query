package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Meters count events to produce exponentially-weighted moving average rates
// at one-, five-, and fifteen-minutes and a mean rate.
type Meter interface {
	Count() int64
	Mark(int64)
	Rate1() float64
	Rate5() float64
	Rate15() float64
	RateMean() float64
}

// GetOrRegisterMeter returns an existing Meter or constructs and registers a
// new StandardMeter.
func GetOrRegisterMeter(name string, r Registry) Meter {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewMeter).(Meter)
}

// NewMeter constructs a new StandardMeter and launches a goroutine.
func NewMeter() Meter {
	m := newStandardMeter()
	arbiter.Lock()
	defer arbiter.Unlock()
	arbiter.meters = append(arbiter.meters, m)
	if !arbiter.started {
		arbiter.started = true
		go arbiter.tick()
	}
	return m
}

// NewMeter constructs and registers a new StandardMeter and launches a
// goroutine.
func NewRegisteredMeter(name string, r Registry) Meter {
	c := NewMeter()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// meterSnapshot is a read-only copy of another Meter.
type meterSnapshot struct {
	count                          int64
	rate1, rate5, rate15, rateMean float64
}

// StandardMeter is the standard implementation of a Meter.
type StandardMeter struct {
	lock        sync.RWMutex
	snapshot    meterSnapshot
	a1, a5, a15 EWMA
	startTime   time.Time
}

func newStandardMeter() *StandardMeter {
	return &StandardMeter{
		a1:        NewEWMA1(),
		a5:        NewEWMA5(),
		a15:       NewEWMA15(),
		startTime: time.Now(),
	}
}

// Count returns the number of events recorded.
func (m *StandardMeter) Count() int64 {
	return atomic.LoadInt64(&m.snapshot.count)
}

// Mark records the occurance of n events.
func (m *StandardMeter) Mark(n int64) {
	atomic.AddInt64(&m.snapshot.count, n)
	m.a1.Update(n)
	m.a5.Update(n)
	m.a15.Update(n)
}

// Rate1 returns the one-minute moving average rate of events per second.
func (m *StandardMeter) Rate1() float64 {
	m.lock.Lock()
	m.updateSnapshot()
	rate1 := m.snapshot.rate1
	m.lock.Unlock()
	return rate1
}

// Rate5 returns the five-minute moving average rate of events per second.
func (m *StandardMeter) Rate5() float64 {
	m.lock.Lock()
	m.updateSnapshot()
	rate5 := m.snapshot.rate5
	m.lock.Unlock()
	return rate5
}

// Rate15 returns the fifteen-minute moving average rate of events per second.
func (m *StandardMeter) Rate15() float64 {
	m.lock.Lock()
	m.updateSnapshot()
	rate15 := m.snapshot.rate15
	m.lock.Unlock()
	return rate15
}

// RateMean returns the meter's mean rate of events per second.
func (m *StandardMeter) RateMean() float64 {
	m.lock.Lock()
	m.updateSnapshot()
	rateMean := m.snapshot.rateMean
	m.lock.Unlock()
	return rateMean
}

// has to be run with write lock held on m.lock
func (m *StandardMeter) updateSnapshot() {
	m.snapshot.rate1 = m.a1.Rate()
	m.snapshot.rate5 = m.a5.Rate()
	m.snapshot.rate15 = m.a15.Rate()
	m.snapshot.rateMean = float64(m.snapshot.count) / time.Since(m.startTime).Seconds()
}

func (m *StandardMeter) tick() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.a1.Tick()
	m.a5.Tick()
	m.a15.Tick()
	m.updateSnapshot()
}

type meterArbiter struct {
	sync.RWMutex
	started bool
	meters  []*StandardMeter
	ticker  *time.Ticker
}

var arbiter = meterArbiter{ticker: time.NewTicker(_TICK_FREQUENCY)}

// Ticks meters on the scheduled interval
func (ma *meterArbiter) tick() {
	for {
		select {
		case <-ma.ticker.C:
			ma.tickMeters()
		}
	}
}

func (ma *meterArbiter) tickMeters() {
	ma.RLock()
	defer ma.RUnlock()
	for _, meter := range ma.meters {
		meter.tick()
	}
}
