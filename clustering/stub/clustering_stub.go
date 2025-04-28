//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package clustering_stub provides a stubbed implementation of the clustering package.

The Stubbed Configuration Store has one cluster, a Stubbed Cluster. This in turn has one
Query Node, a stubbed Query Node.

Hard coded names are used for Configuration Store Id and URL, Cluster Name and
Query Node Name, Query Endpoint and Cluster Endpoint.
*/
package clustering_stub

import (
	"net/http"

	"github.com/couchbase/query/accounting"
	accounting_stub "github.com/couchbase/query/accounting/stub"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

const (
	VERSION_STUB                     string = "VersionStub"
	CONFIGURATION_STORE_STUB_ID      string = "ConfigurationStoreStubID"
	CONFIGURATION_STORE_STUB_URL     string = "ConfigurationStoreStubURL"
	CLUSTER_STUB_ID                  string = "ClusterStubID"
	CLUSTER_STUB_NAME                string = "ClusterStubName"
	QUERY_NODE_STUB_ID               string = "QueryNodeStubID"
	QUERY_NODE_STUB_UUID             string = "QueryNodeStubUUID"
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

func (ConfigurationStoreStub) SetOptions(httpAddr, httpsAddr string, managed bool) errors.Error {
	return nil
}

func (ConfigurationStoreStub) ClusterNames() ([]string, errors.Error) {
	return []string{ClusterStub{}.Name()}, nil
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

func (ConfigurationStoreStub) Authorize(*http.Request, []clustering.Privilege) errors.Error {
	return nil
}

func (ConfigurationStoreStub) WhoAmI() (string, errors.Error) {
	return "", nil
}

func (ConfigurationStoreStub) State() (clustering.Mode, errors.Error) {
	return clustering.STANDALONE, nil
}

func (ConfigurationStoreStub) Cluster() (clustering.Cluster, errors.Error) {
	return nil, nil
}

// ClusterStub is a stub implementation of clustering.Cluster
// It has one Query Node, an instance of QueryNodeStub
type ClusterStub struct{}

func (ClusterStub) ConfigurationStoreId() string {
	return ConfigurationStoreStub{}.Id()
}

func (ClusterStub) Name() string {
	return CLUSTER_STUB_NAME
}

func (ClusterStub) QueryNodeNames() ([]string, errors.Error) {
	return []string{QueryNodeStub{}.Name()}, nil
}

func (ClusterStub) QueryNodeByName(name string) (clustering.QueryNode, errors.Error) {
	queryNode := QueryNodeStub{}
	if name != queryNode.Name() {
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

func (ClusterStub) Version() clustering.Version {
	return VersionStub{}
}

func (ClusterStub) ClusterManager() clustering.ClusterManager {
	return ClusterManagerStub{}
}

func (ClusterStub) Capability(name string) bool {
	return false
}

func (ClusterStub) Settings() (map[string]interface{}, errors.Error) {
	return nil, nil
}

func (ClusterStub) ReportEventAsync(event string) {
}

func (ClusterStub) NodeUUID(host string) (string, errors.Error) {
	return "", nil
}

func (ClusterStub) UUIDToHost(uuid string) (string, errors.Error) {
	return "", nil
}

// StandaloneStub is a stub implementation of clustering.Standalone
type StandaloneStub struct{}

func (StandaloneStub) Datastore() datastore.Datastore {
	return nil
}

func (StandaloneStub) AccountingStore() accounting.AccountingStore {
	return accounting_stub.AccountingStoreStub{}
}

func (StandaloneStub) ConfigurationStore() clustering.ConfigurationStore {
	return ConfigurationStoreStub{}
}

func (StandaloneStub) Version() clustering.Version {
	return VersionStub{}
}

// QueryNodeStub is a stub implementation of clustering.QueryNode
type QueryNodeStub struct{}

func (QueryNodeStub) Cluster() clustering.Cluster {
	return ClusterStub{}
}

func (QueryNodeStub) Name() string {
	return QUERY_NODE_STUB_ID
}

func (QueryNodeStub) NodeUUID() string {
	return QUERY_NODE_STUB_UUID
}

func (QueryNodeStub) Healthy() bool {
	return true
}

func (QueryNodeStub) QueryEndpoint() string {
	return QUERY_NODE_STUB_QUERY_ENDPOINT
}

func (QueryNodeStub) ClusterEndpoint() string {
	return QUERY_NODE_STUB_CLUSTER_ENDPOINT
}

func (QueryNodeStub) QuerySecure() string {
	return QUERY_NODE_STUB_QUERY_ENDPOINT
}

func (QueryNodeStub) ClusterSecure() string {
	return QUERY_NODE_STUB_CLUSTER_ENDPOINT
}

func (QueryNodeStub) Standalone() clustering.Standalone {
	return nil
}

func (QueryNodeStub) Options() clustering.QueryNodeOptions {
	return &clustering.ClOptions{}
}

// ConfigurationManagerStub is a stub implementation of clustering.ConfigurationManager
type ConfigurationManagerStub struct{}

func (ConfigurationManagerStub) ConfigurationStore() clustering.ConfigurationStore {
	return ConfigurationStoreStub{}
}

func (ConfigurationManagerStub) AddCluster(c clustering.Cluster) (clustering.Cluster, errors.Error) {
	return ClusterStub{}, nil
}

func (ConfigurationManagerStub) RemoveCluster(c clustering.Cluster) (bool, errors.Error) {
	return false, nil
}

func (ConfigurationManagerStub) RemoveClusterByName(name string) (bool, errors.Error) {
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

func (ClusterManagerStub) RemoveQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	return QueryNodeStub{}, nil
}

func (ClusterManagerStub) RemoveQueryNodeByName(name string) (clustering.QueryNode, errors.Error) {
	return QueryNodeStub{}, nil
}

func (ClusterManagerStub) GetQueryNodes() ([]clustering.QueryNode, errors.Error) {
	return []clustering.QueryNode{QueryNodeStub{}}, nil
}
