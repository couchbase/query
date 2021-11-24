//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package clustering_stub

import (
	"testing"

	"github.com/couchbase/query/accounting/stub"
)

func TestStub(t *testing.T) {
	cs, _ := NewConfigurationStore()

	c, _ := cs.ClusterByName("cluster_id")
	if c != nil {
		t.Fatalf("Expected nil cluster")
	}

	cnames, _ := cs.ClusterNames()
	if len(cnames) != 1 {
		t.Fatalf("Expected length of cluster names to be one")
	}

	c, _ = cs.ClusterByName(cnames[0])
	if c == nil {
		t.Fatalf("Expected to retrieve cluster using name from ClusterNames()")
	}

	if c.ConfigurationStoreId() != cs.Id() {
		t.Fatalf("Cluster does not have expected configuration store ID")
	}

	qnames, _ := c.QueryNodeNames()
	if len(qnames) != 1 {
		t.Fatalf("Expected length of Query Node names to be one")
	}

	q, _ := c.QueryNodeByName(qnames[0])
	if q == nil {
		t.Fatalf("Expected to retrieve Query Node using name from QueryNodeNames()")
	}

	if q.Cluster().Name() != c.Name() {
		t.Fatalf("Unexpected cluster name in Query Node: %v", q.Cluster().Name())
	}

	as := q.Cluster().AccountingStore()

	if as.Id() != c.AccountingStore().Id() {
		t.Fatalf("Unexpected Accounting store id in Query Node: %v", as.Id())
	}

	mr := as.MetricRegistry()

	mr.Register("metric1", accounting_stub.GaugeStub{})

	g := mr.Get("metric1")
	if g != nil {
		t.Fatalf("MetricsRegsitryStub should not have any state")
	}

	gauges := mr.Gauges()

	for k, v := range gauges {
		t.Fatalf("Gauges map should be empty, found values: %v, %v", k, v)
	}
}
