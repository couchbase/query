//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package accounting_gm

import (
	"testing"
)

func TestGoMetrics(t *testing.T) {
	acctstore := NewAccountingStore()

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
