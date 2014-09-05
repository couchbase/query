package clustering_zk

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/samuel/go-zookeeper/zk"
)

const _PREFIX = "zookeeper:"
const _RESERVED_NAME = "zookeeper"

type zkVersion struct {
	VersionString string
}

func NewVersion(version string) *zkVersion {
	return &zkVersion{
		VersionString: version,
	}
}

func (z *zkVersion) String() string {
	return z.VersionString
}

func (z *zkVersion) Compatible(v clustering.Version) bool {
	return v.String() == z.String()
}

type zkConfigStore struct {
	conn *zk.Conn
	url  string
}

func NewConfigstore(path string) (clustering.ConfigurationStore, errors.Error) {
	if strings.HasPrefix(path, _PREFIX) {
		path = path[len(_PREFIX):]
	}
	zks := strings.Split(path, ",")
	conn, _, err := zk.Connect(zks, time.Second)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	return &zkConfigStore{
		conn: conn,
		url:  path,
	}, nil
}

func (z *zkConfigStore) String() string {
	return fmt.Sprintf("url=%v", z.url)
}

func (z *zkConfigStore) Id() string {
	return z.URL()
}

func (z *zkConfigStore) URL() string {
	return "zookeeper:" + z.url
}

func (z *zkConfigStore) ClusterIds() ([]string, errors.Error) {
	clusterIds := []string{}
	nodes, _, err := z.conn.Children("/")
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	for _, name := range nodes {
		clusterIds = append(clusterIds, name)
	}
	return clusterIds, nil
}

func (z *zkConfigStore) ClusterNames() ([]string, errors.Error) {
	return z.ClusterIds()
}

func (z *zkConfigStore) ClusterById(id string) (clustering.Cluster, errors.Error) {
	data, _, err := z.conn.Get("/" + id)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	var clusterConfig zkCluster
	err = json.Unmarshal(data, &clusterConfig)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	clusterConfig.configStore = z
	return &clusterConfig, nil
}

func (z *zkConfigStore) ClusterByName(name string) (clustering.Cluster, errors.Error) {
	return z.ClusterById(name)
}

func (z *zkConfigStore) ConfigurationManager() clustering.ConfigurationManager {
	return z
}

func (z *zkConfigStore) ConfigurationStore() clustering.ConfigurationStore {
	return z
}

func (z *zkConfigStore) AddCluster(c clustering.Cluster) (clustering.Cluster, errors.Error) {
	flags := int32(0)
	acl := zk.WorldACL(zk.PermAll)
	clusterBytes, err := json.Marshal(c)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	_, err = z.conn.Create("/"+c.Id(), clusterBytes, flags, acl)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	return c, nil
}

func (z *zkConfigStore) RemoveCluster(c clustering.Cluster) (bool, errors.Error) {
	return z.RemoveClusterById(c.Id())
}

func (z *zkConfigStore) RemoveClusterById(id string) (bool, errors.Error) {
	err := z.conn.Delete("/"+id, 0)
	if err != nil {
		return false, errors.NewError(err, "")
	} else {
		return true, nil
	}

}

func (z *zkConfigStore) GetClusters() ([]clustering.Cluster, errors.Error) {
	clusters := []clustering.Cluster{}
	nodes, _, err := z.conn.Children("/")
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	for _, name := range nodes {
		if name == _RESERVED_NAME {
			continue
		}
		data, _, err := z.conn.Get("/" + name)
		if err != nil {
			return nil, errors.NewError(err, "")
		}
		cluster := &zkCluster{}
		err = json.Unmarshal(data, cluster)
		if err != nil {
			return nil, errors.NewError(err, "")
		}
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

type zkCluster struct {
	configStore    clustering.ConfigurationStore `json:"-"`
	dataStore      datastore.Datastore           `json:"-"`
	acctStore      accounting.AccountingStore    `json:"-"`
	ClusterName    string                        `json:"name"`
	DatastoreURI   string                        `json:"datastore_uri"`
	ConfigstoreURI string                        `json:"configstore_uri"`
	AccountingURI  string                        `json:"accountstore_uri"`
}

func NewCluster(Name string, cs clustering.ConfigurationStore, ds datastore.Datastore, as accounting.AccountingStore) clustering.Cluster {
	cluster := zkCluster{
		configStore:    cs,
		dataStore:      ds,
		acctStore:      as,
		ClusterName:    Name,
		DatastoreURI:   ds.URL(),
		ConfigstoreURI: cs.URL(),
		AccountingURI:  as.URL(),
	}
	return &cluster
}

func (z *zkCluster) String() string {
	return fmt.Sprintf("name=%v, configstoreURI=%v, datastoreURI=%v, accountingURI=%v", z.ClusterName, z.ConfigstoreURI, z.DatastoreURI, z.AccountingURI)
}

func (z *zkCluster) ConfigurationStoreId() string {
	return z.configStore.Id()
}
func (z *zkCluster) Id() string {
	return z.ClusterName
}
func (z *zkCluster) Name() string {
	return z.Id()
}

func getConfigStoreImplementation(z *zkCluster) (impl *zkConfigStore, ok bool) {
	impl, ok = z.configStore.(*zkConfigStore)
	return
}

func (z *zkCluster) QueryNodeIds() ([]string, errors.Error) {
	queryNodeNames := []string{}
	impl, ok := getConfigStoreImplementation(z)
	if !ok {
		return nil, errors.NewWarning(fmt.Sprintf("Unable to get connection to zookeeper at %s", z.ConfigurationStoreId()))
	}
	nodes, _, err := impl.conn.Children("/" + z.ClusterName)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	for _, name := range nodes {
		queryNodeNames = append(queryNodeNames, name)
	}
	return queryNodeNames, nil
}

func (z *zkCluster) QueryNodeById(id string) (clustering.QueryNode, errors.Error) {
	impl, ok := getConfigStoreImplementation(z)
	if !ok {
		return nil, errors.NewWarning(fmt.Sprintf("Unable to get connection to zookeeper at %s", z.ConfigurationStoreId()))
	}
	data, _, err := impl.conn.Get("/" + z.ClusterName + "/" + id)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	var queryNode zkQueryNodeConfig
	err = json.Unmarshal(data, &queryNode)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	return &queryNode, nil
}

func (z *zkCluster) Datastore() datastore.Datastore {
	return z.dataStore
}

func (z *zkCluster) AccountingStore() accounting.AccountingStore {
	return z.acctStore
}

func (z *zkCluster) ConfigurationStore() clustering.ConfigurationStore {
	return z.configStore
}

func (z *zkCluster) ClusterManager() clustering.ClusterManager {
	return z
}

func (z *zkCluster) Cluster() clustering.Cluster {
	return z
}

func (z *zkCluster) AddQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	// Get connection to actual config store backend
	impl, ok := getConfigStoreImplementation(z)
	if !ok {
		return nil, errors.NewWarning(fmt.Sprintf("Unable to get connection to zookeeper at %s", z.ConfigurationStoreId()))
	}
	// Add entry for query node: ephemeral node
	flags := int32(zk.FlagEphemeral)
	acl := zk.WorldACL(zk.PermAll) // TODO: credentials - expose in the API
	key := "/" + z.Id() + "/" + n.Id()
	value, err := json.Marshal(n)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	// Check that query node has compatible backend connections:
	if n.Datastore().URL() != z.DatastoreURI {
		return nil, errors.NewWarning(fmt.Sprintf("Failed to add Query Node %v: incompatible datastore with cluster %s", n, z.DatastoreURI))
	}
	if n.ConfigurationStore().URL() != z.ConfigstoreURI {
		return nil, errors.NewWarning(fmt.Sprintf("Failed to add Query Node %v: incompatible configstore with cluster %s", n, z.ConfigstoreURI))
	}
	// Check that query node is version compatible with the cluster:
	qryNodes, err := z.GetQueryNodes()
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	if len(qryNodes) > 0 {
		v := qryNodes[0].Version()
		if !n.Version().Compatible(v) {
			return nil, errors.NewWarning(fmt.Sprintf("Failed to add Query Node %v: not version compatible with cluster (%v)", n, v))
		}
	}
	// query node has passed checks (backend connections and version compatibility)
	_, err = impl.conn.Create(key, value, flags, acl)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	return n, nil
}

func (z *zkCluster) RemoveQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	return z.RemoveQueryNodeById(n.Id())
}

func (z *zkCluster) RemoveQueryNodeById(id string) (clustering.QueryNode, errors.Error) {
	impl, ok := getConfigStoreImplementation(z)
	if !ok {
		return nil, errors.NewWarning(fmt.Sprintf("Unable to get connection to zookeeper at %s", z.ConfigurationStoreId()))
	}
	err := impl.conn.Delete("/"+z.Id()+"/"+id, 0)
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	return nil, nil
}

func (z *zkCluster) GetQueryNodes() ([]clustering.QueryNode, errors.Error) {
	impl, ok := getConfigStoreImplementation(z)
	if !ok {
		return nil, errors.NewWarning(fmt.Sprintf("Unable to get connection to zookeeper at %s", z.ConfigurationStoreId()))
	}
	qryNodes := []clustering.QueryNode{}
	nodes, _, err := impl.conn.Children("/" + z.Id())
	if err != nil {
		return nil, errors.NewError(err, "")
	}
	for _, name := range nodes {
		data, _, err := impl.conn.Get("/" + z.Id() + "/" + name)
		if err != nil {
			return nil, errors.NewError(err, "")
		}
		queryNode := &zkQueryNodeConfig{}
		err = json.Unmarshal(data, queryNode)
		if err != nil {
			return nil, errors.NewError(err, "")
		}
		qryNodes = append(qryNodes, queryNode)
	}
	return qryNodes, nil
}

type zkQueryNodeConfig struct {
	configStore      clustering.ConfigurationStore `json:"-"`
	dataStore        datastore.Datastore           `json:"-"`
	acctStore        accounting.AccountingStore    `json:"-"`
	ClusterName      string                        `json:"cluster_name"`
	QueryNodeName    string                        `json:"name"`
	QueryEndpointURL string                        `json:"query_endpoint"`
	AdminEndpointURL string                        `json:"admin_endpoint"`
	DatastoreURI     string                        `json:"datastore_uri"`
	ConfigstoreURI   string                        `json:"configstore_uri"`
	AccountingURI    string                        `json:"accountstore_uri"`
	Vers             *zkVersion                    `json:"version"`
}

func NewQueryNode(ClusterName string, Name string, VersionString string, queryEndpoint string, adminEndpoint string, cs clustering.ConfigurationStore, ds datastore.Datastore, as accounting.AccountingStore) clustering.QueryNode {
	node := zkQueryNodeConfig{
		configStore:      cs,
		dataStore:        ds,
		acctStore:        as,
		ClusterName:      ClusterName,
		QueryNodeName:    Name,
		QueryEndpointURL: queryEndpoint,
		AdminEndpointURL: adminEndpoint,
		DatastoreURI:     ds.URL(),
		ConfigstoreURI:   cs.URL(),
		AccountingURI:    as.URL(),
		Vers:             NewVersion(VersionString),
	}
	return &node
}

func (z *zkQueryNodeConfig) String() string {
	return fmt.Sprintf("name=%s, queryEndpoint=%s, adminEndpoint=%s, datastoreURI=%s, configstoreURI=%s, accountingURI=%s, version=%s", z.QueryNodeName, z.QueryEndpointURL, z.AdminEndpointURL, z.DatastoreURI, z.ConfigstoreURI, z.AccountingURI, z.Vers)
}

func (z *zkQueryNodeConfig) ClusterId() string {
	return z.ClusterName
}

func (z *zkQueryNodeConfig) Id() string {
	return z.QueryNodeName
}

func (z *zkQueryNodeConfig) Hostname() string {
	return ""
}

func (z *zkQueryNodeConfig) IPAddress() string {
	return ""
}

func (z *zkQueryNodeConfig) QueryEndpoint() string {
	return z.QueryEndpointURL
}

func (z *zkQueryNodeConfig) ClusterEndpoint() string {
	return z.AdminEndpointURL
}

func (z *zkQueryNodeConfig) Version() clustering.Version {
	return z.Vers
}

func (z *zkQueryNodeConfig) Mode() clustering.Mode {
	if z.configStore == nil {
		return clustering.STANDALONE
	} else {
		return clustering.CLUSTER
	}
}

func (z *zkQueryNodeConfig) Datastore() datastore.Datastore {
	return z.dataStore
}

func (z *zkQueryNodeConfig) AccountingStore() accounting.AccountingStore {
	return z.acctStore
}

func (z *zkQueryNodeConfig) ConfigurationStore() clustering.ConfigurationStore {
	return z.configStore
}
