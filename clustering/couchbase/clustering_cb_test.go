//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package clustering_cb

import (
	"fmt"
	"testing"

	"github.com/couchbaselabs/query/accounting/stub"
	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/datastore/mock"
	_ "github.com/couchbaselabs/query/logging/resolver"
)

func TestCBClustering(t *testing.T) {
	ds, err := mock.NewDatastore("mock:")
	as, err := accounting_stub.NewAccountingStore("stub:")
	cs, err := NewConfigstore("http://127.0.0.1:8091")
	if err != nil {
		t.Errorf("Error creating configstore: ", err)
	}
	version := clustering.NewVersion("0.7.0")

	fmt.Printf("Created config store %v\n\n", cs)

	cfm := cs.ConfigurationManager()

	cluster1, _ := NewCluster("cluster1", version, cs, ds, as)

	fmt.Printf("Creating cluster %v\n\n", cluster1)

	cluster1, err = cfm.AddCluster(cluster1)

	if err != nil {
		t.Errorf("Error adding cluster: ", err)
	}

	// There should be a cluster called "default" in the Couchbase installation:
	cluster1check, errCheck := cs.ClusterByName("default")
	if errCheck != nil {
		t.Errorf("Unexpected Error retrieving cluster by name: ", errCheck)
	}

	fmt.Printf("Retrieved cluster: %v\n\n", cluster1check)

	cm := cs.ConfigurationManager()

	// Get all clusters. There should be at least one ("default")
	clusters, errCheck := cm.GetClusters()
	if errCheck != nil {
		t.Errorf("Unexpected Error retrieving all cluster configs: ", errCheck)
	}
	iterateClusters(clusters, t)
}

func iterateClusters(clusters []clustering.Cluster, t *testing.T) {
	for _, c := range clusters {
		fmt.Printf("Retrieved cluster: %v\n\n", c)
		queryNodeNames, errCheck := c.QueryNodeNames()
		if errCheck != nil {
			t.Errorf("Unexpected Error retrieving query node names: ", errCheck)
		}
		for _, qn := range queryNodeNames {
			fmt.Printf("QueryNodeName=%s\n", qn)
			qryNode, errCheck := c.QueryNodeByName(qn)
			if errCheck != nil {
				t.Errorf("Unexpected Error retrieving query node by name: ", errCheck)
			}
			fmt.Printf("QueryNode=%v\n", qryNode)
		}
		clm := c.ClusterManager()
		queryNodes, errCheck := clm.GetQueryNodes()
		if errCheck != nil {
			t.Errorf("Unexpected Error retrieving query nodes: ", errCheck)
		}
		for _, qryNode := range queryNodes {
			fmt.Printf("QueryNode=%v\n", qryNode)
		}
	}
}
