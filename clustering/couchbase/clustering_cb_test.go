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
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/couchbaselabs/go-couchbase"
	"github.com/couchbase/query/accounting/stub"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore/mock"
	_ "github.com/couchbase/query/logging/resolver"
)

const (
	couchbase_location = "localhost"
)

func TestCBClustering(t *testing.T) {
	if !couchbase_running(couchbase_location) {
		t.Skip("Couchbase not running - skipping test")
	}
	ds, err := mock.NewDatastore("mock:")
	as, err := accounting_stub.NewAccountingStore("stub:")
	cs, err := NewConfigstore("http://" + couchbase_location + ":8091")
	if err != nil {
		t.Fatalf("Error creating configstore: ", err)
	}
	version := clustering.NewVersion("0.7.0")

	fmt.Printf("Created config store %v\n\n", cs)

	cfm := cs.ConfigurationManager()

	cluster1, _ := NewCluster("cluster1", version, cs, ds, as)

	fmt.Printf("Creating cluster %v\n\n", cluster1)

	cluster1, err = cfm.AddCluster(cluster1)

	if err != nil {
		t.Fatalf("Error adding cluster: ", err)
	}

	_, no_such_cluster := cs.ClusterByName("no_such_cluster")
	if no_such_cluster == nil {
		t.Fatalf("Expected error retrieving configuration of non-existent cluster")
	}
	if no_such_cluster.Code() != 2040 && no_such_cluster.TranslationKey() != "admin.clustering.get_cluster_error" {
		t.Fatalf("Expected error code %d", 2010)
	}
	// There should be a cluster called "default" in the Couchbase installation:
	cluster1check, errCheck := cs.ClusterByName("default")
	if errCheck != nil {
		t.Fatalf("Unexpected Error retrieving cluster by name: ", errCheck)
	}

	fmt.Printf("Retrieved cluster: %v\n\n", cluster1check)

	cm := cs.ConfigurationManager()

	// Get all clusters. There should be at least one ("default")
	clusters, errCheck := cm.GetClusters()
	clusters_json, json_err := json.Marshal(clusters)
	if err != nil {
		t.Fatalf("Unexpected Error marshalling GetClusters: ", json_err)
	}

	fmt.Printf("Retrieved clusters: %s\n", string(clusters_json))
	if errCheck != nil {
		t.Fatalf("Unexpected Error retrieving all cluster configs: ", errCheck)
	}
	iterateClusters(clusters, t)
}

func iterateClusters(clusters []clustering.Cluster, t *testing.T) {
	for _, c := range clusters {
		queryNodeNames, errCheck := c.QueryNodeNames()
		if errCheck != nil {
			t.Fatalf("Unexpected Error retrieving query node names: ", errCheck)
		}
		for _, qn := range queryNodeNames {
			qryNode, errCheck := c.QueryNodeByName(qn)
			if errCheck != nil {
				t.Fatalf("Unexpected Error retrieving query node by name: ", errCheck)
			}
			json_node, json_err := json.Marshal(qryNode)
			if json_err != nil {
				t.Fatalf("Unexpected Error marshalling query node: ", json_err)
			}
			fmt.Printf("QueryNode=%s\n", string(json_node))
		}
		clm := c.ClusterManager()
		queryNodes, errCheck := clm.GetQueryNodes()
		if errCheck != nil {
			t.Fatalf("Unexpected Error retrieving query nodes: ", errCheck)
		}
		for _, qryNode := range queryNodes {
			json_node, json_err := json.Marshal(qryNode)
			if json_err != nil {
				t.Fatalf("Unexpected Error marshalling query node: ", json_err)
			}
			fmt.Printf("QueryNode=%s\n", string(json_node))
		}
	}
}

func couchbase_running(where string) bool {
	url_parts := []string{"http://", where, ":8091/"}
	_, err := couchbase.Connect(strings.Join(url_parts, ""))
	return err == nil
}
