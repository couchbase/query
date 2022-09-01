//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package resolver

import (
	"strings"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/clustering/couchbase"
	"github.com/couchbase/query/clustering/stub"
	"github.com/couchbase/query/clustering/zookeeper"
	"github.com/couchbase/query/datastore"

	"github.com/couchbase/query/errors"
)

func NewConfigstore(uri string) (clustering.ConfigurationStore, errors.Error) {
	if strings.HasPrefix(uri, "http:") {
		return clustering_cb.NewConfigstore(uri)
	}

	if strings.HasPrefix(uri, "zookeeper:") {
		return clustering_zk.NewConfigstore(uri)
	}

	if strings.HasPrefix(uri, "stub:") {
		return clustering_stub.NewConfigurationStore()
	}
	return nil, errors.NewAdminInvalidURL("ConfigurationStore", uri)
}

func NewClusterConfig(uri string,
	clusterName string,
	version string,
	datastore datastore.Datastore,
	acctstore accounting.AccountingStore,
	cfgstore clustering.ConfigurationStore) (clustering.Cluster, errors.Error) {

	if strings.HasPrefix(uri, "zookeeper:") {
		v := clustering.NewVersion(version)
		return clustering_zk.NewCluster(clusterName, v, cfgstore, datastore, acctstore)
	}
	return nil, errors.NewAdminInvalidURL("ConfigurationStore", uri)
}

func NewQueryNodeConfig(uri string,
	version string,
	httpAddr string,
	opts clustering.ClOptions,
	datastore datastore.Datastore,
	acctstore accounting.AccountingStore,
	cfgstore clustering.ConfigurationStore) (clustering.QueryNode, errors.Error) {

	if strings.HasPrefix(uri, "zookeeper:") {
		v := clustering.NewVersion(version)
		s := clustering.NewStandalone(v, cfgstore, datastore, acctstore)
		return clustering_zk.NewQueryNode(httpAddr, s, &opts)
	}
	return nil, errors.NewAdminInvalidURL("ConfigurationStore", uri)
}
