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
	"time"

	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/accounting/stub"
	"github.com/couchbaselabs/query/errors"
	metrics "github.com/rcrowley/go-metrics"
)

type gometricsAccountingStore struct {
	registry accounting.MetricRegistry
	reporter accounting.MetricReporter
}

func NewAccountingStore() accounting.AccountingStore {
	return &gometricsAccountingStore{
		registry: &goMetricRegistry{},
		reporter: &goMetricReporter{},
	}
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
