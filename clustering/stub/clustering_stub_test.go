//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package clustering_stub

import "testing"

func TestStub(t *testing.T) {
	cs := NewConfigurationStore()

	c, _ := cs.ClusterById("cluster_id")
	if c != nil {
		t.Errorf("Expected nil cluster")
	}

	cnames, _ := cs.ClusterNames()
	if len(cnames) != 1 {
		t.Errorf("Expected length of cluster names to be one")
	}

	c, _ = cs.ClusterByName(cnames[0])
	if c == nil {
		t.Errorf("Expected to retrieve cluster using name from ClusterNames()")
	}

	if c.ConfigurationStoreId() != cs.Id() {
		t.Errorf("Cluster does not have expected configuration store ID")
	}

	qnames, _ := c.QueryNodeIds()
	if len(qnames) != 1 {
		t.Errorf("Expected length of Query Node names to be one")
	}

	q, _ := c.QueryNodeById(qnames[0])
	if q == nil {
		t.Errorf("Expected to retrieve Query Node using Id from QueryNodeIds()")
	}

	if q.ClusterId() != c.Id() {
		t.Errorf("Unexpected cluster id in Query Node: %v", q.ClusterId())
	}

}
