//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package clustering_cb

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/accounting/stub"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore/mock"
	_ "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/server"
)

var (
	couchbase_location = "localhost"
)

func init() {
	// For constructing URLs with raw IPv6 addresses- the IPv6 address
	// must be enclosed within ‘[‘ and ‘]’ brackets.
	couchbase_location = server.GetIP(true)
}

func TestCBClustering(t *testing.T) {
	if !couchbase_running(couchbase_location) {
		t.Skip("Couchbase not running - skipping test")
	}

	// Normally, cbauth would initialize from an environment parameter, CBAUTH_REVRPC_URL.
	// Here in the test environment we cannot set the parameter early enough to be caught by the initilization code,
	// so we direct cbauth to retry using a URL we provide.
	ok, err := cbauth.InternalRetryDefaultInitWithService("query", "localhost:8091", "Administrator", "password")
	if !ok {
		t.Fatalf("Unable to initialize cbauth: %s", err.Error())
	}

	ds, err := mock.NewDatastore("mock:")
	as, err := accounting_stub.NewAccountingStore("stub:")

	cs, err := NewConfigstore("http://" + couchbase_location + ":8091")
	if err != nil {
		t.Fatal("Error creating configstore: ", err)
	}
	version := clustering.NewVersion("0.7.0")

	fmt.Printf("Created config store %v\n\n", cs)

	cfm := cs.ConfigurationManager()

	cluster1, _ := NewCluster("cluster1", version, cs, ds, as)

	fmt.Printf("Creating cluster %v\n\n", cluster1)

	cluster1, err = cfm.AddCluster(cluster1)

	if err != nil {
		t.Fatal("Error adding cluster: ", err)
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
		t.Fatal("Unexpected Error retrieving cluster by name: ", errCheck)
	}

	fmt.Printf("Retrieved cluster: %v\n\n", cluster1check)

	cm := cs.ConfigurationManager()

	// Get all clusters. There should be at least one ("default")
	clusters, errCheck := cm.GetClusters()
	clusters_json, json_err := json.Marshal(clusters)
	if err != nil {
		t.Fatal("Unexpected Error marshalling GetClusters: ", json_err)
	}

	fmt.Printf("Retrieved clusters: %s\n", string(clusters_json))
	if errCheck != nil {
		t.Fatal("Unexpected Error retrieving all cluster configs: ", errCheck)
	}
	iterateClusters(clusters, t)
}

func iterateClusters(clusters []clustering.Cluster, t *testing.T) {
	for _, c := range clusters {
		queryNodeNames, errCheck := c.QueryNodeNames()
		if errCheck != nil {
			t.Fatal("Unexpected Error retrieving query node names: ", errCheck)
		}
		for _, qn := range queryNodeNames {
			qryNode, errCheck := c.QueryNodeByName(qn)
			if errCheck != nil {
				t.Fatal("Unexpected Error retrieving query node by name: ", errCheck)
			}
			if qryNode.QueryEndpoint() == "" {
				t.Logf("Query node %s does not have QueryEndpoint", qryNode.Name())
			}
			if qryNode.QuerySecure() == "" {
				t.Logf("Query node %s does not have QuerySecure", qryNode.Name())
			}
			json_node, json_err := json.Marshal(qryNode)
			if json_err != nil {
				t.Fatal("Unexpected Error marshalling query node: ", json_err)
			}
			fmt.Printf("QueryNode=%s\n", string(json_node))
		}
		clm := c.ClusterManager()
		queryNodes, errCheck := clm.GetQueryNodes()
		if errCheck != nil {
			t.Fatal("Unexpected Error retrieving query nodes: ", errCheck)
		}
		for _, qryNode := range queryNodes {
			json_node, json_err := json.Marshal(qryNode)
			if json_err != nil {
				t.Fatal("Unexpected Error marshalling query node: ", json_err)
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
