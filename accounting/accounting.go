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
	"strings"
	"time"

	"github.com/couchbase/query/errors"
)

// AccountingStore represents a store for maintaining all accounting data (metrics, statistics, events)
type AccountingStore interface {
	Id() string                               // Id of this AccountingStore
	URL() string                              // URL to this AccountingStore
	MetricRegistry() MetricRegistry           // The MetricRegistry that this AccountingStore is managing
	MetricReporter() MetricReporter           // The MetricReporter that this AccountingStore is using
	HealthCheckRegistry() HealthCheckRegistry // The HealthCheckRegistry that this AccountingStore is managing
}

// Metric types

// A Metric is a property that can be measured repeatedly and/or periodically
type Metric interface {
}

// Counter is an incrementing/decrementing count (#requests in a queue, #garbage collections)
type Counter interface {
	Metric
	Inc(amount int64) // Increment the counter by the given amount
	Dec(amount int64) // Decrement the counter by the given amount
	Count() int64     // Current Count value
	Clear()
}

// Gauge is an instantaneous measurement of a property (cpu load, response size)
type Gauge interface {
	Metric
	Value() int64 // The value of the Gauge
}

// Meter is a rate of change metric (queries per second, garbage collections per minute)
type Meter interface {
	Metric
	Rate1() float64    // 1-minute moving average rate
	Rate5() float64    // 5-minute moving average rate
	Rate15() float64   // 15-minute moving average rate
	RateMean() float64 // Mean throughput rate
	Mark(n int64)      // Mark the occurance of n events
	Count() int64      // The overall count of events
}

// Histogram provides summary statistics for a metric within a time window
type Histogram interface {
	Metric
	Clear()                            // Clear the histogram
	Count() int64                      // The number of values in the histogram
	Max() int64                        // The maximum value in the histogram
	Mean() float64                     // The mean value in the histogram
	Min() int64                        // The minimum value in the histogram
	Sum() int64                        // The sum of all values in the histogram
	Percentile(n float64) float64      // The Nth percentile value (e.g. n = 50)
	Percentiles(n []float64) []float64 // The Nth percentiles values (e.g. n = {50, 75, 90, 95, 99, 99.9})
	StdDev() float64                   // The Standard Deviation of the values in the histogram
	Variance() float64                 // The Variance of the values in the histogram
	Update(n int64)                    // Sample a new value
}

// Timer is a measurement of how long an activity took
type Timer interface {
	Metric
	Count() int64                      // The number of values in the timer
	Rate1() float64                    // 1-minute moving average rate
	Rate5() float64                    // 5-minute moving average rate
	Rate15() float64                   // 15-minute moving average rate
	RateMean() float64                 // Mean throughput rate
	Max() int64                        // The maximum value in the timer
	Mean() float64                     // The mean value in the timer
	Min() int64                        // The minimum value in the timer
	Sum() int64                        // The sum of all values in the timer
	Percentile(n float64) float64      // The Nth percentile value (e.g. n = 50)
	Percentiles(n []float64) []float64 // The Nth percentiles values (e.g. n = {50, 75, 90, 95, 99, 99.9})
	StdDev() float64                   // The Standard Deviation of the values in the timer
	Variance() float64                 // The Variance of the values in the timer
	Update(t time.Duration)            // Sample a new value
}

// MetricRegistry is the container for creating and maintaining Metrics
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

	Counters() map[string]Counter     // all registered counters
	Gauges() map[string]Gauge         // all registered gauges
	Meters() map[string]Meter         // all registered meters
	Timers() map[string]Timer         // all registered timers
	Histograms() map[string]Histogram // all registered histograms
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
type MetricReporter interface {
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

// Define names for all the metrics we are interested in:
const (
	REQUESTS         = "requests"
	SELECTS          = "selects"
	UPDATES          = "updates"
	INSERTS          = "inserts"
	DELETES          = "deletes"
	ACTIVE_REQUESTS  = "active_requests"
	QUEUED_REQUESTS  = "queued_requests"
	INVALID_REQUESTS = "invalid_requests"
	REQUEST_TIME     = "request_time"
	SERVICE_TIME     = "service_time"
	RESULT_COUNT     = "result_count"
	RESULT_SIZE      = "result_size"
	ERRORS           = "errors"
	WARNINGS         = "warnings"
	MUTATIONS        = "mutations"
	REQUESTS_250MS   = "requests_250ms"
	REQUESTS_500MS   = "requests_500ms"
	REQUESTS_1000MS  = "requests_1000ms"
	REQUESTS_5000MS  = "requests_5000ms"

	DURATION_250MS  = 250 * time.Millisecond
	DURATION_500MS  = 500 * time.Millisecond
	DURATION_1000MS = 1000 * time.Millisecond
	DURATION_5000MS = 5000 * time.Millisecond
)

var metricNames = []string{REQUESTS, SELECTS, UPDATES, INSERTS, DELETES, ACTIVE_REQUESTS,
	QUEUED_REQUESTS, INVALID_REQUESTS, REQUEST_TIME, SERVICE_TIME, RESULT_COUNT, RESULT_SIZE, ERRORS,
	REQUESTS_250MS, REQUESTS_500MS, REQUESTS_1000MS, REQUESTS_5000MS,
	WARNINGS, MUTATIONS}

// Use the give AccountingStore to create counters for all the metrics we are interested in:
func RegisterMetrics(acctstore AccountingStore) {
	ms := acctstore.MetricRegistry()
	for _, name := range metricNames {
		ms.Counter(name)
	}
}

func RecordMetrics(acctstore AccountingStore,
	request_time time.Duration, service_time time.Duration,
	result_count int, result_size int,
	error_count int, warn_count int, stmt string) {

	ms := acctstore.MetricRegistry()
	ms.Counter(REQUESTS).Inc(1)
	ms.Counter(REQUEST_TIME).Inc(int64(request_time))
	ms.Counter(SERVICE_TIME).Inc(int64(service_time))
	ms.Counter(RESULT_COUNT).Inc(int64(result_count))
	ms.Counter(RESULT_SIZE).Inc(int64(result_size))
	ms.Counter(ERRORS).Inc(int64(error_count))
	ms.Counter(WARNINGS).Inc(int64(warn_count))

	switch {
	case request_time >= DURATION_5000MS:
		ms.Counter(REQUESTS_5000MS).Inc(1)
	case request_time >= DURATION_1000MS:
		ms.Counter(REQUESTS_1000MS).Inc(1)
	case request_time >= DURATION_500MS:
		ms.Counter(REQUESTS_500MS).Inc(1)
	case request_time >= DURATION_250MS:
		ms.Counter(REQUESTS_250MS).Inc(1)
	default:
	}

	// Do not record the type of request if errors
	if error_count > 0 {
		return
	}

	stmt_tokens := strings.Split(strings.TrimSpace(stmt), " ")
	if len(stmt_tokens) < 1 {
		return
	}

	switch strings.ToLower(stmt_tokens[0]) {
	case "select":
		ms.Counter(SELECTS).Inc(1)
	case "update":
		ms.Counter(UPDATES).Inc(1)
	case "insert":
		ms.Counter(INSERTS).Inc(1)
	case "delete":
		ms.Counter(DELETES).Inc(1)
	}

}
