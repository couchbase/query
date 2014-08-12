//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

 Package clustering provides a common clustering abstraction for cluster
 configuration management.

 The main abstractions, and their relationships, are:
 	ConfigurationStore - used for storing all cluster configuration data.
	Cluster - Configuration common to a cluster. A cluster is just a set of Query Nodes, so all
		the Query Nodes belonging to the cluster will share this configuration.
	Query Node - Configuration for a single instance of the Query Engine. Provides sufficient
		information to uniquely identify, and interact with, a Query Engine instance.
The logical hierarchy is as follows:
	ConfigurationStore -> Clusters -> Query Nodes
*/
package clustering

import (
	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
)

type Mode string // A Query Node runs in a particular Mode

const (
	STANDALONE Mode = "standalone" // Query Node is running by itself, it is not part of a cluster
	CLUSTER    Mode = "cluster"    // Query Node is part of a cluster (could be a single node cluster)
)

// Version provides a abstraction of logical software version for Query Nodes;
// it could represent server build version or API version
type Version interface {
	string() string            // Return a string representation of the version
	Compatible(Version v) bool // Return true if the given Version is compatible with this Version
}

// ConfigurationStore represents a store for maintaining all cluster configuration data.
type ConfigurationStore interface {
	Id() string                                        // Id of this ConfigurationStore
	URL() string                                       // URL to this ConfigurationStore
	ClusterIds() ([]string, errors.Error)              // Ids of the Clusters in this ConfigurationStore
	ClusterNames() ([]string, errors.Error)            // Names of the Clusters in this ConfigurationStore
	ClusterById(id string) (Cluster, errors.Error)     // Find a Cluster in this ConfigurationStore using the Cluster's id
	ClusterByName(name string) (Cluster, errors.Error) // Find a Cluster in this ConfigurationStore using the Cluster's name
}

// Cluster is a named collection of Query Nodes. It is basically a single-level namespace for one or more Query Nodes.
// It also provides configuration common to all the Query Nodes in a cluster: Datastore, AccountingStore and ConfigurationStore.
type Cluster interface {
	ConfigurationStoreId() string                      // Id of the ConfigurationStore that contains this Cluster
	Id() string                                        // Id of this Cluster (unique within the ConfigurationStore)
	Name() string                                      // Name of this Cluster (unique within the ConfigurationStore)
	QueryNodeIds() ([]string, errors.Error)            // Ids of all the Query Nodes in this Cluster
	QueryNodeById(id string) (QueryNode, errors.Error) // Find a Query Node in this Cluster using the Query Node's id
	Datastore() datastore.Datastore                    // The Datastore used by all Query Nodes in the cluster
	AccountingStore() accounting.AccountingStore       // The AccountingStore used by all Query Nodes in the cluster
	ConfigurationStore() ConfigurationStore            // The ConfigurationStore used by all Query Nodes in the cluster
}

// QueryNode is the configuration for a single instance of a Query Engine.
type QueryNode interface {
	ClusterId() string                           // Id of the Cluster that this QueryNode belongs to
	Id() string                                  // Id of this QueryNode (unique within the cluster)
	Hostname() string                            // Name of the host that this QueryNode is running on
	IPAddress() string                           // IP address of the host this QueryNode is running on
	QueryEndpoint() string                       // Endpoint for serving N1QL queries
	ClusterEndpoint() string                     // Endpoint for serving cluster management commands
	Version() Version                            // Logical version of the software that the QueryNode is running
	Mode() Mode                                  // Running mode; this will be one of: standalone, cluster
	Datastore() datastore.Datastore              // The Datastore used by this Query Node
	AccountingStore() accounting.AccountingStore // The AccountingStore used by this Query Node
	ConfigurationStore() ConfigurationStore      // The ConfigurationStore used by this Query Node
}

// ConfigurationManager is the interface for managing cluster lifecycles -
// addition and removal of clusters from the ConfigurationStore.
type ConfigurationManager interface {
	// The ConfigurationStore that this ConfigurationManager is managing
	ConfigurationStore() ConfigurationStore

	// Add a cluster to the configuration
	// Possible reasons for error:
	//	- Configuration contains a Cluster with the same identity
	// Returns updated Cluster if no error (Cluster is now part of the ConfigurationStore)
	AddCluster(c Cluster) (Cluster, errors.Error)

	// Remove a cluster from the configuration
	// Possible reasons for error:
	//	- Cluster is not empty (contains one or more QueryNodes)
	// Returns true if no error (Cluster is no longer in the ConfigurationStore)
	RemoveCluster(c Cluster) (bool, errors.Error)

	// Remove the named cluster from the configuration
	// Possible reasons for error:
	//	- Configuration does not have a cluster with the given id
	//	- Cluster is not empty (contains one or more QueryNodes)
	RemoveClusterById(id string) (bool, errors.Error)

	// The clusters in the configuration
	GetClusters() ([]Cluster, errors.Error)
}

// ClusterManager is the interface the actions that can be done to a Cluster;
// it is intended to support the lifecycle of QueryNodes - addition and removal of
// QueryNodes from a Cluster.
type ClusterManager interface {
	// The Cluster that this ClusterManager is managing
	Cluster() Cluster

	// Add the given QueryNode to the Cluster
	// Possible reasons for error:
	//	- Cluster contains a QueryNode with the same identity
	//	- Version incompatibility
	//	- Given QueryNode is using a different Datastore
	//	- Given QueryNode is not in standalone mode
	// Returns the updated QueryNode if no error (cluster mode, connected to Cluster)
	AddQueryNode(n QueryNode) (QueryNode, errors.Error)

	// Remove the given QueryNode from the Cluster
	// Possible reasons for error:
	//	- Cluster does not contain the given QueryNode
	//	- Given QueryNode is running in standalone mode
	// Returns the updated QueryNode if no error (standalone mode, no cluster id)
	RemoveQueryNode(n QueryNode) (QueryNode, errors.Error)

	// Remove the QueryNode with the given id from the Cluster
	// Possible reasons for error:
	//	-- Cluster does not contain a QueryNode with the given id
	// Returns the updated QueryNode if no error (standalone mode, no cluster id)
	RemoveQueryNodeById(id string) (QueryNode, errors.Error)

	// Return the QueryNodes in the Cluster
	GetQueryNodes() ([]QueryNode, errors.Error)
}
