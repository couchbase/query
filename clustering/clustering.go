//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

type Mode string // A Query Node runs in a particular Mode

const (
	STARTING     Mode = "starting"     // Query Node is starting up
	STANDALONE   Mode = "standalone"   // Query Node is running by itself, it is not part of a cluster
	CLUSTERED    Mode = "clustered"    // Query Node is part of a cluster (could be a single node cluster)
	UNCLUSTERED  Mode = "unclustered"  // Query Node is not part of a cluster. Can serve queries
	DISCONNECTED Mode = "disconnected" // Query Node is disconnected from datastore. It cannot serve queries
	STOPPING     Mode = "stopping"     // Query Node is in the process of shutting down
)

type Privilege int

const (
	PRIV_READ      Privilege = 1 // read operations (e.g. retrieve node configuration)
	PRIV_SYS_ADMIN Privilege = 2 // system administrator operations (e.g. add node, reload ssl certificate)
)

// Version provides a abstraction of logical software version for Query Nodes;
// it could represent server build version or API version
type Version interface {
	String() string            // Return a string representation of the version
	Compatible(v Version) bool // Return true if the given Version is compatible with this Version
}

// ConfigurationStore represents a store for maintaining all cluster configuration data.
type ConfigurationStore interface {
	Id() string                                                       // Id of this ConfigurationStore
	URL() string                                                      // URL to this ConfigurationStore
	ClusterNames() ([]string, errors.Error)                           // Names of the Clusters in this ConfigurationStore
	ClusterByName(name string) (Cluster, errors.Error)                // Find a Cluster in this ConfigurationStore using the Cluster's name
	ConfigurationManager() ConfigurationManager                       // Get a ConfigurationManager for this ConfigurationStore
	Authorize(map[string]string, []Privilege) errors.Error            // Do authorization returning an error if unsuccessful
	WhoAmI() (string, errors.Error)                                   // The Id of the local node, if clustered
	State() (Mode, errors.Error)                                      // The clustering state of the local node
	Cluster() (Cluster, errors.Error)                                 // The cluster the local belongs to
	SetOptions(httpAddr, httpsAddr string, managed bool) errors.Error // Set options for the local ConfigurationStore
}

// Cluster is a named collection of Query Nodes. It is basically a single-level namespace for one or more Query Nodes.
// It also provides configuration common to all the Query Nodes in a cluster: Datastore, AccountingStore and ConfigurationStore.
type Cluster interface {
	ConfigurationStoreId() string                          // Id of the ConfigurationStore that contains this Cluster
	Name() string                                          // Name of this Cluster (unique within the ConfigurationStore)
	QueryNodeNames() ([]string, errors.Error)              // Names of all the Query Nodes in this Cluster
	QueryNodeByName(name string) (QueryNode, errors.Error) // Find a Query Node in this Cluster using the Query Node's id
	Datastore() datastore.Datastore                        // The Datastore used by all Query Nodes in the cluster
	AccountingStore() accounting.AccountingStore           // The AccountingStore used by all Query Nodes in the cluster
	ConfigurationStore() ConfigurationStore                // The ConfigurationStore used by all Query Nodes in the cluster
	Version() Version                                      // Logical version of the software that the QueryNodes in the cluster are running
	ClusterManager() ClusterManager                        // Get a ClusterManager for this Cluster
	Capability(string) bool                                // Check if cluster possesses a certain capability
	Settings() (map[string]interface{}, errors.Error)      // Get cluster wide settings

	ReportEventAsync(event string)            // Cluster's event logging
	NodeUUID(string) (string, errors.Error)   // Retrieve the UUID of the host
	UUIDToHost(string) (string, errors.Error) // Retrieve the hostname for the UUID
}

type Standalone interface {
	Datastore() datastore.Datastore              // The Datastore used by all Query Nodes in the cluster
	AccountingStore() accounting.AccountingStore // The AccountingStore used by all Query Nodes in the cluster
	ConfigurationStore() ConfigurationStore      // The ConfigurationStore used by all Query Nodes in the cluster
	Version() Version                            // Logical version of the software that the QueryNodes in the cluster are running
}

// QueryNode is the configuration for a single instance of a Query Engine.
type QueryNode interface {
	Cluster() Cluster          // The Cluster that this QueryNode belongs to
	Name() string              // Name of this QueryNode (unique within the cluster)
	NodeUUID() string          // UUID of the QueryNode
	QueryEndpoint() string     // Endpoint for serving N1QL queries
	ClusterEndpoint() string   // Endpoint for serving admin commands
	QuerySecure() string       // Endpoint for serving secure N1QL queries
	ClusterSecure() string     // Endpoint for serving secure admin commands
	Standalone() Standalone    // The QueryNode's configuration when unclustered
	Options() QueryNodeOptions // The command line options the query node was started with
}

type QueryNodeOptions interface {
	Datastore() string       // Datastore address
	Configstore() string     // Configstore address
	Accountingstore() string // Accountingstore address
	Namespace() string       //default namespace
	Readonly() bool          // Read-only mode
	Signature() bool         // Whether to provide Signature
	Metrics() bool           // Whether to provide Metrics
	RequestCap() int         // Max number of queued requests
	Threads() int            // Thread count
	OrderLimit() int         // Max LIMIT for ORDER BY clauses
	UpdateLimit() int        // Max LIMIT for data modification statements
	Http() string            // HTTP service address
	Https() string           // HTTPS service address
	Cafile() string          //HTTPS certificate file
	Certfile() string        // HTTPS certificate chain
	Keyfile() string         // HTTPS private key file
	Logger() string          // Name of Logger implementation
	Debug() bool             // Debug mode
	Cluster() string         // Name of the cluster to join
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
	RemoveClusterByName(name string) (bool, errors.Error)

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
	RemoveQueryNodeByName(name string) (QueryNode, errors.Error)

	// Return the QueryNodes in the Cluster
	GetQueryNodes() ([]QueryNode, errors.Error)
}

// Standard Version implementation - this can be used by all configstore implementations
type StdVersion struct {
	VersionString string
}

func NewVersion(version string) *StdVersion {
	return &StdVersion{
		VersionString: version,
	}
}

func (st *StdVersion) String() string {
	return st.VersionString
}

func (st *StdVersion) Compatible(v Version) bool {
	return v.String() == st.String()
}

// Standard QueryNodeOptions implementation - this can be used by all configstore implementations
type ClOptions struct {
	DatastoreURL string `json:"datastore"`
	CfgstoreURL  string `json:"configstore"`
	AcctstoreURL string `json:"acctstore"`
	NamespaceDef string `json:"namespace"`
	ReadMode     bool   `json:"readonly"`
	SignReqd     bool   `json:"readonly"`
	MetricsReqd  bool   `json:"metrics"`
	ReqCap       int    `json:"requestcap"`
	NumThreads   int    `json:"threads"`
	OrdLimit     int    `json:"orderlimit"`
	UpdLimit     int    `json:"updatelimit"`
	HttpAddr     string `json:"http"`
	HttpsAddr    string `json:"https"`
	LoggerImpl   string `json:"logger"`
	DebugFlag    bool   `json:"debug"`
	ClusterName  string `json:"cluster"`
	CaFile       string `json:"cafile"`
	CertFile     string `json:"certfile"`
	KeyFile      string `json:"keyfile"`
}

func (c *ClOptions) Datastore() string {
	return c.DatastoreURL
}

func (c *ClOptions) Logger() string {
	return c.LoggerImpl
}

func (c *ClOptions) Debug() bool {
	return c.DebugFlag
}

func (c *ClOptions) Cluster() string {
	return c.ClusterName
}

func (c *ClOptions) Configstore() string {
	return c.CfgstoreURL
}

func (c *ClOptions) Accountingstore() string {
	return c.AcctstoreURL
}

func (c *ClOptions) Namespace() string {
	return c.NamespaceDef
}

func (c *ClOptions) Readonly() bool {
	return c.ReadMode
}

func (c *ClOptions) Signature() bool {
	return c.SignReqd
}

func (c *ClOptions) Metrics() bool {
	return c.MetricsReqd
}

func (c *ClOptions) RequestCap() int {
	return c.ReqCap
}

func (c *ClOptions) Threads() int {
	return c.NumThreads
}

func (c *ClOptions) OrderLimit() int {
	return c.OrdLimit
}

func (c *ClOptions) UpdateLimit() int {
	return c.UpdLimit
}

func (c *ClOptions) Http() string {
	return c.HttpAddr
}

func (c *ClOptions) Https() string {
	return c.HttpsAddr
}

func (c *ClOptions) Cafile() string {
	return c.CaFile
}

func (c *ClOptions) Certfile() string {
	return c.CertFile
}

func (c *ClOptions) Keyfile() string {
	return c.KeyFile
}

func NewOptions(datastoreURL string, cfgstoreURL string, acctstoreURL string, namespace string,
	readOnly bool, signature bool, metrics bool, reqCap int, threads int, ordLim int, updLim int,
	http string, https string, loggerImpl string, debugFlag bool, clustName string, cafile string, certFile string,
	keyFile string) *ClOptions {
	return &ClOptions{
		DatastoreURL: datastoreURL,
		CfgstoreURL:  cfgstoreURL,
		AcctstoreURL: acctstoreURL,
		NamespaceDef: namespace,
		ReadMode:     readOnly,
		SignReqd:     signature,
		MetricsReqd:  metrics,
		ReqCap:       reqCap,
		NumThreads:   threads,
		OrdLimit:     ordLim,
		UpdLimit:     updLim,
		HttpAddr:     http,
		HttpsAddr:    https,
		LoggerImpl:   loggerImpl,
		DebugFlag:    debugFlag,
		ClusterName:  clustName,
		CaFile:       cafile,
		CertFile:     certFile,
		KeyFile:      keyFile,
	}
}

// Standard Standalone implementation - this can be used by all configstore implementations
type StdStandalone struct {
	configStore ConfigurationStore         `json:"-"`
	dataStore   datastore.Datastore        `json:"-"`
	acctStore   accounting.AccountingStore `json:"-"`
	Vers        Version                    `json:"version"`
}

func NewStandalone(version Version, cs ConfigurationStore, ds datastore.Datastore, as accounting.AccountingStore) *StdStandalone {
	return &StdStandalone{
		configStore: cs,
		dataStore:   ds,
		acctStore:   as,
		Vers:        version,
	}
}

func (st *StdStandalone) Datastore() datastore.Datastore {
	return st.dataStore
}

func (st *StdStandalone) AccountingStore() accounting.AccountingStore {
	return st.acctStore
}

func (st *StdStandalone) ConfigurationStore() ConfigurationStore {
	return st.configStore
}

func (st *StdStandalone) Version() Version {
	return st.Vers
}
