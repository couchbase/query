//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package accounting_gm

import (
	"testing"
)

func TestGoMetrics(t *testing.T) {
	acctstore, _ := NewAccountingStore()

	if acctstore == nil {
		t.Fatalf("Expected to create AccountingStore")
	}

	mr := acctstore.MetricRegistry()

	c := mr.Counter("my_counter")

	if c == nil {
		t.Fatalf("Expected to create Counter")
	}

	c.Inc(10)

	c2 := mr.Counter("my_counter")

	if c2.Count() != 10 {
		t.Fatalf("Expected counter value to be 10")
	}

	acctstore.MetricRegistry().Histogram("my_histogram")

	h := acctstore.MetricRegistry().Histogram("my_histogram")
	if h == nil {
		t.Fatalf("Expected to create a histogram")
	}

	h.Update(1)

	acctstore.MetricRegistry().Counter("request_count")
	acctstore.MetricRegistry().Counter("request_overall_time")
	acctstore.MetricRegistry().Meter("request_rate")
	acctstore.MetricRegistry().Histogram("response_count")
	acctstore.MetricRegistry().Timer("request_time")
}
