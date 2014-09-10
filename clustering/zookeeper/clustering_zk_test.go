//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package clustering_zk

import (
	"fmt"
	"testing"

	"github.com/couchbaselabs/query/accounting/stub"
	"github.com/couchbaselabs/query/datastore/mock"
)

func TestZKClustering(t *testing.T) {
	cs, err := NewConfigstore("localhost:2181")
	ds, err := mock.NewDatastore("mock:")
	as, err := accounting_stub.NewAccountingStore("stub:")
	version := "0.7.0"
	version2 := "0.7.9"

	if err != nil {
		t.Errorf("Error creating configstore: ", err)
	}
	fmt.Printf("Created config store %v\n\n", cs)

	cfm := cs.ConfigurationManager()

	cluster1, _ := cfm.CreateCluster("cluster1", ds, as)

	fmt.Printf("Creating cluster %v\n\n", cluster1)

	cluster1, err = cfm.AddCluster(cluster1)

	if err != nil {
		t.Errorf("Error adding cluster: ", err)
	}

	cluster1check, errCheck := cs.ClusterByName("cluster1")
	if errCheck != nil {
		t.Errorf("Unexpected Error retrieving cluster by name: ", errCheck)
	}

	fmt.Printf("Retrieved cluster: %v\n\n", cluster1check)

	qn, err_qn := cluster1check.QueryNodeById("qryNode")
	if err_qn == nil {
		t.Errorf("Expected error getting query node: ", qn)
	}
	if qn != nil {
		t.Errorf("Unexpected query node! ", qn)
	}

	clusterMgr := cluster1check.ClusterManager()

	queryNode, _ := clusterMgr.CreateQueryNode(version, ":8093", ds, as)
	queryNode, err_qn = clusterMgr.AddQueryNode(queryNode)
	if err_qn != nil {
		t.Errorf("Unexpected error adding query node: ", err_qn)
	}
	fmt.Printf("Added query node %v to cluster %v\n\n", queryNode, cluster1)

	queryNode, _ = clusterMgr.CreateQueryNode(version, ":8094", ds, as)
	queryNode, err_qn = clusterMgr.AddQueryNode(queryNode)
	if err_qn != nil {
		t.Errorf("Unexpected error adding query node: ", err_qn)
	}
	fmt.Printf("Added query node %v to cluster %v\n\n", queryNode, cluster1)

	queryNode, _ = clusterMgr.CreateQueryNode(version, ":8095", ds, as)
	queryNode, err_qn = clusterMgr.AddQueryNode(queryNode)
	if err_qn != nil {
		t.Errorf("Unexpected error adding query node: ", err_qn)
	}
	fmt.Printf("Added query node %v to cluster %v\n\n", queryNode, cluster1)

	queryNode, _ = clusterMgr.CreateQueryNode(version2, ":8095", ds, as)
	queryNode, err_qn = clusterMgr.AddQueryNode(queryNode)
	if err_qn == nil {
		t.Errorf("Expected error adding query node: version incompatibility")
	}
	fmt.Printf("Version incompatibility adding query node 4: %v\n\n", err_qn)

	qryNodes, errT := clusterMgr.GetQueryNodes()
	if errT != nil {
		t.Errorf("Unexpected error getting query nodes: ", errT)
	}
	for i, qNode := range qryNodes {
		fmt.Printf("Query Node %d: configuration = %v\n\n", i, qNode)
	}

	clusters, errC := cfm.GetClusters()
	if errC != nil {
		t.Errorf("Unexpected error getting clusters: ", errC)
	}
	for c, cluster := range clusters {
		fmt.Printf("Cluster %d: configuration = %v\n\n", c, cluster)
	}

	for _, qNode := range qryNodes {
		qNode, errT = clusterMgr.RemoveQueryNodeById(qNode.Id())
		if errT != nil {
			t.Errorf("Unexpected error removing query node: ", errT)
		}
	}
	r, err := cfm.RemoveClusterById(cluster1check.Id())
	if err != nil {
		t.Errorf("Unexpected error removing cluster: ", err)
	}
	if r {
		fmt.Printf("Successfully removed cluster \n\n", cluster1)
	}

}
