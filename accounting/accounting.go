//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Package accounting provides a common API for workload and monitoring data - metrics, statistics, events.

package accounting

import (
	"strings"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
)

// AccountingStore represents a store for maintaining all accounting data (metrics, statistics, events)
type AccountingStore interface {
	Id() string                                                       // Id of this AccountingStore
	URL() string                                                      // URL to this AccountingStore
	MetricRegistry() MetricRegistry                                   // The MetricRegistry that this AccountingStore is managing
	MetricReporter() MetricReporter                                   // The MetricReporter that this AccountingStore is using
	Vitals(util.DurationStyle) (map[string]interface{}, errors.Error) // The Vital Signs of the entity that this AccountingStore
	ExternalVitals(map[string]interface{}) map[string]interface{}     // Vitals comeing from outside
	NewCounter() Counter                                              // Create individual metrics
	NewGauge() Gauge
	NewMeter() Meter
	NewTimer() Timer
	NewHistogram() Histogram
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
	Update(int64) // Set the gauge's value
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
	Stop()             // Stop accruing data
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
	INDEX_SCANS_GSI
	PRIMARY_SCANS_GSI
	INDEX_SCANS_FTS
	PRIMARY_SCANS_FTS
	INDEX_SCANS_SEQ
	PRIMARY_SCANS_SEQ

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

	// user error metrics
	TIMEOUTS
	MEM_QUOTA_EXCEEDED_ERRORS
	UNAUTHORIZED_USERS
	BULK_GET_ERRORS
	CAS_MISMATCH_ERRORS
	TEMP_SPACE_ERRORS

	CURL_CALLS
	CURL_CALL_ERRORS

	// error count for sre alerts: https://issues.couchbase.com/browse/MB-58037
	USER_ERROR_COUNT
	SYSTEM_ERROR_COUNT

	SPILLS_ORDER

	// unknown is always the last and does not have a corresponding name or metric
	UNKNOWN
)

const (
	USED_MEMORY_HWM_ID CounterId = iota
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

	_INDEX_SCANS       = "index_scans"
	_PRIMARY_SCANS     = "primary_scans"
	_INDEX_SCANS_GSI   = "index_scans_gsi"
	_PRIMARY_SCANS_GSI = "primary_scans_gsi"
	_INDEX_SCANS_FTS   = "index_scans_fts"
	_PRIMARY_SCANS_FTS = "primary_scans_fts"
	_INDEX_SCANS_SEQ   = "index_scans_seq"
	_PRIMARY_SCANS_SEQ = "primary_scans_seq"

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

	// user error metrics
	_TIMEOUTS                  = "timeouts"
	_MEM_QUOTA_EXCEEDED_ERRORS = "mem_quota_exceeded_errors"
	_UNAUTHORIZED_USERS        = "unauthorized_users"
	_BULK_GET_ERRORS           = "bulk_get_errors"
	_CAS_MISMATCH_ERRORS       = "cas_mismatch_errors"
	_TEMP_SPACE_ERRORS         = "temp_space_errors"

	_CURL_CALLS       = "curl_calls"
	_CURL_CALL_ERRORS = "curl_call_errors"

	_USER_ERROR_COUNT   = "user_error_count"
	_SYSTEM_ERROR_COUNT = "engine_error_count"

	SPILLS_ORDER_STR = "spills_order"

	// gauges
	USED_MEMORY_HWM = "used_memory_hwm"
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
	_INDEX_SCANS_GSI,
	_PRIMARY_SCANS_GSI,
	_INDEX_SCANS_FTS,
	_PRIMARY_SCANS_FTS,
	_INDEX_SCANS_SEQ,
	_PRIMARY_SCANS_SEQ,

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

	_TIMEOUTS,
	_MEM_QUOTA_EXCEEDED_ERRORS,
	_UNAUTHORIZED_USERS,
	_BULK_GET_ERRORS,
	_CAS_MISMATCH_ERRORS,
	_TEMP_SPACE_ERRORS,

	_CURL_CALLS,
	_CURL_CALL_ERRORS,

	_USER_ERROR_COUNT,
	_SYSTEM_ERROR_COUNT,

	SPILLS_ORDER_STR,
}

var gaugeNames = []string{
	USED_MEMORY_HWM,
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

var errMetricsMap = map[errors.ErrorCode]CounterId{
	// timeouts
	errors.E_SERVICE_TIMEOUT:       TIMEOUTS,
	errors.W_INFER_TIMEOUT:         TIMEOUTS,
	errors.E_CB_INDEX_SCAN_TIMEOUT: TIMEOUTS,
	errors.E_SS_TIMEOUT:            TIMEOUTS,
	errors.E_SS_FETCH_WAIT_TIMEOUT: TIMEOUTS,

	// mem_quota_exceeded
	errors.E_MEMORY_QUOTA_EXCEEDED:             MEM_QUOTA_EXCEEDED_ERRORS,
	errors.E_NODE_QUOTA_EXCEEDED:               MEM_QUOTA_EXCEEDED_ERRORS,
	errors.E_TENANT_QUOTA_EXCEEDED:             MEM_QUOTA_EXCEEDED_ERRORS,
	errors.E_TRANSACTION_MEMORY_QUOTA_EXCEEDED: MEM_QUOTA_EXCEEDED_ERRORS,

	// unauthorized_users
	errors.E_SERVICE_TENANT_NOT_AUTHORIZED: UNAUTHORIZED_USERS,
	errors.E_DATASTORE_AUTHORIZATION:       UNAUTHORIZED_USERS,

	// bulk_get_errors
	errors.E_CB_BULK_GET: BULK_GET_ERRORS,

	// cas_mismatch_errors
	errors.E_CAS_MISMATCH: CAS_MISMATCH_ERRORS,

	// temp_space_errors
	errors.E_GSI_TEMP_FILE_SIZE: TEMP_SPACE_ERRORS,
	errors.E_TEMP_FILE_QUOTA:    TEMP_SPACE_ERRORS,
}

var acctstore AccountingStore
var counters []Counter = make([]Counter, len(metricNames))
var requestTimer Timer
var gauges []Gauge = make([]Gauge, len(gaugeNames))

// Use the given AccountingStore to create counters for all the metrics we are interested in:
func RegisterMetrics(acctStore AccountingStore) {
	acctstore = acctStore
	ms := acctstore.MetricRegistry()
	for id, name := range metricNames {
		counters[id] = ms.Counter(name)
	}

	requestTimer = ms.Timer(REQUEST_TIMER)

	for id, name := range gaugeNames {
		gauges[id] = ms.Gauge(name)
	}
}

// Record request metrics
func RecordMetrics(request_time, service_time, transaction_time time.Duration, result_count int, result_size int, error_count int,
	warn_count int, errs errors.Errors, stmt string, prepared bool, cancelled bool, index_scans int, primary_scans int,
	index_scans_gsi int, primary_scans_gsi int, index_scans_fts int, primary_scans_fts int, index_scans_seq int,
	primary_scans_seq int, scanConsistency string, used_memory uint64) {

	if acctstore == nil {
		return
	}

	if uint64(gauges[USED_MEMORY_HWM_ID].Value()) < used_memory {
		gauges[USED_MEMORY_HWM_ID].Update(int64(used_memory))
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
	counters[INDEX_SCANS_GSI].Inc(int64(index_scans_gsi))
	counters[PRIMARY_SCANS_GSI].Inc(int64(primary_scans_gsi))
	counters[INDEX_SCANS_FTS].Inc(int64(index_scans_fts))
	counters[PRIMARY_SCANS_FTS].Inc(int64(primary_scans_fts))
	counters[INDEX_SCANS_SEQ].Inc(int64(index_scans_seq))
	counters[PRIMARY_SCANS_SEQ].Inc(int64(primary_scans_seq))
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

	if error_count == 0 {
		// record the type of request if 0 errors
		if t := requestType(stmt); t != UNKNOWN {
			counters[t].Inc(1)
		}
	} else {
		toInc := map[CounterId]bool{}
		for _, err := range errs {
			for errCode, mid := range errMetricsMap {
				if _, pres := toInc[mid]; pres {
					continue
				}

				if err.HasCause(errCode) || err.HasICause(errCode) {
					// itself and cause path or icause path
					toInc[mid] = true
					counters[mid].Inc(1)
				}
			}

			// is the error an user or system error
			if errors.IsUserError(err.Code()) {
				counters[USER_ERROR_COUNT].Inc(1)
			} else if errors.IsSystemError(err.Code()) {
				counters[SYSTEM_ERROR_COUNT].Inc(1)
			}
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
