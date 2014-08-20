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
package accounting

import (
	"time"

	"github.com/couchbaselabs/query/errors"
)

// AccountingStore represents a store for maintaining all accounting data (metrics, statistics, events)
type AccountingStore interface {
	Id() string                               // Id of this AccountingStore
	URL() string                              // URL to this AccountingStore
	MetricRegistry() MetricRegistry           // The MetricRegistry that this AccountingStore is managing
	HealthCheckRegistry() HealthCheckRegistry // The HealthCheckRegistry that this AccountingStore is managing
}

// A point in time value
type TimeSeriesPoint struct {
	timestamp time.Time   // The time this TimeSeriesPoint was created
	value     interface{} // The value for this TimeSeriesPoint
}

// Metric types

// A Metric is a property that can be measured repeatedly and/or periodically
type Metric interface {
	History() []TimeSeriesPoint // The history of time series points captured for this metric
}

// Counter is an incrementing/decrementing count (#requests in a queue, #garbage collections)
type Counter interface {
	Metric
	Inc(amount int64) // Increment the counter by the given amount
	Dec(amount int64) // Decrement the counter by the given amount
	Count() int64     // Current Count value
}

// Gauge is an instantaneous measurement of a property (cpu load, response size)
type Gauge interface {
	Metric
	Value() float64 // The value of the Gauge
}

// Meter is a rate of change metric (queries per second, garbage collections per minute)
type Meter interface {
	Metric
	RateN(n int) float64 // n-minute moving average rate
	Mean() float64       // Mean throughput rate
	Mark(n int64)        // Mark the occurance of n events
	Count() int64        // The overall count of events
}

// Timer is a measurement of how long an activity took
type Timer interface {
	Metric
	Start()               // Start the timer
	Stop()                // Stop the timer
	Value() time.Duration // Current value of the timer
}

// Histogram provides summary statistics for a metric within a time window
type Histogram interface {
	Metric
	Percentile(n float64) float64      // The Nth percentile value (e.g. n = 50)
	Percentiles(n []float64) []float64 // The Nth percentiles values (e.g. n = {50, 75, 90, 95, 99, 99.9})
	StdDev() float64                   // The Standard Deviation of the values in the histogram
	Variance() float64                 // The Variance of the values in the histogram
}

// Aggregate provides aggregate statistics for a metric within a time window
type Aggregate interface {
	Metric
	Count() int64  // The number of values in the aggregate
	Max() float64  // The maximum value in the aggregate
	Mean() float64 // The mean value in the aggregate
	Min() float64  // The minimum value in the aggregate
	Sum() float64  // The sum of all values in the aggregate
}

// MetricsRegistry is the container for creating and maintaining Metrics
type MetricRegistry interface {
	// Register a metric with a name.
	// Possible reasons for error: name already in use
	Register(name string, metric Metric) errors.Error

	// Get the named metric or nil if no such name in use
	Get(name string) Metric

	// Unregister the metric with the given name
	// Possible reasons for error: no such name in use
	Unregister(name string) errors.Error

	// The following methods create or fetch a specific
	// type of metric with the given name
	Counter(name string) Counter
	Gauge(name string) Gauge
	Meter(name string) Meter
	Timer(name string) Timer
	Histogram(name string) Histogram
	Aggregate(name string) Aggregate

	Counters() map[string]Counter     // all registered counters
	Gauges() map[string]Gauge         // all registered gauges
	Meters() map[string]Meter         // all registered meters
	Timers() map[string]Timer         // all registered timers
	Histograms() map[string]Histogram // all registered histograms
	Aggregates() map[string]Aggregate // all registered aggregates
}

// A check that tests the status of an entity or compares a metric value against a
// configurable threshold.
type HealthCheck interface {
	// Perform the health check returning a healthy or unhealthy result
	// If an error occurs during the check an unhealthy result is returned
	// with the error.
	Check() (HealthCheckResult, errors.Error)
}

// The result of a health check; the possibilities are: healthy with optional message
// or unhealthy with an error message or error object.
type HealthCheckResult interface {
	IsHealthy() bool     // true if result is that the health check passed
	Message() string     // Return message for the result (or nil if no message)
	Error() errors.Error // Return error for the result (or nil if no error)
}

// HealthCheckRegistry is a centralized container for managing all health checks.
type HealthCheckRegistry interface {
	// Register a health check with the given name.
	// Reason for error: given name already in use.
	Register(name string, hc HealthCheck) errors.Error

	// Unregister the health check with the given name
	// Reasons for error: no such name in use
	Unregister(name string) errors.Error

	// Run all registered health checks returning a map of results
	RunHealthChecks() (map[string]HealthCheckResult, errors.Error)

	// Run the named health check returning the result or an error
	// if there is no health check registered with the given name
	RunHealthCheck(name string) (HealthCheckResult, errors.Error)
}

// Periodically report all registered metrics to a source (console, log, service)
type MetricsReporter interface {
	MetricRegistry() MetricRegistry // The Metrics Registry being reported on

	// Start reporting at the given interval and unit
	// (e.g. interval=10, unit=Second => report every 10 seconds)
	Start(interval int64, unit time.Duration)

	// Stop reporting
	Stop()

	// Report current values of all metrics in the registry
	Report()

	// The rate unit to use for reporting
	RateUnit() time.Duration
}
