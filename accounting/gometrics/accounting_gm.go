//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

 Implementation of accounting API using the go-metrics
*/
package accounting_gm

import (
	"expvar"
	"fmt"
	"runtime"
	"sync"
	//	"syscall"
	"time"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/accounting/stub"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	metrics "github.com/rcrowley/go-metrics"
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

func (g *gometricsAccountingStore) HealthCheckRegistry() accounting.HealthCheckRegistry {
	return accounting_stub.HealthCheckRegistryStub{}
}

func (g *gometricsAccountingStore) Vitals() (interface{}, errors.Error) {
	var mem runtime.MemStats

	runtime.ReadMemStats(&mem)
	request_timer := g.registry.Timer(accounting.REQUEST_TIMER)
	request_rate := g.registry.Meter(accounting.REQUEST_RATE)

	// FIXME
	//	ru := syscall.Rusage{}
	//	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
	// TODO: log error
	//	}

	now := time.Now()
	newUtime := int64(0) //ru.Utime.Nano()
	newStime := int64(0) //ru.Stime.Nano()
	// end FIXME
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

	return VitalsRecord{
		Uptime:         uptime.String(),
		Threads:        runtime.NumGoroutine(),
		Cores:          runtime.GOMAXPROCS(0),
		GCNum:          mem.NextGC,
		GCPauseTime:    time.Duration(mem.PauseTotalNs).String(),
		GCPausePercent: util.RoundPlaces(pausePerc, 4),
		MemoryUsage:    mem.Alloc,
		MemoryTotal:    mem.TotalAlloc,
		MemorySys:      mem.Sys,
		CPUUser:        util.RoundPlaces(uPerc, 4),
		CPUSys:         util.RoundPlaces(sPerc, 4),
		ReqCount:       request_rate.Count(),
		Req1min:        util.RoundPlaces(request_rate.Rate1(), 4),
		Req5min:        util.RoundPlaces(request_rate.Rate5(), 4),
		Req15min:       util.RoundPlaces(request_rate.Rate15(), 4),
		ReqMean:        time.Duration(request_timer.Mean()).String(),
		ReqMedian:      time.Duration(request_timer.Percentile(.5)).String(),
		Req80:          time.Duration(request_timer.Percentile(.8)).String(),
		Req95:          time.Duration(request_timer.Percentile(.95)).String(),
		Req99:          time.Duration(request_timer.Percentile(.99)).String(),
	}, nil

}

type VitalsRecord struct {
	Uptime         string  `json:"uptime"`
	Threads        int     `json:"threads"`
	Cores          int     `json:"cores"`
	GCNum          uint64  `json:"gc.num"`
	GCPauseTime    string  `json:"gc.pause.time"`
	GCPausePercent float64 `json:"gc.pause.percent"`
	MemoryUsage    uint64  `json:"memory.usage"`
	MemoryTotal    uint64  `json:"memory.total"`
	MemorySys      uint64  `json:"memory.system"`
	CPUUser        float64 `json:"cpu.user.percent"`
	CPUSys         float64 `json:"cpu.sys.percent"`
	ReqCount       int64   `json:"request.count"`
	Req1min        float64 `json:"request.per.sec.1min"`
	Req5min        float64 `json:"request.per.sec.5min"`
	Req15min       float64 `json:"request.per.sec.15min"`
	ReqMean        string  `json:"request_time.mean"`
	ReqMedian      string  `json:"request_time.median"`
	Req80          string  `json:"request_time.80percentile"`
	Req95          string  `json:"request_time.95percentile"`
	Req99          string  `json:"request_time.99percentile"`

	// FIXME Active vs Queued threads, local time, version, direct vs prepared, network
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
