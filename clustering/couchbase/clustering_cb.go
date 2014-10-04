package clustering_cb

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/logging"
)

const _PREFIX = "couchbase:"
const _RESERVED_NAME = "couchbase"

// cbConfigStore implements clustering.ConfigurationStore
type cbConfigStore struct {
	adminUrl string
}

// create a cbConfigStore given the path to a couchbase instance
func NewConfigstore(path string) (clustering.ConfigurationStore, errors.Error) {
	if strings.HasPrefix(path, _PREFIX) {
		path = path[len(_PREFIX):]
	}
	enable_ns_server_shutdown()
	return &cbConfigStore{
		adminUrl: path,
	}, nil
}

// ns_server shutdown protocol: poll stdin and exit upon reciept of EOF
func enable_ns_server_shutdown() {
	go pollStdinForEOF()
}

func pollStdinForEOF() {
	reader := bufio.NewReader(os.Stdin)
	buf := make([]byte, 4)
	logging.Infop("pollEOF: About to start stdin polling")
	for {
		_, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				logging.Infop("Received EOF; Exiting...")
				os.Exit(0)
			}
			logging.Errorp("Unexpected error polling stdin",
				logging.Pair{"error", err},
			)
		}
	}
}

// Implement Stringer interface
func (c *cbConfigStore) String() string {
	return fmt.Sprintf("url=%v", c.adminUrl)
}

// Implement clustering.ConfigurationStore interface
func (c *cbConfigStore) Id() string {
	return c.URL()
}

func (c *cbConfigStore) URL() string {
	return "couchbase:" + c.adminUrl
}

func (c *cbConfigStore) ClusterNames() ([]string, errors.Error) {
	clusterIds := []string{}
	// TODO: implement Recipe:
	// Invoke curl http://localhost:8091/pools/
	// (More specifically, c.URL + "/pools"
	// Get the pools array; for each element, get the attribute called name and add its value to the clusterIds slice
	return clusterIds, nil
}

func (c *cbConfigStore) ClusterByName(name string) (clustering.Cluster, errors.Error) {
	// TODO: implement Recipe:
	// Invoke curl http://localhost:8091/pools/<cluster name>
	// (More specifically: c.URL + "/pools/" + name
	// Need to deal with: Not found.
	// If there is no such cluster <cluster name>
	var clusterConfig cbCluster
	clusterConfig.configStore = c
	// populate clusterConfig with stuff from REST request
	return &clusterConfig, nil
}

func (c *cbConfigStore) ConfigurationManager() clustering.ConfigurationManager {
	return c
}

// cbConfigStore also implements clustering.ConfigurationManager interface
func (c *cbConfigStore) ConfigurationStore() clustering.ConfigurationStore {
	return c
}

func (c *cbConfigStore) AddCluster(l clustering.Cluster) (clustering.Cluster, errors.Error) {
	// NOP. This is a read-only implementation
	return nil, nil
}

func (c *cbConfigStore) RemoveCluster(l clustering.Cluster) (bool, errors.Error) {
	// NOP. This is a read-only implementation
	return false, nil
}

func (c *cbConfigStore) RemoveClusterByName(name string) (bool, errors.Error) {
	// NOP. This is a read-only implementation
	return false, nil
}

func (c *cbConfigStore) GetClusters() ([]clustering.Cluster, errors.Error) {
	clusters := []clustering.Cluster{}
	// TODO: implement Recipe:
	// foreach name n in  c.ClusterNames(), add c.ClusterByName(n) to clusters
	return clusters, nil
}

// cbCluster implements clustering.Cluster
type cbCluster struct {
	configStore    clustering.ConfigurationStore `json:"-"`
	dataStore      datastore.Datastore           `json:"-"`
	acctStore      accounting.AccountingStore    `json:"-"`
	ClusterName    string                        `json:"name"`
	DatastoreURI   string                        `json:"datastore_uri"`
	ConfigstoreURI string                        `json:"configstore_uri"`
	AccountingURI  string                        `json:"accountstore_uri"`
	version        clustering.Version            `json:"-"`
	VersionString  string                        `json:"version"`
}

// Create a new cbCluster instance
func NewCluster(name string,
	version clustering.Version,
	configstore clustering.ConfigurationStore,
	datastore datastore.Datastore,
	acctstore accounting.AccountingStore) (clustering.Cluster, errors.Error) {
	c := makeCbCluster(name, version, configstore, datastore, acctstore)
	return c, nil
}

func makeCbCluster(name string,
	version clustering.Version,
	cs clustering.ConfigurationStore,
	ds datastore.Datastore,
	as accounting.AccountingStore) clustering.Cluster {
	cluster := cbCluster{
		configStore:    cs,
		dataStore:      ds,
		acctStore:      as,
		ClusterName:    name,
		DatastoreURI:   ds.URL(),
		ConfigstoreURI: cs.URL(),
		AccountingURI:  as.URL(),
		version:        version,
		VersionString:  version.String(),
	}
	return &cluster
}

// cbCluster implements Stringer interface
func (c *cbCluster) String() string {
	return getJsonString(c)
}

// cbCluster implements clustering.Cluster interface
func (c *cbCluster) ConfigurationStoreId() string {
	return c.configStore.Id()
}

func (c *cbCluster) Name() string {
	return c.ClusterName
}

func (c *cbCluster) QueryNodeNames() ([]string, errors.Error) {
	queryNodeNames := []string{}
	// TODO: implement Recipe:
	// Invoke "http://localhost:8091/pools/" + c.Name() + "/nodeServices"
	// add hostname + nodesExt.n1ql to queryNodeNames
	return queryNodeNames, nil
}

func (c *cbCluster) QueryNodeByName(name string) (clustering.QueryNode, errors.Error) {
	// TODO: implement Recipe:
	// Invoke "http://localhost:8091/pools/" + c.Name() + "/nodeServices"
	// check that name matches one of hostname + nodesExt.n1ql
	// if no match, return nil
	var queryNode cbQueryNodeConfig
	// if match,
	return &queryNode, nil
}

func (c *cbCluster) Datastore() datastore.Datastore {
	return c.dataStore
}

func (c *cbCluster) AccountingStore() accounting.AccountingStore {
	return c.acctStore
}

func (c *cbCluster) ConfigurationStore() clustering.ConfigurationStore {
	return c.configStore
}

func (c *cbCluster) Version() clustering.Version {
	if c.version == nil {
		c.version = clustering.NewVersion(c.VersionString)
	}
	return c.version
}

func (c *cbCluster) ClusterManager() clustering.ClusterManager {
	return c
}

// cbCluster implements clustering.ClusterManager interface
func (c *cbCluster) Cluster() clustering.Cluster {
	return c
}

func (c *cbCluster) AddQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	// NOP. This is a read-only implementation
	return nil, nil
}

func (c *cbCluster) RemoveQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	return c.RemoveQueryNodeByName(n.Name())
}

func (c *cbCluster) RemoveQueryNodeByName(name string) (clustering.QueryNode, errors.Error) {
	// NOP. This is a read-only implementation
	return nil, nil
}

func (z *cbCluster) GetQueryNodes() ([]clustering.QueryNode, errors.Error) {
	qryNodes := []clustering.QueryNode{}
	// TODO: implement recipe:
	// for each name n in c.QueryNodeNames(), add c.QueryNodeByName(n) to qryNodes
	return qryNodes, nil
}

// cbQueryNodeConfig implements clustering.QueryNode
type cbQueryNodeConfig struct {
	ClusterName      string                    `json:"cluster_name"`
	QueryNodeName    string                    `json:"name"`
	QueryEndpointURL string                    `json:"query_endpoint"`
	AdminEndpointURL string                    `json:"admin_endpoint"`
	ClusterRef       *cbCluster                `json:"-"`
	StandaloneRef    *clustering.StdStandalone `json:"-"`
	OptionsCL        *clustering.ClOptions     `json:"options"`
}

// cbQueryNodeConfig implements Stringer interface
func (c *cbQueryNodeConfig) String() string {
	return getJsonString(c)
}

// cbQueryNodeConfig implements clustering.QueryNode interface
func (c *cbQueryNodeConfig) Cluster() clustering.Cluster {
	return c.ClusterRef
}

func (c *cbQueryNodeConfig) Name() string {
	return c.QueryNodeName
}

func (c *cbQueryNodeConfig) QueryEndpoint() string {
	return c.QueryEndpointURL
}

func (c *cbQueryNodeConfig) ClusterEndpoint() string {
	return c.AdminEndpointURL
}

func (c *cbQueryNodeConfig) Standalone() clustering.Standalone {
	return c.StandaloneRef
}

func (c *cbQueryNodeConfig) Options() clustering.QueryNodeOptions {
	return c.OptionsCL
}

func getJsonString(i interface{}) string {
	serialized, _ := json.Marshal(i)
	s := bytes.NewBuffer(append(serialized, '\n'))
	return s.String()
}
