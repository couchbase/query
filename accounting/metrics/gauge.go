package metrics

import "sync/atomic"

// Gauges hold an int64 value that can be set arbitrarily.
type Gauge interface {
	Update(int64)
	Value() int64
}

// GetOrRegisterGauge returns an existing Gauge or constructs and registers a
// new StandardGauge.
func GetOrRegisterGauge(name string, r Registry) Gauge {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewGauge).(Gauge)
}

// NewGauge constructs a new StandardGauge.
func NewGauge() Gauge {
	return &StandardGauge{0}
}

// NewRegisteredGauge constructs and registers a new StandardGauge.
func NewRegisteredGauge(name string, r Registry) Gauge {
	c := NewGauge()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// StandardGauge is the standard implementation of a Gauge and uses the
// sync/atomic package to manage a single int64 value.
type StandardGauge struct {
	value int64
}

// Update updates the gauge's value.
func (g *StandardGauge) Update(v int64) {
	atomic.StoreInt64(&g.value, v)
}

// Value returns the gauge's current value.
func (g *StandardGauge) Value() int64 {
	return atomic.LoadInt64(&g.value)
}
