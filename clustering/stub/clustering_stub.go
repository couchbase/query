//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package clustering_stub provides a stubbed implementation of the clustering package.

The Stubbed Configuration Store has one cluster, a Stubbed Cluster. This in turn has one
Query Node, a stubbed Query Node.

Hard coded names are used for Configuration Store Id and URL, Cluster Id and Name and
Query Node Id, Name, Query Endpoint and Cluster Endpoint.

*/
package clustering_stub

import (
	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/accounting/stub"
	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
)

const (
	VERSION_STUB                     string = "VersionStub"
	CONFIGURATION_STORE_STUB_ID      string = "ConfigurationStoreStubID"
	CONFIGURATION_STORE_STUB_URL     string = "ConfigurationStoreStubURL"
	CLUSTER_STUB_ID                  string = "ClusterStubID"
	CLUSTER_STUB_NAME                string = "ClusterStubName"
	QUERY_NODE_STUB_ID               string = "QueryNodeStubID"
	QUERY_NODE_STUB_QUERY_ENDPOINT   string = "QueryNodeStubQueryEndPoint"
	QUERY_NODE_STUB_CLUSTER_ENDPOINT string = "QueryNodeStubClusterEndPoint"
)

// VersionStub is a stub implementation of clustering.Version
type VersionStub struct{}

func (VersionStub) String() string {
	return VERSION_STUB
}

func (VersionStub) Compatible(v clustering.Version) bool {
	return false
}

// ConfigurationStoreStub is a stub implementation of clustering.ConfigurationStore
// It has one cluster, an instance of ClusterStub.
type ConfigurationStoreStub struct{}

func (ConfigurationStoreStub) Id() string {
	return CONFIGURATION_STORE_STUB_ID
}

func (ConfigurationStoreStub) URL() string {
	return CONFIGURATION_STORE_STUB_URL
}

func (ConfigurationStoreStub) ClusterIds() ([]string, errors.Error) {
	return []string{ClusterStub{}.Id()}, nil
}

func (ConfigurationStoreStub) ClusterNames() ([]string, errors.Error) {
	return []string{ClusterStub{}.Name()}, nil
}

func (ConfigurationStoreStub) ClusterById(id string) (clustering.Cluster, errors.Error) {
	cluster := ClusterStub{}
	if id != cluster.Id() {
		return nil, nil
	}
	return cluster, nil
}

func (ConfigurationStoreStub) ClusterByName(name string) (clustering.Cluster, errors.Error) {
	cluster := ClusterStub{}
	if name != cluster.Name() {
		return nil, nil
	}
	return cluster, nil
}

func (ConfigurationStoreStub) ConfigurationManager() clustering.ConfigurationManager {
	return ConfigurationManagerStub{}
}

func NewConfigurationStore() (clustering.ConfigurationStore, errors.Error) {
	return ConfigurationStoreStub{}, nil
}

// ClusterStub is a stub implementation of clustering.Cluster
// It has one Query Node, an instance of QueryNodeStub
type ClusterStub struct{}

func (ClusterStub) ConfigurationStoreId() string {
	return ConfigurationStoreStub{}.Id()
}

func (ClusterStub) Id() string {
	return CLUSTER_STUB_ID
}

func (ClusterStub) Name() string {
	return CLUSTER_STUB_NAME
}

func (ClusterStub) QueryNodeIds() ([]string, errors.Error) {
	return []string{QueryNodeStub{}.Id()}, nil
}

func (ClusterStub) QueryNodeById(id string) (clustering.QueryNode, errors.Error) {
	queryNode := QueryNodeStub{}
	if id != queryNode.Id() {
		return nil, nil
	}
	return queryNode, nil
}

func (ClusterStub) Datastore() datastore.Datastore {
	return nil
}

func (ClusterStub) AccountingStore() accounting.AccountingStore {
	return accounting_stub.AccountingStoreStub{}
}

func (ClusterStub) ConfigurationStore() clustering.ConfigurationStore {
	return ConfigurationStoreStub{}
}

func (ClusterStub) ClusterManager() clustering.ClusterManager {
	return ClusterManagerStub{}
}

// QueryNodeStub is a stub implementation of clustering.QueryNode
type QueryNodeStub struct{}

func (QueryNodeStub) ClusterId() string {
	return CLUSTER_STUB_ID
}

func (QueryNodeStub) Id() string {
	return QUERY_NODE_STUB_ID
}

func (QueryNodeStub) QueryEndpoint() string {
	return QUERY_NODE_STUB_QUERY_ENDPOINT
}

func (QueryNodeStub) ClusterEndpoint() string {
	return QUERY_NODE_STUB_CLUSTER_ENDPOINT
}

func (QueryNodeStub) Version() clustering.Version {
	return VersionStub{}
}

func (QueryNodeStub) Mode() clustering.Mode {
	return clustering.STUB
}

func (QueryNodeStub) Datastore() datastore.Datastore {
	return nil
}

func (QueryNodeStub) AccountingStore() accounting.AccountingStore {
	return accounting_stub.AccountingStoreStub{}
}

func (QueryNodeStub) ConfigurationStore() clustering.ConfigurationStore {
	return ConfigurationStoreStub{}
}

// ConfigurationManagerStub is a stub implementation of clustering.ConfigurationManager
type ConfigurationManagerStub struct{}

func (ConfigurationManagerStub) ConfigurationStore() clustering.ConfigurationStore {
	return ConfigurationStoreStub{}
}

func (ConfigurationManagerStub) AddCluster(c clustering.Cluster) (clustering.Cluster, errors.Error) {
	return ClusterStub{}, nil
}

func (ConfigurationManagerStub) CreateCluster(id string, datastore datastore.Datastore, acctstore accounting.AccountingStore) (clustering.Cluster, errors.Error) {
	return ClusterStub{}, nil
}

func (ConfigurationManagerStub) RemoveCluster(c clustering.Cluster) (bool, errors.Error) {
	return false, nil
}

func (ConfigurationManagerStub) RemoveClusterById(id string) (bool, errors.Error) {
	return false, nil
}

func (ConfigurationManagerStub) GetClusters() ([]clustering.Cluster, errors.Error) {
	return []clustering.Cluster{ClusterStub{}}, nil
}

// ClusterManagerStub is a stub implementation of clustering.ClusterManager
type ClusterManagerStub struct{}

func (ClusterManagerStub) Cluster() clustering.Cluster {
	return ClusterStub{}
}

func (ClusterManagerStub) AddQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	return QueryNodeStub{}, nil
}

func (ClusterManagerStub) CreateQueryNode(version string, query_addr string, datastore datastore.Datastore, acctstore accounting.AccountingStore) (clustering.QueryNode, errors.Error) {
	return QueryNodeStub{}, nil
}

func (ClusterManagerStub) RemoveQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	return QueryNodeStub{}, nil
}

func (ClusterManagerStub) RemoveQueryNodeById(id string) (clustering.QueryNode, errors.Error) {
	return QueryNodeStub{}, nil
}

func (ClusterManagerStub) GetQueryNodes() ([]clustering.QueryNode, errors.Error) {
	return []clustering.QueryNode{QueryNodeStub{}}, nil
}
