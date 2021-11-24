//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package clustering_zk

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/couchbase/query/accounting/stub"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore/mock"
	"github.com/samuel/go-zookeeper/zk"

	"github.com/couchbase/query/server"
)

func TestZKClustering(t *testing.T) {
	// When creating a new configstore we call zookeeper.connect which
	// uses a url. Hence for IPv6 this value needs to be changed to [::1]
	localhost := server.GetIP(true) + ":2181"
	cs, err := NewConfigstore(localhost)
	ds, err := mock.NewDatastore("mock:")
	as, err := accounting_stub.NewAccountingStore("stub:")
	version := clustering.NewVersion("0.7.0")
	version2 := clustering.NewVersion("0.7.9")
	stdCfg := clustering.NewStandalone(version, cs, ds, as)
	stdCfg2 := clustering.NewStandalone(version2, cs, ds, as)
	stdOpts := clustering.NewOptions(ds.URL(), cs.URL(), as.URL(), "default", false, false, true,
		runtime.NumCPU()<<16, runtime.NumCPU()<<6, 0, 0, ":8093", ":8094", "", false, "cluster1", "", "", "")

	if !zookeeper_running() {
		t.Skip("Zookeeper not running - skipping test")
	}

	if err != nil {
		t.Fatal("Error creating configstore: ", err)
	}
	fmt.Printf("Created config store %v\n\n", cs)

	cfm := cs.ConfigurationManager()

	cluster1, _ := NewCluster("cluster1", version, cs, ds, as)

	fmt.Printf("Creating cluster %v\n\n", cluster1)

	cluster1, err = cfm.AddCluster(cluster1)

	if err != nil {
		t.Fatal("Error adding cluster: ", err)
	}

	cluster1check, errCheck := cs.ClusterByName("cluster1")
	if errCheck != nil {
		t.Fatal("Unexpected Error retrieving cluster by name: ", errCheck)
	}

	fmt.Printf("Retrieved cluster: %v\n\n", cluster1check)

	qn, err_qn := cluster1check.QueryNodeByName("qryNode")
	if err_qn == nil {
		t.Fatal("Expected error getting query node: ", qn)
	}
	if qn != nil {
		t.Fatal("Unexpected query node! ", qn)
	}

	clusterMgr := cluster1check.ClusterManager()

	queryNode, _ := NewQueryNode(":8093", stdCfg, stdOpts)
	queryNode, err_qn = clusterMgr.AddQueryNode(queryNode)
	if err_qn != nil {
		t.Fatal("Unexpected error adding query node: ", err_qn)
	}
	fmt.Printf("Added query node %v to cluster %v\n\n", queryNode, cluster1)

	queryNode, _ = NewQueryNode(":8094", stdCfg, stdOpts)
	queryNode, err_qn = clusterMgr.AddQueryNode(queryNode)
	if err_qn != nil {
		t.Fatal("Unexpected error adding query node: ", err_qn)
	}
	fmt.Printf("Added query node %v to cluster %v\n\n", queryNode, cluster1)

	queryNode, _ = NewQueryNode(":8095", stdCfg, stdOpts)
	queryNode, err_qn = clusterMgr.AddQueryNode(queryNode)
	if err_qn != nil {
		t.Fatal("Unexpected error adding query node: ", err_qn)
	}
	fmt.Printf("Added query node %v to cluster %v\n\n", queryNode, cluster1)

	queryNode, _ = NewQueryNode(":8095", stdCfg2, stdOpts)
	queryNode, err_qn = clusterMgr.AddQueryNode(queryNode)
	if err_qn == nil {
		t.Fatalf("Expected error adding query node: version incompatibility")
	}
	fmt.Printf("Version incompatibility adding query node 4: %v\n\n", err_qn)

	qryNodes, errT := clusterMgr.GetQueryNodes()
	if errT != nil {
		t.Fatal("Unexpected error getting query nodes: ", errT)
	}
	for i, qNode := range qryNodes {
		fmt.Printf("Query Node %d: configuration = %v\n\n", i, qNode)
	}

	clusters, errC := cfm.GetClusters()
	if errC != nil {
		t.Fatal("Unexpected error getting clusters: ", errC)
	}
	for c, cluster := range clusters {
		fmt.Printf("Cluster %d: configuration = %v\n\n", c, cluster)
	}

	for _, qNode := range qryNodes {
		qNode, errT = clusterMgr.RemoveQueryNodeByName(qNode.Name())
		if errT != nil {
			t.Fatal("Unexpected error removing query node: ", errT)
		}
	}
	r, err := cfm.RemoveClusterByName(cluster1check.Name())
	if err != nil {
		t.Fatal("Unexpected error removing cluster: ", err)
	}
	if r {
		fmt.Printf("Successfully removed cluster %v\n\n", cluster1)
	}

}

func zookeeper_running() bool {
	// For constructing URLs with raw IPv6 addresses- the IPv6 address
	// must be enclosed within ‘[‘ and ‘]’ brackets.
	c, _, err1 := zk.Connect([]string{server.GetIP(true)}, time.Second) //*10)
	_, _, _, err2 := c.ChildrenW("/")
	return err1 == nil && err2 == nil
}
