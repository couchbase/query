package clustering_cb

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/couchbaselabs/go-couchbase"
	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/logging"
	"github.com/couchbaselabs/query/server/http"
	"github.com/couchbaselabs/query/util"
)

const _PREFIX = "couchbase:"

///////// Notes about Couchbase implementation of Clustering API
//
// clustering_cb (this package) -> go-couchbase -> couchbase cluster
//
// pool is a synonym for cluster
//

// cbConfigStore implements clustering.ConfigurationStore
type cbConfigStore struct {
	adminUrl string
	cbConn   *couchbase.Client
}

// create a cbConfigStore given the path to a couchbase instance
func NewConfigstore(path string) (clustering.ConfigurationStore, errors.Error) {
	if strings.HasPrefix(path, _PREFIX) {
		path = path[len(_PREFIX):]
	}
	c, err := couchbase.Connect(path)
	if err != nil {
		return nil, errors.NewAdminConnectionError(err,
			fmt.Sprintf("Cannot connect to %s", path))
	}
	return &cbConfigStore{
		adminUrl: path,
		cbConn:   &c,
	}, nil
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
	return c.adminUrl
}

func (c *cbConfigStore) ClusterNames() ([]string, errors.Error) {
	clusterIds := []string{}
	// TODO: refresh pools (is it likely to go stale over lifetime of n1ql process?):
	for _, pool := range c.cbConn.Info.Pools {
		clusterIds = append(clusterIds, pool.Name)
	}
	return clusterIds, nil
}

func (c *cbConfigStore) ClusterByName(name string) (clustering.Cluster, errors.Error) {
	_, err := c.cbConn.GetPool(name)
	if err != nil {
		return nil, errors.NewAdminClusterConfigError(err, fmt.Sprintf("Cluster %s", name))
	}
	return &cbCluster{
		configStore:    c,
		ClusterName:    name,
		ConfigstoreURI: c.URL(),
		DatastoreURI:   c.URL(),
	}, nil
}

func (c *cbConfigStore) ConfigurationManager() clustering.ConfigurationManager {
	return c
}

// helper method to get all the services in a pool
func (c *cbConfigStore) getPoolServices(name string) (couchbase.PoolServices, errors.Error) {
	poolServices, err := c.cbConn.GetPoolServices(name)
	if err != nil {
		return poolServices, errors.NewAdminClusterConfigError(err, name)
	}
	return poolServices, nil
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
	// foreach name n in  c.ClusterNames(), add c.ClusterByName(n) to clusters
	clusterNames, _ := c.ClusterNames()
	for _, name := range clusterNames {
		cluster, err := c.ClusterByName(name)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

// cbCluster implements clustering.Cluster
type cbCluster struct {
	sync.Mutex
	configStore    clustering.ConfigurationStore `json:"-"`
	dataStore      datastore.Datastore           `json:"-"`
	acctStore      accounting.AccountingStore    `json:"-"`
	ClusterName    string                        `json:"name"`
	DatastoreURI   string                        `json:"datastore"`
	ConfigstoreURI string                        `json:"configstore"`
	AccountingURI  string                        `json:"accountstore"`
	version        clustering.Version            `json:"-"`
	VersionString  string                        `json:"version"`
	queryNodeNames []string                      `json:"-"`
	queryNodes     map[string]int                `json:"-"`
	poolSrvRev     int                           `json:"-"`
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
		queryNodeNames: []string{},
		poolSrvRev:     -999,
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
	// Get a handle of the go-couchbase connection:
	cbConn, ok := c.configStore.(*cbConfigStore)
	if !ok {
		return nil, errors.NewAdminConnectionError(nil,
			fmt.Sprintf("Cannot connect to %s", c.ConfigurationStoreId()))
	}
	poolServices, err := cbConn.getPoolServices(c.ClusterName)
	if err != nil {
		return queryNodeNames, err
	}
	// If the go-couchbase pool services rev matches the cluster's rev, return cluster's query node names:
	if poolServices.Rev == c.poolSrvRev {
		return c.queryNodeNames, nil
	}
	// If the rev numbers do not match, update the cluster's rev and query node data:
	queryNodeNames = []string{}
	queryNodes := make(map[string]int)
	for _, ns := range poolServices.NodesExt {
		n1qlPort := ns.Services["n1ql"]
		hostname := ns.Hostname

		if n1qlPort == 0 { // no n1ql service in this node
			continue
		}

		// TODO: check ns.ThisNode also
		if hostname == "" {
			hostname, _ = getHostnameFromURI(c.configStore.URL())
		}
		queryNodeNames = append(queryNodeNames, hostname)
		queryNodes[hostname] = n1qlPort
	}

	c.Lock()
	defer c.Unlock()
	c.queryNodeNames = queryNodeNames
	c.queryNodes = queryNodes
	c.poolSrvRev = poolServices.Rev
	return c.queryNodeNames, nil
}

func (c *cbCluster) QueryNodeByName(name string) (clustering.QueryNode, errors.Error) {
	qryNodeNames, err := c.QueryNodeNames()
	if err != nil {
		return nil, err
	}
	qryNodeName := ""
	for _, q := range qryNodeNames {
		if name == q {
			qryNodeName = q
			break
		}
	}
	if qryNodeName == "" {
		return nil, errors.NewAdminNodeConfigError(nil,
			fmt.Sprintf("No query node %s", name))
	}
	return &cbQueryNodeConfig{
		ClusterName:      c.Name(),
		QueryNodeName:    qryNodeName,
		QueryEndpointURL: http.GetServiceURL(qryNodeName, c.queryNodes[qryNodeName]),
		AdminEndpointURL: http.GetAdminURL(qryNodeName, c.queryNodes[qryNodeName]),
	}, nil
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

func (c *cbCluster) GetQueryNodes() ([]clustering.QueryNode, errors.Error) {
	qryNodes := []clustering.QueryNode{}
	// for each name n in c.QueryNodeNames(), add c.QueryNodeByName(n) to qryNodes
	names, err := c.QueryNodeNames()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		qryNode, err := c.QueryNodeByName(name)
		if err != nil {
			return nil, err
		}
		qryNodes = append(qryNodes, qryNode)
	}
	return qryNodes, nil
}

// cbQueryNodeConfig implements clustering.QueryNode
type cbQueryNodeConfig struct {
	ClusterName      string                    `json:"cluster"`
	QueryNodeName    string                    `json:"name"`
	QueryEndpointURL string                    `json:"queryEndpoint"`
	AdminEndpointURL string                    `json:"adminEndpoint"`
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

func getHostnameFromURI(uri string) (hostname string, err error) {
	tokens := strings.Split(uri, ":")
	name := strings.Split(tokens[1], "//")
	switch strings.ToLower(name[1]) {
	case "localhost", "127.0.0.1":
		hostname, err = util.ExternalIP()
	default:
		hostname, err = name[1], nil
	}
	return
}

func getJsonString(i interface{}) string {
	serialized, _ := json.Marshal(i)
	s := bytes.NewBuffer(append(serialized, '\n'))
	return s.String()
}

// ns_server shutdown protocol: poll stdin and exit upon reciept of EOF
func Enable_ns_server_shutdown() {
	go pollStdin()
}

func pollStdin() {
	reader := bufio.NewReader(os.Stdin)
	logging.Infop("pollEOF: About to start stdin polling")
	for {
		ch, err := reader.ReadByte()
		if err == io.EOF {
			logging.Infop("Received EOF; Exiting...")
			os.Exit(0)
		}
		if err != nil {
			logging.Errorp("Unexpected error polling stdin",
				logging.Pair{"error", err})
			os.Exit(1)
		}
		if ch == '\n' || ch == '\r' {
			logging.Infop("Received EOL; Exiting...")
			// TODO: "graceful" shutdown should be placed here
			os.Exit(0)
		}
	}
}
