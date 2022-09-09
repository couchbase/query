//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Stub accounting package

package accounting_stub

import (
	"fmt"
	"net/http"
	"time"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/errors"
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

func (AccountingStoreStub) MetricReporter() accounting.MetricReporter {
	return MetricReporterStub{}
}

func (AccountingStoreStub) Vitals() (map[string]interface{}, errors.Error) {
	return nil, nil
}

func (AccountingStoreStub) ExternalVitals(vals map[string]interface{}) map[string]interface{} {
	return nil
}

func (AccountingStoreStub) NewCounter() accounting.Counter {
	return CounterStub{}
}

func (AccountingStoreStub) NewGauge() accounting.Gauge {
	return GaugeStub{}
}

func (AccountingStoreStub) NewMeter() accounting.Meter {
	return MeterStub{}
}

func (AccountingStoreStub) NewTimer() accounting.Timer {
	return TimerStub{}
}

func (AccountingStoreStub) NewHistogram() accounting.Histogram {
	return HistogramStub{}
}

// CounterStub is a stub implementation of Counter
type CounterStub struct{}

func (CounterStub) Inc(amount int64) {} // Nop

func (CounterStub) Dec(amount int64) {} // Nop

func (CounterStub) Count() int64 { return 0 }

func (CounterStub) Clear() {} // Nop

// GaugeStub is a stub implementation of Gauge
type GaugeStub struct{}

func (GaugeStub) Value() int64 { return 0 }

// MeterStub is a stub implementation of Meter
type MeterStub struct{}

func (MeterStub) Rate1() float64 { return 0.0 }

func (MeterStub) Rate5() float64 { return 0.0 }

func (MeterStub) Rate15() float64 { return 0.0 }

func (MeterStub) RateMean() float64 { return 0.0 }

func (MeterStub) Mark(n int64) {} // Nop

func (MeterStub) Count() int64 { return 0 }

func (MeterStub) Stop() {} // nop

// HistogramStub is a stub implementation of Histogram
type HistogramStub struct{}

func (HistogramStub) Percentile(n float64) float64 { return 0.0 }

func (HistogramStub) Percentiles(n []float64) []float64 { return make([]float64, len(n)) }

func (HistogramStub) StdDev() float64 { return 0.0 }

func (HistogramStub) Variance() float64 { return 0.0 }

func (HistogramStub) Clear() {} // Nop

func (HistogramStub) Count() int64 { return 0 }

func (HistogramStub) Max() int64 { return 0 }

func (HistogramStub) Mean() float64 { return 0.0 }

func (HistogramStub) Min() int64 { return 0 }

func (HistogramStub) Sum() int64 { return 0 }

func (HistogramStub) Update(int64) {} // Nop

// TimerStub is a stub implementation of Timer
type TimerStub struct{}

func (TimerStub) Percentile(n float64) float64 { return 0.0 }

func (TimerStub) Percentiles(n []float64) []float64 { return make([]float64, len(n)) }

func (TimerStub) StdDev() float64 { return 0.0 }

func (TimerStub) Variance() float64 { return 0.0 }

func (TimerStub) Clear() {} // Nop

func (TimerStub) Count() int64 { return 0 }

func (TimerStub) Max() int64 { return 0 }

func (TimerStub) Mean() float64 { return 0.0 }

func (TimerStub) Min() int64 { return 0 }

func (TimerStub) Sum() int64 { return 0 }

func (TimerStub) Update(time.Duration) {} // Nop

func (TimerStub) Rate1() float64 { return 0.0 }

func (TimerStub) Rate5() float64 { return 0.0 }

func (TimerStub) Rate15() float64 { return 0.0 }

func (TimerStub) RateMean() float64 { return 0.0 }

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

// Periodically report all registered metrics to a source (console, log, service)
type MetricReporterStub struct{}

func (MetricReporterStub) MetricRegistry() accounting.MetricRegistry {
	return MetricRegistryStub{}
}

func statsHandler(w http.ResponseWriter, req *http.Request) {
	// NOP: write empty stats body
	fmt.Fprintf(w, "{}")
}

func (MetricReporterStub) Start(interval int64, unit time.Duration) {
	http.HandleFunc("/query/stats/", statsHandler)
}

func (MetricReporterStub) Stop() {}

func (MetricReporterStub) Report() {}

func (MetricReporterStub) RateUnit() time.Duration { return 0 }
