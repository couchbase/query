//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/couchbase/query/logging"
)

const (
	_VITALS_NODE_MEMORY_USAGE = "node.memory.usage"
	_VITALS_ACTIVE_REQUESTS   = "request.active.count"

	// the Query service only refreshes node.memory.usage in its vitals every 30 seconds so wait a little longer than
	// that to ensure the value reported was sampled after all requests had completed
	_NODE_QUOTA_SETTLE_WAIT = time.Second * 35
)

// whether the configuration enables the node quota for the Query node
func nodeQuotaEnabled() bool {
	if DB == nil {
		return false
	}
	switch v := DB.queryConfig["node-quota"].(type) {
	case float64:
		return v > 0
	case string:
		n, err := strconv.ParseFloat(v, 64)
		return err == nil && n > 0
	}
	return false
}

// issues a GET to /admin/vitals and returns the decoded response
func getVitals() (map[string]interface{}, error) {
	resp, err := doQueryGet("/admin/vitals", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Status: %v. %s", resp.Status, string(b))
	}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	var vitals map[string]interface{}
	if err = dec.Decode(&vitals); err != nil {
		return nil, fmt.Errorf("Failed to decode vitals: %v", err)
	}
	return vitals, nil
}

// extracts a non-negative integer field from the vitals
func vitalsUint(vitals map[string]interface{}, field string) (uint64, error) {
	v, ok := vitals[field]
	if !ok {
		return 0, fmt.Errorf("\"%s\" not present in vitals", field)
	}
	n, ok := v.(json.Number)
	if !ok {
		return 0, fmt.Errorf("\"%s\" in vitals is not a number: %v", field, v)
	}
	u, err := strconv.ParseUint(string(n), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("\"%s\" in vitals is invalid: %v", field, err)
	}
	return u, nil
}

// records the node-wide memory pool usage before any statements have been issued; failures are logged only
func captureNodeQuotaBaseline() {
	if !nodeQuotaEnabled() {
		return
	}
	vitals, err := getVitals()
	if err != nil {
		logging.Errorf("Node quota: failed to read vitals for baseline memory usage: %v", err)
		return
	}
	usage, err := vitalsUint(vitals, _VITALS_NODE_MEMORY_USAGE)
	if err != nil {
		logging.Errorf("Node quota: unable to determine baseline memory usage: %v", err)
		return
	}
	DB.nodeQuotaBase = &usage
	logging.Infof("Node quota: baseline node memory usage: %d bytes", usage)
}

// once all test requests have completed, checks that the node-wide memory pool usage has returned to its baseline;
// returns an error only if it has not despite there being no active requests
func checkNodeQuotaUsage() error {
	if !nodeQuotaEnabled() {
		return nil
	}
	if DB.nodeQuotaBase == nil {
		logging.Errorf("Node quota: no baseline memory usage was captured; skipping post-test memory usage check.")
		return nil
	}
	baseline := *DB.nodeQuotaBase

	logging.Infof("Node quota: waiting %v before checking node memory usage.", _NODE_QUOTA_SETTLE_WAIT)
	time.Sleep(_NODE_QUOTA_SETTLE_WAIT)

	vitals, err := getVitals()
	if err != nil {
		logging.Errorf("Node quota: failed to read vitals for post-test memory usage: %v", err)
		return nil
	}
	usage, err := vitalsUint(vitals, _VITALS_NODE_MEMORY_USAGE)
	if err != nil {
		logging.Errorf("Node quota: unable to determine post-test memory usage: %v", err)
		return nil
	}
	active, err := vitalsUint(vitals, _VITALS_ACTIVE_REQUESTS)
	if err != nil {
		logging.Errorf("Node quota: unable to determine the active request count: %v", err)
		return nil
	}
	logging.Infof("Node quota: post-test node memory usage: %d bytes (baseline: %d bytes), active requests: %d",
		usage, baseline, active)

	if active != 0 {
		logging.Warnf("Node quota: %d request(s) still active; skipping post-test memory usage check.", active)
		return nil
	}
	if usage != baseline {
		return fmt.Errorf("Node quota memory usage did not return to its baseline with no active requests: "+
			"%d bytes in use after the test vs. %d bytes before it (difference: %+d bytes).",
			usage, baseline, int64(usage)-int64(baseline))
	}
	return nil
}
