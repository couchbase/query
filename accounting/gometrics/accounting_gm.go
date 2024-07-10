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
	"github.com/couchbase/query/ffdc"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/memory"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/system"
	"github.com/couchbase/query/util"
)

type gometricsAccountingStore struct {
	sync.Mutex
	registry      accounting.MetricRegistry
	reporter      accounting.MetricReporter
	stats         *system.SystemStats
	lastUtime     int64
	lastStime     int64
	lastPauseTime uint64
	lastNow       time.Time
	startTime     time.Time
	vitals        map[string]interface{}
}

func NewAccountingStore() (accounting.AccountingStore, errors.Error) {
	var err error
	rv := &gometricsAccountingStore{
		registry: &goMetricRegistry{},
		reporter: &goMetricReporter{},
	}

	// open sigar for stats
	rv.stats, err = system.NewSystemStats()
	if err != nil {
		logging.Errorf("Accounting store error : %v", err)
		return nil, errors.NewAdminStartError(err)
	}

	// skip the first one
	rv.stats.ProcessCpuStats()
	rv.stats.ProcessRSS()
	rv.stats.GetTotalAndFreeMem()

	rv.startTime = time.Now()

	return rv, nil
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

func (g *gometricsAccountingStore) CompletedRequests() int64 {
	request_timer := g.registry.Timer(accounting.REQUEST_TIMER)
	return request_timer.Count()
}

func (g *gometricsAccountingStore) Vitals(style util.DurationStyle) (map[string]interface{}, errors.Error) {
	var mem runtime.MemStats

	runtime.ReadMemStats(&mem)
	request_timer := g.registry.Timer(accounting.REQUEST_TIMER)
	prepared := g.registry.Counter(accounting.PREPAREDS)
	used_memory_hwm := g.registry.Gauge(accounting.USED_MEMORY_HWM)
	spillsOrder := g.registry.Counter(accounting.SPILLS_ORDER_STR)

	now := time.Now()
	newUtime, newStime := util.CpuTimes()

	g.Lock()
	uptime := now.Sub(g.startTime)
	dur := float64(now.Sub(g.lastNow))
	uPerc := float64(newUtime-g.lastUtime) / dur
	sPerc := float64(newStime-g.lastStime) / dur
	pausePerc := float64(mem.PauseTotalNs-g.lastPauseTime) / dur

	g.lastNow = now
	g.lastUtime = newUtime
	g.lastStime = newStime
	g.lastPauseTime = mem.PauseTotalNs
	g.Unlock()

	actCount, _ := server.ActiveRequestsCount()
	totCount := request_timer.Count()
	var prepPercent float64
	if totCount > 0 {
		prepPercent = float64(prepared.Count()) / float64(totCount)
	} else {
		prepPercent = 0.0
	}
	rv := map[string]interface{}{
		"uptime":                    util.FormatDuration(uptime, style),
		"local.time":                now.Format(util.DEFAULT_FORMAT),
		"version":                   util.VERSION,
		"total.threads":             runtime.NumGoroutine(),
		"cores":                     runtime.GOMAXPROCS(0),
		"gc.num":                    mem.NextGC,
		"gc.pause.time":             util.FormatDuration(time.Duration(mem.PauseTotalNs), style),
		"gc.pause.percent":          util.RoundPlaces(pausePerc, 4),
		"memory.usage":              mem.Alloc,
		"memory.total":              mem.TotalAlloc,
		"memory.system":             mem.Sys,
		"cpu.user.percent":          util.RoundPlaces(uPerc, 4),
		"cpu.sys.percent":           util.RoundPlaces(sPerc, 4),
		"request.completed.count":   totCount,
		"request.active.count":      int64(actCount),
		"request.quota.used.hwm":    used_memory_hwm.Value(),
		"request.per.sec.1min":      util.RoundPlaces(request_timer.Rate1(), 4),
		"request.per.sec.5min":      util.RoundPlaces(request_timer.Rate5(), 4),
		"request.per.sec.15min":     util.RoundPlaces(request_timer.Rate15(), 4),
		"request_time.mean":         util.FormatDuration(time.Duration(request_timer.Mean()), style),
		"request_time.median":       util.FormatDuration(time.Duration(request_timer.Percentile(.5)), style),
		"request_time.80percentile": util.FormatDuration(time.Duration(request_timer.Percentile(.8)), style),
		"request_time.95percentile": util.FormatDuration(time.Duration(request_timer.Percentile(.95)), style),
		"request_time.99percentile": util.FormatDuration(time.Duration(request_timer.Percentile(.99)), style),
		"request.prepared.percent":  prepPercent,
		"spills.order":              spillsOrder.Count(),
	}
	g.Lock()
	_, rss, total, free, _, err := system.GetSystemStats(g.stats, false, true)
	if err == nil {
		rv["process.rss"] = rss
		rv["host.memory.total"] = total
		rv["host.memory.free"] = free

	} else {
		logging.Debugf("statsCollector error : %v", err)
	}
	rv["host.memory.quota"] = memory.NodeQuota() * util.MiB
	rv["host.memory.value_quota"] = memory.Quota() * util.MiB
	tc, th := util.TempStats()
	rv["temp.usage"] = tc
	rv["temp.hwm"] = th
	for n, v := range g.vitals {
		rv[n] = v
	}
	ffdc.Stats("ffdc.", rv, false)
	server.RequestsFileStreamStats(rv)
	g.Unlock()
	return rv, nil
}

func (g *gometricsAccountingStore) ExternalVitals(vitals map[string]interface{}) map[string]interface{} {
	g.Lock()
	oldVitals := g.vitals
	g.vitals = vitals
	g.Unlock()
	return oldVitals
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
