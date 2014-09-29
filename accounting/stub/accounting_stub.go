//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

 Packace accounting provides a common API for workload and monitoring data - metrics, statistics, events.
*/
package accounting_stub

import (
	"time"

	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/errors"
)

const (
	ACCOUNTING_STORE_STUB_ID  string = "AccountingStoreStubId"
	ACCOUNTING_STORE_STUB_URL string = "AccountingStoreStubURL"
)

// AccountingStoreStub is a stub implementation of AccountingStore
type AccountingStoreStub struct{}

func NewAccountingStore(path string) (accounting.AccountingStore, errors.Error) {
	return &AccountingStoreStub{}, nil
}

func (AccountingStoreStub) Id() string {
	return ACCOUNTING_STORE_STUB_ID
}

func (AccountingStoreStub) URL() string {
	return ACCOUNTING_STORE_STUB_URL
}

func (AccountingStoreStub) MetricRegistry() accounting.MetricRegistry {
	return MetricRegistryStub{}
}

func (AccountingStoreStub) HealthCheckRegistry() accounting.HealthCheckRegistry {
	return HealthCheckRegistryStub{}
}

// CounterStub is a stub implementation of Counter
type CounterStub struct{}

func (CounterStub) History() []accounting.TimeSeriesPoint {
	return []accounting.TimeSeriesPoint{}
}

func (CounterStub) Inc(amount int64) {} // No-op

func (CounterStub) Dec(amount int64) {} // No-op

func (CounterStub) Count() int64 { return 0 }

// GaugeStub is a stub implementation of Gauge
type GaugeStub struct{}

func (GaugeStub) History() []accounting.TimeSeriesPoint {
	return []accounting.TimeSeriesPoint{}
}

func (GaugeStub) Value() float64 { return 0.0 }

// MeterStub is a stub implementation of Meter
type MeterStub struct{}

func (MeterStub) History() []accounting.TimeSeriesPoint {
	return []accounting.TimeSeriesPoint{}
}

func (MeterStub) RateN(n int) float64 { return 0.0 }

func (MeterStub) Mean() float64 { return 0.0 }

func (MeterStub) Mark(n int64) {} // No-op

func (MeterStub) Count() int64 { return 0 }

// TimerStub is a stub implementation of Timer
type TimerStub struct{}

func (TimerStub) History() []accounting.TimeSeriesPoint {
	return []accounting.TimeSeriesPoint{}
}

func (TimerStub) Start() {} // No-op

func (TimerStub) Stop() {} // No-op

func (TimerStub) Value() time.Duration { return 0 }

// HistogramStub is a stub implementation of Histogram
type HistogramStub struct{}

func (HistogramStub) History() []accounting.TimeSeriesPoint {
	return []accounting.TimeSeriesPoint{}
}

func (HistogramStub) Percentile(n float64) float64 { return 0.0 }

func (HistogramStub) Percentiles(n []float64) []float64 { return make([]float64, len(n)) }

func (HistogramStub) StdDev() float64 { return 0.0 }

func (HistogramStub) Variance() float64 { return 0.0 }

// AggregateStub is a stub implementation of Aggregate
type AggregateStub struct{}

func (AggregateStub) History() []accounting.TimeSeriesPoint {
	return []accounting.TimeSeriesPoint{}
}

func (AggregateStub) Count() int64 { return 0 }

func (AggregateStub) Max() float64 { return 0.0 }

func (AggregateStub) Mean() float64 { return 0.0 }

func (AggregateStub) Min() float64 { return 0.0 }

func (AggregateStub) Sum() float64 { return 0.0 }

// MetricRegistryStub is a stub implementation of MetricRegistry
type MetricRegistryStub struct{}

func (MetricRegistryStub) Register(name string, metric accounting.Metric) errors.Error {
	return nil
}

func (MetricRegistryStub) Get(name string) accounting.Metric {
	return nil
}

func (MetricRegistryStub) Unregister(name string) errors.Error {
	return nil
}

func (MetricRegistryStub) Counter(name string) accounting.Counter {
	return CounterStub{}
}

func (MetricRegistryStub) Gauge(name string) accounting.Gauge {
	return GaugeStub{}
}

func (MetricRegistryStub) Meter(name string) accounting.Meter {
	return MeterStub{}
}

func (MetricRegistryStub) Timer(name string) accounting.Timer {
	return TimerStub{}
}

func (MetricRegistryStub) Histogram(name string) accounting.Histogram {
	return HistogramStub{}
}

func (MetricRegistryStub) Aggregate(name string) accounting.Aggregate {
	return AggregateStub{}
}

func (MetricRegistryStub) Counters() map[string]accounting.Counter {
	return nil
}

func (MetricRegistryStub) Gauges() map[string]accounting.Gauge {
	return nil
}

func (MetricRegistryStub) Meters() map[string]accounting.Meter {
	return nil
}

func (MetricRegistryStub) Timers() map[string]accounting.Timer {
	return nil
}

func (MetricRegistryStub) Histograms() map[string]accounting.Histogram {
	return nil
}

func (MetricRegistryStub) Aggregates() map[string]accounting.Aggregate {
	return nil
}

// A check that tests the status of an entity or compares a metric value against a
// configurable threshold.
type HealthCheckStub struct{}

func (HealthCheckStub) Check() (accounting.HealthCheckResult, errors.Error) {
	return HealthCheckResultStub{}, nil
}

type HealthCheckResultStub struct{}

func (HealthCheckResultStub) IsHealthy() bool { return true }

func (HealthCheckResultStub) Message() string { return "" }

func (HealthCheckResultStub) Error() errors.Error { return nil }

// HealthCheckRegistry is a centralized container for managing all health checks.
type HealthCheckRegistryStub struct{}

func (HealthCheckRegistryStub) Register(name string, hc accounting.HealthCheck) errors.Error {
	return nil
}

func (HealthCheckRegistryStub) Unregister(name string) errors.Error {
	return nil
}

func (HealthCheckRegistryStub) RunHealthChecks() (map[string]accounting.HealthCheckResult, errors.Error) {
	return nil, nil
}

func (HealthCheckRegistryStub) RunHealthCheck(name string) (accounting.HealthCheckResult, errors.Error) {
	return nil, nil
}

// Periodically report all registered metrics to a source (console, log, service)
type MetricsReporterStub struct{}

func (MetricsReporterStub) MetricRegistry() accounting.MetricRegistry {
	return MetricRegistryStub{}
}

func (MetricsReporterStub) Start(interval int64, unit time.Duration) {}

func (MetricsReporterStub) Stop() {}

func (MetricsReporterStub) Report() {}

func (MetricsReporterStub) RateUnit() time.Duration { return 0 }
