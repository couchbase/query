//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

// Package accounting provides a common API for workload and monitoring data - metrics, statistics, events.

package accounting

import (
	"strings"
	"time"

	"github.com/couchbase/query/errors"
)

// AccountingStore represents a store for maintaining all accounting data (metrics, statistics, events)
type AccountingStore interface {
	Id() string                          // Id of this AccountingStore
	URL() string                         // URL to this AccountingStore
	MetricRegistry() MetricRegistry      // The MetricRegistry that this AccountingStore is managing
	MetricReporter() MetricReporter      // The MetricReporter that this AccountingStore is using
	Vitals() (interface{}, errors.Error) // The Vital Signs of the entity that this AccountingStore
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

// define metrics mnemonics
type CounterId int

const (
	REQUESTS CounterId = iota
	CANCELLED

	UNBOUNDED
	AT_PLUS
	SCAN_PLUS

	SELECTS
	UPDATES
	INSERTS
	DELETES

	TRANSACTIONS

	INDEX_SCANS
	PRIMARY_SCANS

	INVALID_REQUESTS

	REQUEST_TIME
	SERVICE_TIME
	TRANSACTION_TIME

	RESULT_COUNT
	RESULT_SIZE
	ERRORS
	WARNINGS
	MUTATIONS

	REQUESTS_250MS
	REQUESTS_500MS
	REQUESTS_1000MS
	REQUESTS_5000MS

	PREPARED

	AUDIT_REQUESTS_TOTAL
	AUDIT_REQUESTS_FILTERED
	AUDIT_ACTIONS
	AUDIT_ACTIONS_FAILED

	// unknown is always the last and does not have a corresponding name or metric
	UNKNOWN
)

// Define names for all the metrics we are interested in:
const (
	_REQUESTS  = "requests"
	_CANCELLED = "cancelled"

	_UNBOUNDED = "unbounded"
	_AT_PLUS   = "at_plus"
	_SCAN_PLUS = "scan_plus"

	_SELECTS = "selects"
	_UPDATES = "updates"
	_INSERTS = "inserts"
	_DELETES = "deletes"

	_TRANSACTIONS = "transactions"

	_INDEX_SCANS   = "index_scans"
	_PRIMARY_SCANS = "primary_scans"

	_INVALID_REQUESTS = "invalid_requests"

	_REQUEST_TIME     = "request_time"
	_SERVICE_TIME     = "service_time"
	_TRANSACTION_TIME = "transaction_time"

	_RESULT_COUNT = "result_count"
	_RESULT_SIZE  = "result_size"
	_ERRORS       = "errors"
	_WARNINGS     = "warnings"
	_MUTATIONS    = "mutations"

	_REQUESTS_250MS  = "requests_250ms"
	_REQUESTS_500MS  = "requests_500ms"
	_REQUESTS_1000MS = "requests_1000ms"
	_REQUESTS_5000MS = "requests_5000ms"

	PREPAREDS = "prepared" // Global for gometrics

	_AUDIT_REQUESTS_TOTAL    = "audit_requests_total"
	_AUDIT_REQUESTS_FILTERED = "audit_requests_filtered"
	_AUDIT_ACTIONS           = "audit_actions"
	_AUDIT_ACTIONS_FAILED    = "audit_actions_failed"

	REQUEST_RATE  = "request_rate"
	REQUEST_TIMER = "request_timer"
)

// please keep in sync with the mnemonics
var metricNames = []string{
	_REQUESTS,
	_CANCELLED,

	_UNBOUNDED,
	_AT_PLUS,
	_SCAN_PLUS,

	_SELECTS,
	_UPDATES,
	_INSERTS,
	_DELETES,

	_TRANSACTIONS,

	_INDEX_SCANS,
	_PRIMARY_SCANS,

	_INVALID_REQUESTS,

	_REQUEST_TIME,
	_SERVICE_TIME,
	_TRANSACTION_TIME,

	_RESULT_COUNT,
	_RESULT_SIZE,
	_ERRORS,
	_WARNINGS,
	_MUTATIONS,

	_REQUESTS_250MS,
	_REQUESTS_500MS,
	_REQUESTS_1000MS,
	_REQUESTS_5000MS,

	PREPAREDS,

	_AUDIT_REQUESTS_TOTAL,
	_AUDIT_REQUESTS_FILTERED,
	_AUDIT_ACTIONS,
	_AUDIT_ACTIONS_FAILED,
}

const (
	_DURATION_0MS    = 0 * time.Millisecond
	_DURATION_250MS  = 250 * time.Millisecond
	_DURATION_500MS  = 500 * time.Millisecond
	_DURATION_1000MS = 1000 * time.Millisecond
	_DURATION_5000MS = 5000 * time.Millisecond
)

// Map each duration to its metrics
var slowMetricsMap = map[time.Duration][]CounterId{

	// FIXME MB-15575 would like to use durations as duration windows, not overall
	_DURATION_5000MS: {REQUESTS_5000MS, REQUESTS_1000MS, REQUESTS_500MS, REQUESTS_250MS},
	_DURATION_1000MS: {REQUESTS_1000MS, REQUESTS_500MS, REQUESTS_250MS},
	_DURATION_500MS:  {REQUESTS_500MS, REQUESTS_250MS},
	_DURATION_250MS:  {REQUESTS_250MS},
	_DURATION_0MS:    {},
}

var acctstore AccountingStore
var counters []Counter = make([]Counter, len(metricNames))
var requestTimer Timer

// Use the give AccountingStore to create counters for all the metrics we are interested in:
func RegisterMetrics(acctStore AccountingStore) {
	acctstore = acctStore
	ms := acctstore.MetricRegistry()
	for id, name := range metricNames {
		counters[id] = ms.Counter(name)
	}

	requestTimer = ms.Timer(REQUEST_TIMER)
}

// Record request metrics
func RecordMetrics(request_time, service_time, transaction_time time.Duration,
	result_count int, result_size int,
	error_count int, warn_count int, stmt string, prepared bool,
	cancelled bool, index_scans int, primary_scans int, scanConsistency string) {

	if acctstore == nil {
		return
	}

	counters[REQUESTS].Inc(1)
	if cancelled {
		counters[CANCELLED].Inc(1)
	}
	switch scanConsistency {
	case "unbounded":
		counters[UNBOUNDED].Inc(1)
	case "scan_plus":
		counters[SCAN_PLUS].Inc(1)
	case "at_plus":
		counters[AT_PLUS].Inc(1)
	}

	counters[INDEX_SCANS].Inc(int64(index_scans))
	counters[PRIMARY_SCANS].Inc(int64(primary_scans))
	counters[REQUEST_TIME].Inc(int64(request_time))
	counters[SERVICE_TIME].Inc(int64(service_time))
	counters[TRANSACTION_TIME].Inc(int64(transaction_time))
	counters[RESULT_COUNT].Inc(int64(result_count))
	counters[RESULT_SIZE].Inc(int64(result_size))
	counters[ERRORS].Inc(int64(error_count))
	counters[WARNINGS].Inc(int64(warn_count))

	requestTimer.Update(request_time)

	if prepared {
		counters[PREPARED].Inc(1)
	}

	// Determine slow metrics based on request duration
	slowMetrics := slowMetricsMap[_DURATION_0MS]

	switch {
	case request_time >= _DURATION_5000MS:
		slowMetrics = slowMetricsMap[_DURATION_5000MS]
	case request_time >= _DURATION_1000MS:
		slowMetrics = slowMetricsMap[_DURATION_1000MS]
	case request_time >= _DURATION_500MS:
		slowMetrics = slowMetricsMap[_DURATION_500MS]
	case request_time >= _DURATION_250MS:
		slowMetrics = slowMetricsMap[_DURATION_250MS]
	default:
	}

	for _, durationMetric := range slowMetrics {
		counters[durationMetric].Inc(1)
	}

	// record the type of request if 0 errors
	if error_count == 0 {
		if t := requestType(stmt); t != UNKNOWN {
			counters[t].Inc(1)
		}
	}
}

func requestType(stmt string) CounterId {

	switch strings.ToUpper(stmt) {
	case "SELECT":
		return SELECTS
	case "UPDATE":
		return UPDATES
	case "INSERT":
		return INSERTS
	case "DELETE":
		return DELETES
	case "START_TRANSACTION":
		return TRANSACTIONS
	}
	return UNKNOWN
}

func UpdateCounter(id CounterId) {
	if acctstore == nil {
		return
	}
	counters[id].Inc(1)
}
