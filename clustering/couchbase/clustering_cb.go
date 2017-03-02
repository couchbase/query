package clustering_cb

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/go-couchbase"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/server/http"
	"github.com/couchbase/query/util"
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
	c, err := couchbase.ConnectWithAuth(path, cbauth.NewAuthHandler(nil))
	if err != nil {
		return nil, errors.NewAdminConnectionError(err, path)
	}
	return &cbConfigStore{
		adminUrl: path,
		cbConn:   &c,
	}, nil
}

// Implement Stringer interface
func (this *cbConfigStore) String() string {
	return fmt.Sprintf("url=%v", this.adminUrl)
}

// Implement clustering.ConfigurationStore interface
func (this *cbConfigStore) Id() string {
	return this.URL()
}

func (this *cbConfigStore) URL() string {
	return this.adminUrl
}

func (this *cbConfigStore) ClusterNames() ([]string, errors.Error) {
	clusterIds := []string{}
	for _, pool := range this.getPools() {
		clusterIds = append(clusterIds, pool.Name)
	}
	return clusterIds, nil
}

func (this *cbConfigStore) ClusterByName(name string) (clustering.Cluster, errors.Error) {
	_, err := this.cbConn.GetPool(name)
	if err != nil {
		return nil, errors.NewAdminGetClusterError(err, name)
	}
	return &cbCluster{
		configStore:    this,
		ClusterName:    name,
		ConfigstoreURI: this.URL(),
		DatastoreURI:   this.URL(),
	}, nil
}

func (this *cbConfigStore) ConfigurationManager() clustering.ConfigurationManager {
	return this
}

// Helper method to retrieve all pools
func (this *cbConfigStore) getPools() []couchbase.RestPool {
	return this.cbConn.Info.Pools
}

// Helper method to retrieve Couchbase services data (/pools/default/nodeServices)
// and Couchbase pool (cluster) data (/pools/default)
//
func (this *cbConfigStore) getPoolServices(name string) (*couchbase.Pool, *couchbase.PoolServices, errors.Error) {
	nodeServices, err := this.cbConn.GetPoolServices(name)
	if err != nil {
		return nil, nil, errors.NewAdminGetClusterError(err, name)
	}

	pool, err := this.cbConn.GetPool(name)
	if err != nil {
		return nil, nil, errors.NewAdminGetClusterError(err, name)
	}

	return &pool, &nodeServices, nil
}

// cbConfigStore also implements clustering.ConfigurationManager interface
func (this *cbConfigStore) ConfigurationStore() clustering.ConfigurationStore {
	return this
}

func (this *cbConfigStore) AddCluster(l clustering.Cluster) (clustering.Cluster, errors.Error) {
	// NOP. This is a read-only implementation
	return nil, nil
}

func (this *cbConfigStore) RemoveCluster(l clustering.Cluster) (bool, errors.Error) {
	// NOP. This is a read-only implementation
	return false, nil
}

func (this *cbConfigStore) RemoveClusterByName(name string) (bool, errors.Error) {
	// NOP. This is a read-only implementation
	return false, nil
}

func (this *cbConfigStore) GetClusters() ([]clustering.Cluster, errors.Error) {
	clusters := []clustering.Cluster{}
	clusterNames, _ := this.ClusterNames()
	for _, name := range clusterNames {
		cluster, err := this.ClusterByName(name)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

func (this *cbConfigStore) Authorize(credentials map[string]string, privileges []clustering.Privilege) errors.Error {
	if len(credentials) == 0 {
		return errors.NewAdminAuthError(nil, "no credentials provided")
	}

	for username, password := range credentials {
		auth, err := cbauth.Auth(username, password)
		if err != nil {
			return errors.NewAdminAuthError(err, "unable to authenticate with given credential")
		}
		for _, requested := range privileges {
			switch requested {
			case clustering.PRIV_SYS_ADMIN:
				isAdmin, err := auth.IsAllowed("cluster.settings!write")
				if err != nil {
					return errors.NewAdminAuthError(err, "")
				}
				if isAdmin {
					return nil
				}
				return errors.NewAdminAuthError(nil, "sys admin requires administrator credentials")
			case clustering.PRIV_READ:
				isPermitted, err := auth.IsAllowed("cluster.settings!read")
				if err != nil {
					return errors.NewAdminAuthError(err, "")
				}
				if isPermitted {
					return nil
				}
				return errors.NewAdminAuthError(nil, "read not authorized")
			default:
				return errors.NewAdminAuthError(nil, fmt.Sprintf("unexpected authorization %v", requested))
			}
		}
	}
	return errors.NewAdminAuthError(nil, "unrecognized authorization request")
}

const n1qlService = "n1ql"

func (this *cbConfigStore) WhoAmI() (string, errors.Error) {

	// this will exhaust all possibilities in the hope of
	// finding a good name and only return an error
	// if we could not find a name at all
	var err errors.Error

	localIp, _ := util.ExternalIP()
	localName, _ := os.Hostname()
	for _, p := range this.getPools() {
		pool, newErr := this.cbConn.GetPool(p.Name)
		if newErr != nil {
			if err == nil {
				err = errors.NewAdminGetClusterError(newErr, p.Name)
			}
			continue
		}

		for _, node := range pool.Nodes {
			isN1ql := false
			for _, s := range node.Services {
				if s == n1qlService {
					isN1ql = true
					break
				}
			}
			if !isN1ql {
				continue
			}
			theName := nodeName(node)

			// Is it the IP?
			if len(localIp) != 0 {
				if localIp == theName {
					return theName, nil
				}

				// Is it the domain name?
				domainNames, _ := net.LookupAddr(localIp)
				for _, domainName := range domainNames {
					if domainName == theName {
						return theName, nil
					}
				}
			}

			// Is it the hostname?
			if localName == theName {
				return theName, nil
			}

			// No, it's localhost!
			if node.ThisNode && len(localIp) > 0 &&
				(theName == "" || theName == "localhost" || theName == "127.0.0.1") {
				return localIp, nil
			}
		}
	}
	return "", err
}

// Type services associates a protocol with a port number
type services map[string]int

const (
	_HTTP  = "http"
	_HTTPS = "https"
)

// n1qlProtocols associates Couchbase query service names with a protocol
var n1qlProtocols = map[string]string{
	"n1ql":    _HTTP,
	"n1qlSSL": _HTTPS,
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
	queryNodes     map[string]services           `json:"-"`
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
func (this *cbCluster) String() string {
	return getJsonString(this)
}

// cbCluster implements clustering.Cluster interface
func (this *cbCluster) ConfigurationStoreId() string {
	return this.configStore.Id()
}

func (this *cbCluster) Name() string {
	return this.ClusterName
}

func (this *cbCluster) QueryNodeNames() ([]string, errors.Error) {
	queryNodeNames := []string{}
	// Get a handle of the go-couchbase connection:
	cbConn, ok := this.configStore.(*cbConfigStore)
	if !ok {
		return nil, errors.NewAdminConnectionError(nil, this.ConfigurationStoreId())
	}

	pool, poolServices, err := cbConn.getPoolServices(this.ClusterName)
	if err != nil {
		return queryNodeNames, err
	}

	// If pool services rev matches the cluster's rev, return cluster's query node names:
	if poolServices.Rev == this.poolSrvRev {
		return this.queryNodeNames, nil
	}

	// If pool services and cluster rev do not match, update the cluster's rev and query node data:
	queryNodeNames = []string{}
	queryNodes := map[string]services{}
	for _, nodeServices := range poolServices.NodesExt {
		var queryServices services
		for name, protocol := range n1qlProtocols {
			if nodeServices.Services[name] != 0 {
				if queryServices == nil {
					queryServices = services{}
				}
				queryServices[protocol] = nodeServices.Services[name]
			}
		}

		if len(queryServices) == 0 { // no n1ql service at this node
			continue
		}

		hostname := nodeServices.Hostname
		if nodeServices.ThisNode {
			for _, node := range pool.Nodes {
				if node.ThisNode {
					hostname = nodeName(node)
					break
				}
			}
		}
		// if nodes have non localhost ip address as the node name, all fine
		// if named localhost, then things get a bit hairy.
		// when queried from a remote machine, hosts named localhost will return
		// an actual ip address.
		// if the cluster has more than one node, you get an actual ip address
		// even from the local node.
		// single node and queried locally, they may return a blank name or localhost
		// thing is, clustering code is hardwired to regard 127.0.0.1 as not part of
		// a cluster, so we have to fix things by hand.
		if hostname == "" || hostname == "localhost" || hostname == "127.0.0.1" {
			localIp, _ := util.ExternalIP()
			if localIp != "" {
				hostname = localIp
			} else {

				// didn't work out, fix it for blank name
				hostname = "127.0.0.1"
			}
		}

		queryNodeNames = append(queryNodeNames, hostname)
		queryNodes[hostname] = queryServices
	}

	this.Lock()
	defer this.Unlock()
	this.queryNodeNames = queryNodeNames
	this.queryNodes = queryNodes
	this.poolSrvRev = poolServices.Rev
	return this.queryNodeNames, nil
}

func nodeName(node couchbase.Node) string {
	tokens := strings.Split(node.Hostname, ":")
	return tokens[0]
}

func (this *cbCluster) QueryNodeByName(name string) (clustering.QueryNode, errors.Error) {
	qryNodeNames, err := this.QueryNodeNames()
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
		return nil, errors.NewAdminNoNodeError(name)
	}

	qryNode := &cbQueryNodeConfig{
		ClusterName:   this.Name(),
		QueryNodeName: qryNodeName,
	}

	for protocol, port := range this.queryNodes[qryNodeName] {
		switch protocol {
		case _HTTP:
			qryNode.Query = makeURL(protocol, qryNodeName, port, http.ServicePrefix())
			qryNode.Admin = makeURL(protocol, qryNodeName, port, http.AdminPrefix())
		case _HTTPS:
			qryNode.QuerySSL = makeURL(protocol, qryNodeName, port, http.ServicePrefix())
			qryNode.AdminSSL = makeURL(protocol, qryNodeName, port, http.AdminPrefix())
		}
	}

	return qryNode, nil
}

func (this *cbCluster) Datastore() datastore.Datastore {
	return this.dataStore
}

func (this *cbCluster) AccountingStore() accounting.AccountingStore {
	return this.acctStore
}

func (this *cbCluster) ConfigurationStore() clustering.ConfigurationStore {
	return this.configStore
}

func (this *cbCluster) Version() clustering.Version {
	if this.version == nil {
		this.version = clustering.NewVersion(this.VersionString)
	}
	return this.version
}

func (this *cbCluster) ClusterManager() clustering.ClusterManager {
	return this
}

// cbCluster implements clustering.ClusterManager interface
func (this *cbCluster) Cluster() clustering.Cluster {
	return this
}

func (this *cbCluster) AddQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	// NOP. This is a read-only implementation
	return nil, nil
}

func (this *cbCluster) RemoveQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	return this.RemoveQueryNodeByName(n.Name())
}

func (this *cbCluster) RemoveQueryNodeByName(name string) (clustering.QueryNode, errors.Error) {
	// NOP. This is a read-only implementation
	return nil, nil
}

func (this *cbCluster) GetQueryNodes() ([]clustering.QueryNode, errors.Error) {
	qryNodes := []clustering.QueryNode{}
	names, err := this.QueryNodeNames()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		qryNode, err := this.QueryNodeByName(name)
		if err != nil {
			return nil, err
		}
		qryNodes = append(qryNodes, qryNode)
	}
	return qryNodes, nil
}

// cbQueryNodeConfig implements clustering.QueryNode
type cbQueryNodeConfig struct {
	ClusterName   string                    `json:"cluster"`
	QueryNodeName string                    `json:"name"`
	Query         string                    `json:"queryEndpoint,omitempty"`
	Admin         string                    `json:"adminEndpoint,omitempty"`
	QuerySSL      string                    `json:"querySecure,omitempty"`
	AdminSSL      string                    `json:"adminSecure,omitempty"`
	ClusterRef    *cbCluster                `json:"-"`
	StandaloneRef *clustering.StdStandalone `json:"-"`
	OptionsCL     *clustering.ClOptions     `json:"options"`
}

// cbQueryNodeConfig implements Stringer interface
func (this *cbQueryNodeConfig) String() string {
	return getJsonString(this)
}

// cbQueryNodeConfig implements clustering.QueryNode interface
func (this *cbQueryNodeConfig) Cluster() clustering.Cluster {
	return this.ClusterRef
}

func (this *cbQueryNodeConfig) Name() string {
	return this.QueryNodeName
}

func (this *cbQueryNodeConfig) QueryEndpoint() string {
	return this.Query
}

func (this *cbQueryNodeConfig) ClusterEndpoint() string {
	return this.Admin
}

func (this *cbQueryNodeConfig) QuerySecure() string {
	return this.QuerySSL
}

func (this *cbQueryNodeConfig) ClusterSecure() string {
	return this.AdminSSL
}

func (this *cbQueryNodeConfig) Standalone() clustering.Standalone {
	return this.StandaloneRef
}

func (this *cbQueryNodeConfig) Options() clustering.QueryNodeOptions {
	return this.OptionsCL
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

func makeURL(protocol string, host string, port int, endpoint string) string {
	if port == 0 {
		return ""
	}
	urlParts := []string{protocol, "://", host, ":", strconv.Itoa(port), endpoint}
	return strings.Join(urlParts, "")
}
