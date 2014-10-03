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
	"runtime"
	"testing"

	"github.com/couchbaselabs/query/accounting/stub"
	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/datastore/mock"
)

func TestZKClustering(t *testing.T) {
	ds, err := mock.NewDatastore("mock:")
	as, err := accounting_stub.NewAccountingStore("stub:")
	cs, err := NewConfigstore("localhost:8091")
	version := clustering.NewVersion("0.7.0")
	version2 := clustering.NewVersion("0.7.9")
	stdCfg := clustering.NewStandalone(version, cs, ds, as)
	stdCfg2 := clustering.NewStandalone(version2, cs, ds, as)
	stdOpts := clustering.NewOptions(ds.URL(), cs.URL(), as.URL(), "default", false, false, true,
		runtime.NumCPU()<<16, runtime.NumCPU()<<6, 0, 0, ":8093", ":8094", "", false, "cluster1", "", "")

	fmt.Printf("%v %v %v\n", stdCfg, stdCfg2, stdOpts)

	if err != nil {
		t.Errorf("Error creating configstore: ", err)
	}
	fmt.Printf("Created config store %v\n\n", cs)

	cfm := cs.ConfigurationManager()

	cluster1, _ := NewCluster("cluster1", version, cs, ds, as)

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

}
