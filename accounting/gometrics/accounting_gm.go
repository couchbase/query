//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Implementation of accounting API using a go-metrics like package

package accounting_gm

import (
	"expvar"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/accounting/metrics"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/util"
)

type gometricsAccountingStore struct {
	sync.Mutex
	registry accounting.MetricRegistry
	reporter accounting.MetricReporter
	vitals   map[string]interface{}
}

func NewAccountingStore() accounting.AccountingStore {
	rv := &gometricsAccountingStore{
		registry: &goMetricRegistry{},
		reporter: &goMetricReporter{},
		vitals:   map[string]interface{}{},
	}

	var lastUtime, lastStime int64
	var lastPauseTime uint64
	var lastNow time.Time
	startTime := time.Now()

	rv.vitals["lastUtime"] = lastUtime
	rv.vitals["lastStime"] = lastStime
	rv.vitals["lastNow"] = lastNow
	rv.vitals["lastPauseTime"] = lastPauseTime
	rv.vitals["startTime"] = startTime

	return rv
}

func (g *gometricsAccountingStore) Id() string {
	return "gometrics"
}

func (g *gometricsAccountingStore) URL() string {
	return "gometrics"
}

func (g *gometricsAccountingStore) MetricRegistry() accounting.MetricRegistry {
	return g.registry
}

func (g *gometricsAccountingStore) MetricReporter() accounting.MetricReporter {
	return g.reporter
}

func (g *gometricsAccountingStore) Vitals() (interface{}, errors.Error) {
	var mem runtime.MemStats

	runtime.ReadMemStats(&mem)
	request_timer := g.registry.Timer(accounting.REQUEST_TIMER)
	prepared := g.registry.Counter(accounting.PREPAREDS)

	now := time.Now()
	newUtime, newStime := util.CpuTimes()

	g.Lock()
	uptime := now.Sub(g.vitals["startTime"].(time.Time))
	dur := float64(now.Sub(g.vitals["lastNow"].(time.Time)))
	uPerc := float64(newUtime-g.vitals["lastUtime"].(int64)) / dur
	sPerc := float64(newStime-g.vitals["lastStime"].(int64)) / dur
	pausePerc := float64(mem.PauseTotalNs-g.vitals["lastPauseTime"].(uint64)) / dur

	g.vitals["lastNow"] = now
	g.vitals["lastUtime"] = newUtime
	g.vitals["lastStime"] = newStime
	g.vitals["lastPauseTime"] = mem.PauseTotalNs
	g.Unlock()

	actCount, _ := server.ActiveRequestsCount()
	totCount := request_timer.Count()
	var prepPercent float64
	if totCount > 0 {
		prepPercent = float64(prepared.Count()) / float64(totCount)
	} else {
		prepPercent = 0.0
	}

	return VitalsRecord{
		Uptime:         uptime.String(),
		LocalTime:      now.String(),
		Version:        util.VERSION,
		TotThreads:     runtime.NumGoroutine(),
		Cores:          runtime.GOMAXPROCS(0),
		GCNum:          mem.NextGC,
		GCPauseTime:    time.Duration(mem.PauseTotalNs).String(),
		GCPausePercent: util.RoundPlaces(pausePerc, 4),
		MemoryUsage:    mem.Alloc,
		MemoryTotal:    mem.TotalAlloc,
		MemorySys:      mem.Sys,
		CPUUser:        util.RoundPlaces(uPerc, 4),
		CPUSys:         util.RoundPlaces(sPerc, 4),
		ReqCount:       totCount,
		ActCount:       int64(actCount),
		Req1min:        util.RoundPlaces(request_timer.Rate1(), 4),
		Req5min:        util.RoundPlaces(request_timer.Rate5(), 4),
		Req15min:       util.RoundPlaces(request_timer.Rate15(), 4),
		ReqMean:        time.Duration(request_timer.Mean()).String(),
		ReqMedian:      time.Duration(request_timer.Percentile(.5)).String(),
		Req80:          time.Duration(request_timer.Percentile(.8)).String(),
		Req95:          time.Duration(request_timer.Percentile(.95)).String(),
		Req99:          time.Duration(request_timer.Percentile(.99)).String(),
		Prepared:       prepPercent,
	}, nil

}

type VitalsRecord struct {
	Uptime         string  `json:"uptime"`
	LocalTime      string  `json:"local.time"`
	Version        string  `json:"version"`
	TotThreads     int     `json:"total.threads"`
	Cores          int     `json:"cores"`
	GCNum          uint64  `json:"gc.num"`
	GCPauseTime    string  `json:"gc.pause.time"`
	GCPausePercent float64 `json:"gc.pause.percent"`
	MemoryUsage    uint64  `json:"memory.usage"`
	MemoryTotal    uint64  `json:"memory.total"`
	MemorySys      uint64  `json:"memory.system"`
	CPUUser        float64 `json:"cpu.user.percent"`
	CPUSys         float64 `json:"cpu.sys.percent"`
	ReqCount       int64   `json:"request.completed.count"`
	ActCount       int64   `json:"request.active.count"`
	Req1min        float64 `json:"request.per.sec.1min"`
	Req5min        float64 `json:"request.per.sec.5min"`
	Req15min       float64 `json:"request.per.sec.15min"`
	ReqMean        string  `json:"request_time.mean"`
	ReqMedian      string  `json:"request_time.median"`
	Req80          string  `json:"request_time.80percentile"`
	Req95          string  `json:"request_time.95percentile"`
	Req99          string  `json:"request_time.99percentile"`
	Prepared       float64 `json:"request.prepared.percent"`

	// FIXME Active vs Queued threads, local time, version, direct vs prepared, network
}

func (g *gometricsAccountingStore) NewCounter() accounting.Counter {
	return metrics.NewCounter()
}

func (g *gometricsAccountingStore) NewGauge() accounting.Gauge {
	return metrics.NewGauge()
}

func (g *gometricsAccountingStore) NewMeter() accounting.Meter {
	return metrics.NewMeter()
}

func (g *gometricsAccountingStore) NewTimer() accounting.Timer {
	return metrics.NewTimer()
}

func (g *gometricsAccountingStore) NewHistogram() accounting.Histogram {
	return metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015))
}

type goMetricRegistry struct {
}

func (g *goMetricRegistry) Register(name string, metric accounting.Metric) errors.Error {
	err := metrics.Register(name, metric)
	if err != nil {
		return errors.NewAdminMakeMetricError(err, name)
	}
	return nil
}

func (g *goMetricRegistry) Get(name string) accounting.Metric {
	return metrics.Get(name)
}

func (g *goMetricRegistry) Unregister(name string) errors.Error {
	metrics.Unregister(name)
	return nil
}

func (g *goMetricRegistry) Counter(name string) accounting.Counter {
	return metrics.GetOrRegisterCounter(name, metrics.DefaultRegistry)
}

func (g *goMetricRegistry) Gauge(name string) accounting.Gauge {
	return metrics.GetOrRegisterGauge(name, metrics.DefaultRegistry)
}

func (g *goMetricRegistry) Meter(name string) accounting.Meter {
	return metrics.GetOrRegisterMeter(name, metrics.DefaultRegistry)
}

func (g *goMetricRegistry) Timer(name string) accounting.Timer {
	return metrics.GetOrRegisterTimer(name, metrics.DefaultRegistry)
}

func (g *goMetricRegistry) Histogram(name string) accounting.Histogram {
	return metrics.GetOrRegisterHistogram(name, metrics.DefaultRegistry, metrics.NewExpDecaySample(1028, 0.015))
}

func (g *goMetricRegistry) Counters() map[string]accounting.Counter {
	r := metrics.DefaultRegistry
	counters := make(map[string]accounting.Counter)
	r.Each(func(name string, i interface{}) {
		switch m := i.(type) {
		case metrics.Counter:
			counters[name] = m
		}
	})
	return counters
}

func (g *goMetricRegistry) Gauges() map[string]accounting.Gauge {
	r := metrics.DefaultRegistry
	gauges := make(map[string]accounting.Gauge)
	r.Each(func(name string, i interface{}) {
		switch m := i.(type) {
		case metrics.Gauge:
			gauges[name] = m
		}
	})
	return gauges
}

func (g *goMetricRegistry) Meters() map[string]accounting.Meter {
	r := metrics.DefaultRegistry
	meters := make(map[string]accounting.Meter)
	r.Each(func(name string, i interface{}) {
		switch m := i.(type) {
		case metrics.Meter:
			meters[name] = m
		}
	})
	return meters
}

func (g *goMetricRegistry) Timers() map[string]accounting.Timer {
	r := metrics.DefaultRegistry
	timers := make(map[string]accounting.Timer)
	r.Each(func(name string, i interface{}) {
		switch m := i.(type) {
		case metrics.Timer:
			timers[name] = m
		}
	})
	return timers
}

func (g *goMetricRegistry) Histograms() map[string]accounting.Histogram {
	r := metrics.DefaultRegistry
	histograms := make(map[string]accounting.Histogram)
	r.Each(func(name string, i interface{}) {
		switch m := i.(type) {
		case metrics.Histogram:
			histograms[name] = m
		}
	})
	return histograms
}

type goMetricReporter struct {
}

func (g *goMetricReporter) MetricRegistry() accounting.MetricRegistry {
	return &goMetricRegistry{}
}

func (g *goMetricReporter) Start(interval int64, unit time.Duration) {
	// Expose the metrics to expvars
	publish_expvars(metrics.DefaultRegistry)
}

func (g *goMetricReporter) Stop() {
	// Stop exposing the metrics to expvars
}

func (g *goMetricReporter) Report() {
}

func (g *goMetricReporter) RateUnit() time.Duration {
	// Redundant: RateUnit determined by client of expvars
	// (i.e. whatever is polling expvars endpoint)
	return 0
}

// publish_expvars: expose each metric in the given registry to expvars
func publish_expvars(r metrics.Registry) {
	du := float64(time.Nanosecond)
	percentiles := []float64{0.50, 0.75, 0.95, 0.99, 0.999}
	r.Each(func(name string, i interface{}) {
		switch m := i.(type) {
		case metrics.Counter:
			expvar.Publish(fmt.Sprintf("%s.Count", name), expvar.Func(func() interface{} {
				return m.Count()
			}))
		case metrics.Meter:
			expvar.Publish(fmt.Sprintf("%s.Rate1", name), expvar.Func(func() interface{} {
				return m.Rate1()
			}))
			expvar.Publish(fmt.Sprintf("%s.Rate5", name), expvar.Func(func() interface{} {
				return m.Rate5()
			}))
			expvar.Publish(fmt.Sprintf("%s.Rate15", name), expvar.Func(func() interface{} {
				return m.Rate15()
			}))
			expvar.Publish(fmt.Sprintf("%s.Mean", name), expvar.Func(func() interface{} {
				return m.RateMean()
			}))
		case metrics.Histogram:
			expvar.Publish(fmt.Sprintf("%s.Count", name), expvar.Func(func() interface{} {
				return m.Count()
			}))
			expvar.Publish(fmt.Sprintf("%s.Mean", name), expvar.Func(func() interface{} {
				return m.Mean()
			}))
			expvar.Publish(fmt.Sprintf("%s.Min", name), expvar.Func(func() interface{} {
				return m.Min()
			}))
			expvar.Publish(fmt.Sprintf("%s.Max", name), expvar.Func(func() interface{} {
				return m.Max()
			}))
			expvar.Publish(fmt.Sprintf("%s.StdDev", name), expvar.Func(func() interface{} {
				return m.StdDev()
			}))
			expvar.Publish(fmt.Sprintf("%s.Variance", name), expvar.Func(func() interface{} {
				return m.Variance()
			}))
			for _, p := range percentiles {
				expvar.Publish(fmt.Sprintf("%s.Percentile%2.3f", name, p), expvar.Func(func() interface{} {
					return m.Percentile(p)
				}))
			}
		case metrics.Timer:
			expvar.Publish(fmt.Sprintf("%s.Rate1", name), expvar.Func(func() interface{} {
				return m.Rate1()
			}))
			expvar.Publish(fmt.Sprintf("%s.Rate5", name), expvar.Func(func() interface{} {
				return m.Rate5()
			}))
			expvar.Publish(fmt.Sprintf("%s.Rate15", name), expvar.Func(func() interface{} {
				return m.Rate15()
			}))
			expvar.Publish(fmt.Sprintf("%s.RateMean", name), expvar.Func(func() interface{} {
				return m.RateMean()
			}))
			expvar.Publish(fmt.Sprintf("%s.Mean", name), expvar.Func(func() interface{} {
				return du * m.Mean()
			}))
			expvar.Publish(fmt.Sprintf("%s.Min", name), expvar.Func(func() interface{} {
				return int64(du) * m.Min()
			}))
			expvar.Publish(fmt.Sprintf("%s.Max", name), expvar.Func(func() interface{} {
				return int64(du) * m.Max()
			}))
			expvar.Publish(fmt.Sprintf("%s.StdDev", name), expvar.Func(func() interface{} {
				return du * m.StdDev()
			}))
			expvar.Publish(fmt.Sprintf("%s.Variance", name), expvar.Func(func() interface{} {
				return du * m.Variance()
			}))
			for _, p := range percentiles {
				expvar.Publish(fmt.Sprintf("%s.Percentile%2.3f", name, p), expvar.Func(func() interface{} {
					return m.Percentile(p)
				}))
			}
		}
	})
	expvar.Publish("time", expvar.Func(now))
}

func now() interface{} {
	return time.Now().Format(time.RFC3339Nano)
}
