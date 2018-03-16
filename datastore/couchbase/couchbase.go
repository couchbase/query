//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package couchbase provides a couchbase-server implementation of the datastore
package.

*/

package couchbase

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth"
	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/gomemcached"
	gsi "github.com/couchbase/indexing/secondary/queryport/n1ql"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"

	"github.com/couchbase/query/server"
)

var REQUIRE_CBAUTH bool // Connection to authorization system must succeed.

// cbPoolMap and cbPoolServices implement a local cache of the datastore's topology
type cbPoolMap struct {
	sync.RWMutex
	poolServices map[string]cbPoolServices
}

type cbPoolServices struct {
	name         string
	rev          int
	nodeServices map[string]interface{}
}

var _POOLMAP cbPoolMap

func init() {

	// MB-27415 have a larger overflow pool and close overflow connections asynchronously
	cb.SetConnectionPoolParams(64, 64)
	cb.EnableAsynchronousCloser(true)

	val, err := strconv.ParseBool(os.Getenv("REQUIRE_CBAUTH"))
	if err != nil {
		REQUIRE_CBAUTH = val
	} else {
		REQUIRE_CBAUTH = true // default
	}

	// enable data type response
	cb.EnableDataType = true

	// start the fetch workers for servicing the BulkGet operations
	cb.InitBulkGet()
	_POOLMAP.poolServices = make(map[string]cbPoolServices, 1)
}

const (
	PRIMARY_INDEX = "#primary"
)

// store is the root for the couchbase datastore
type store struct {
	client         cb.Client             // instance of go-couchbase client
	namespaceCache map[string]*namespace // map of pool-names and IDs
	CbAuthInit     bool                  // whether cbAuth is initialized
	inferencer     datastore.Inferencer  // what we use to infer schemas
	connectionUrl  string                // where to contact ns_server
}

func (s *store) Id() string {
	return s.URL()
}

func (s *store) URL() string {
	return s.client.BaseURL.String()
}

func (s *store) Info() datastore.Info {
	info := &infoImpl{client: &s.client}
	return info
}

type infoImpl struct {
	client *cb.Client
}

func (info *infoImpl) Version() string {
	return info.client.Info.ImplementationVersion
}

func fullhostName(n string) string {
	hostName, portVal := server.HostNameandPort(n)
	if hostName != "" {
		return n
	}
	return server.GetIP(true) + ":" + portVal
}

func (info *infoImpl) Topology() ([]string, []errors.Error) {
	var nodes []string
	var errs []errors.Error

	for _, p := range info.client.Info.Pools {
		pool, err := info.client.GetPool(p.Name)

		if err == nil {
			for _, node := range pool.Nodes {
				nodes = append(nodes, fullhostName(node.Hostname))
			}
		} else {
			errs = append(errs, errors.NewDatastoreClusterError(err, p.Name))
		}
	}
	return nodes, errs
}

func (info *infoImpl) Services(node string) (map[string]interface{}, []errors.Error) {
	var errs []errors.Error

	isReadLock := true
	_POOLMAP.RLock()
	defer func() {
		if isReadLock {
			_POOLMAP.RUnlock()
		} else {
			_POOLMAP.Unlock()
		}
	}()

	// scan the pools
	for _, p := range info.client.Info.Pools {
		pool, err := info.client.GetPool(p.Name)
		poolServices, pErr := info.client.GetPoolServices(p.Name)

		if err == nil && pErr == nil {
			var found bool = false
			var services cbPoolServices

			services, ok := _POOLMAP.poolServices[p.Name]
			found = ok && (services.rev == poolServices.Rev)

			// missing the information, rebuild
			if !found {

				// promote the lock
				if isReadLock {
					_POOLMAP.RUnlock()
					_POOLMAP.Lock()
					isReadLock = false

					// now that we have promoted the lock, did we get beaten by somebody else to it?
					services, ok = _POOLMAP.poolServices[p.Name]
					found = ok && (services.rev == poolServices.Rev)
					if found {
						continue
					}
				}

				newPoolServices := cbPoolServices{name: p.Name, rev: poolServices.Rev}
				nodeServices := make(map[string]interface{}, len(pool.Nodes))

				// go through all the nodes in the pool
				for _, n := range pool.Nodes {
					var servicesCopy []interface{}

					newServices := make(map[string]interface{}, 3)
					newServices["name"] = fullhostName(n.Hostname)
					for _, s := range n.Services {
						servicesCopy = append(servicesCopy, s)
					}
					newServices["services"] = servicesCopy

					// go through all bucket independet services in the pool
					for _, ns := range poolServices.NodesExt {

						mgmtPort := ns.Services["mgmt"]
						if mgmtPort == 0 {

							// shouldn't happen, there should always be a mgmt port on each node
							// we should return an error
							msg := fmt.Sprintf("NodeServices does not report mgmt endpoint for "+
								"this node: %v", node)
							errs = append(errs, errors.NewAdminGetNodeError(nil, msg))
							continue
						}

						nsHostname := ""
						if ns.Hostname != "" {
							nsHostname = ns.Hostname + ":" + strconv.Itoa(mgmtPort)
						}
						// if we can positively match nodeServices and node, add ports
						if n.Hostname == nsHostname ||
							(nsHostname == "" && ns.ThisNode && n.ThisNode) {
							ports := make(map[string]interface{}, len(ns.Services))

							// only add the ports for those services that are advertised
							for _, s := range n.Services {
								for pn, p := range ns.Services {
									if strings.Index(pn, s) == 0 {
										ports[pn] = p
									}
								}
							}
							newServices["ports"] = ports
							break
						}
					}
					nodeServices[fullhostName(n.Hostname)] = newServices
				}
				newPoolServices.nodeServices = nodeServices
				_POOLMAP.poolServices[p.Name] = newPoolServices
				services = newPoolServices
			}
			nodeServices, ok := services.nodeServices[node]
			if ok {
				return nodeServices.(map[string]interface{}), errs
			}
		} else {
			if err != nil {
				errs = append(errs, errors.NewDatastoreClusterError(err, p.Name))
			}
			if pErr != nil {
				errs = append(errs, errors.NewDatastoreClusterError(pErr, p.Name))
			}
		}
	}
	return map[string]interface{}{}, errs
}

func (s *store) NamespaceIds() ([]string, errors.Error) {
	return s.NamespaceNames()
}

func (s *store) NamespaceNames() ([]string, errors.Error) {
	return []string{"default"}, nil
}

func (s *store) NamespaceById(id string) (p datastore.Namespace, e errors.Error) {
	return s.NamespaceByName(id)
}

func (s *store) NamespaceByName(name string) (p datastore.Namespace, e errors.Error) {
	p, ok := s.namespaceCache[name]
	if !ok {
		var err errors.Error
		p, err = loadNamespace(s, name)
		if err != nil {
			return nil, err
		}
		s.namespaceCache[name] = p.(*namespace)
	}
	return p, nil
}

// The ns_server admin API is open iff we can access the /pools API without a password.
func (s *store) adminIsOpen() bool {
	url := s.connectionUrl + "/pools"
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	return true
}

func (s *store) auth(user, pwd string) (cbauth.Creds, error) {
	return cbauth.Auth(user, pwd)
}

func (s *store) authWebCreds(req *http.Request) (cbauth.Creds, error) {
	return cbauth.AuthWebCreds(req)
}

func (s *store) Authorize(privileges *auth.Privileges, credentials auth.Credentials, req *http.Request) (auth.AuthenticatedUsers, errors.Error) {
	if s.CbAuthInit == false {
		// cbauth is not initialized. Access to SASL protected buckets will be
		// denied by the couchbase server
		logging.Warnf("CbAuth not intialized")
		return nil, nil
	}

	return cbAuthorize(s, privileges, credentials, req)
}

func (s *store) CredsString(req *http.Request) string {
	if req != nil {
		creds, err := cbauth.AuthWebCreds(req)
		if err == nil {
			return creds.Name()
		}
	}
	return ""
}

func (s *store) SetLogLevel(level logging.Level) {
	for _, n := range s.namespaceCache {
		defer n.lock.Unlock()
		n.lock.Lock()
		for _, k := range n.keyspaceCache {
			if k.cbKeyspace == nil {
				continue
			}
			indexers, _ := k.cbKeyspace.Indexers()
			if len(indexers) > 0 {
				for _, idxr := range indexers {
					idxr.SetLogLevel(level)
				}

				return
			}
		}
	}
}

// Ignore the name parameter for now
func (s *store) Inferencer(name datastore.InferenceType) (datastore.Inferencer, errors.Error) {
	return s.inferencer, nil
}

func (s *store) Inferencers() ([]datastore.Inferencer, errors.Error) {
	return []datastore.Inferencer{s.inferencer}, nil
}

func (s *store) AuditInfo() (*datastore.AuditInfo, errors.Error) {
	auditSpec, err := s.client.GetAuditSpec()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err)
	}

	users := make(map[datastore.UserInfo]bool, len(auditSpec.DisabledUsers))
	for _, u := range auditSpec.DisabledUsers {
		ui := datastore.UserInfo{Name: u.Name, Domain: u.Domain}
		users[ui] = true
	}

	events := make(map[uint32]bool, len(auditSpec.Disabled))
	for _, eid := range auditSpec.Disabled {
		events[eid] = true
	}

	ret := &datastore.AuditInfo{
		EventDisabled:   events,
		UserWhitelisted: users,
		AuditEnabled:    auditSpec.AuditdEnabled,
		Uid:             auditSpec.Uid,
	}
	return ret, nil
}

func (s *store) ProcessAuditUpdateStream(callb func(uid string) error) errors.Error {
	f := func(data interface{}) error {
		d, ok := data.(*DefaultObject)
		if !ok {
			return fmt.Errorf("Unable to convert received object to proper type: %T", data)
		}
		return callb(d.Uid)
	}
	do := DefaultObject{}
	err := s.client.ProcessStream("/poolsStreaming/default", f, &do)
	if err != nil {
		return errors.NewAuditStreamHandlerFailed(err)
	}
	return nil
}

type DefaultObject struct {
	Uid string `json:"auditUid"`
}

func (s *store) UserInfo() (value.Value, errors.Error) {
	data, err := s.client.GetUserRoles()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err)
	}
	return value.NewValue(data), nil
}

func (s *store) GetUserInfoAll() ([]datastore.User, errors.Error) {
	sourceUsers, err := s.client.GetUserInfoAll()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err)
	}
	resultUsers := make([]datastore.User, len(sourceUsers))
	for i, u := range sourceUsers {
		resultUsers[i].Name = u.Name
		resultUsers[i].Id = u.Id
		resultUsers[i].Domain = u.Domain
		roles := make([]datastore.Role, len(u.Roles))
		for j, r := range u.Roles {
			roles[j].Name = r.Role
			roles[j].Bucket = r.BucketName
		}
		resultUsers[i].Roles = roles
	}
	return resultUsers, nil
}

func (s *store) PutUserInfo(u *datastore.User) errors.Error {
	var outputUser cb.User
	outputUser.Name = u.Name
	outputUser.Id = u.Id
	outputUser.Roles = make([]cb.Role, len(u.Roles))
	outputUser.Domain = u.Domain
	for i, r := range u.Roles {
		outputUser.Roles[i].Role = r.Name
		outputUser.Roles[i].BucketName = r.Bucket
	}
	err := s.client.PutUserInfo(&outputUser)
	if err != nil {
		return errors.NewSystemUnableToUpdateError(err)
	}
	return nil
}

func (s *store) GetRolesAll() ([]datastore.Role, errors.Error) {
	roleDescList, err := s.client.GetRolesAll()
	if err != nil {
		return nil, errors.NewDatastoreUnableToRetrieveRoles(err)
	}
	roles := make([]datastore.Role, len(roleDescList))
	for i, rd := range roleDescList {
		roles[i].Name = rd.Role
		roles[i].Bucket = rd.BucketName
	}
	return roles, nil
}

func initCbAuth(url string) (*cb.Client, error) {

	transport := cbauth.WrapHTTPTransport(cb.HTTPTransport, nil)
	cb.HTTPClient.Transport = transport

	client, err := cb.ConnectWithAuth(url, cbauth.NewAuthHandler(nil))
	if err != nil {
		return nil, err
	}

	logging.Infof(" Initialization of cbauth succeeded ")

	return &client, nil
}

func parseUrl(u string) (host string, username string, password string, err error) {
	url, err := url.Parse(u)
	if err != nil {
		return "", "", "", err
	}
	if url.User == nil {
		return "", "", "", fmt.Errorf("Unusable url %s. No user information.", u)
	}
	password, _ = url.User.Password()
	if password == "" {
		logging.Warnf("No password found in url <ud>%s</ud>.", u)
	}
	if url.User.Username() == "" {
		logging.Warnf("No username found in url <ud>%s</ud>.", u)
	}
	return url.Host, url.User.Username(), password, nil
}

// NewStore creates a new Couchbase store for the given url.
// In the main server, and error return here will cause the server to shut down.
func NewDatastore(u string) (s datastore.Datastore, e errors.Error) {
	var client cb.Client
	var cbAuthInit bool
	var err error
	var connectionUrl string

	// initialize cbauth
	c, err := initCbAuth(u)
	if err != nil {
		logging.Errorf("Unable to initialize cbauth. Error %v", err)

		// intialize cb_auth variables manually
		host, username, password, err := parseUrl(u)
		if err != nil {
			logging.Warnf("Unable to parse url <ud>%s</ud>: %v", u, err)
		} else {
			set, err := cbauth.InternalRetryDefaultInit(host, username, password)
			if set == false || err != nil {
				logging.Errorf("Unable to initialize cbauth variables. Error %v", err)
			} else {
				c, err = initCbAuth("http://" + host)
				if err != nil {
					logging.Errorf("Unable to initialize cbauth. Error %v", err)
				} else {
					client = *c
					cbAuthInit = true
					connectionUrl = "http://" + host
				}
			}
		}
	} else {
		client = *c
		cbAuthInit = true
		connectionUrl = u
	}

	if !cbAuthInit {
		if REQUIRE_CBAUTH {
			return nil, errors.NewUnableToInitCbAuthError(err)
		}
		// connect without auth
		logging.Warnf("Unable to initialize cbAuth, access to couchbase buckets may be restricted")
		cb.HTTPClient = &http.Client{}
		client, err = cb.Connect(u)
		if err != nil {
			return nil, errors.NewCbConnectionError(err, "url "+u)
		}
	}

	store := &store{
		client:         client,
		namespaceCache: make(map[string]*namespace),
		CbAuthInit:     cbAuthInit,
		connectionUrl:  connectionUrl,
	}

	// get the schema inferencer
	var er errors.Error
	store.inferencer, er = GetDefaultInferencer(store)
	if er != nil {
		return nil, er
	}

	// initialize the default pool.
	// TODO can couchbase server contain more than one pool ?

	defaultPool, er := loadNamespace(store, "default")
	if er != nil {
		logging.Errorf("Cannot connect to default pool")
		return nil, er
	}

	store.namespaceCache["default"] = defaultPool
	logging.Infof("New store created with url %s", u)

	return store, nil
}

func loadNamespace(s *store, name string) (*namespace, errors.Error) {

	cbpool, err := s.client.GetPool(name)
	if err != nil {
		if name == "default" {
			// if default pool is not available, try reconnecting to the server
			url := s.URL()

			var client cb.Client

			if s.CbAuthInit == true {
				client, err = cb.ConnectWithAuth(url, cbauth.NewAuthHandler(nil))
			} else {
				client, err = cb.Connect(url)
			}
			if err != nil {
				return nil, errors.NewCbNamespaceNotFoundError(err, "Namespace "+name)
			}
			// check if the default pool exists
			cbpool, err = client.GetPool(name)
			if err != nil {
				return nil, errors.NewCbNamespaceNotFoundError(err, "Namespace "+name)
			}
			s.client = client
		}
	}

	rv := namespace{
		store:         s,
		name:          name,
		cbNamespace:   cbpool,
		keyspaceCache: make(map[string]*keyspaceEntry),
	}

	return &rv, nil
}

// a namespace represents a couchbase pool
type namespace struct {
	store         *store
	name          string
	cbNamespace   cb.Pool
	keyspaceCache map[string]*keyspaceEntry
	version       uint64
	lock          sync.RWMutex // lock to guard the keyspaceCache
	nslock        sync.RWMutex // lock for this structure
}

type keyspaceEntry struct {
	sync.Mutex
	cbKeyspace datastore.Keyspace
	errCount   int
	errTime    time.Time
	lastUse    time.Time
}

const (
	_MIN_ERR_INTERVAL   time.Duration = 5 * time.Second
	_THROTTLING_TIMEOUT time.Duration = 10 * time.Millisecond
	_CLEANUP_INTERVAL   time.Duration = time.Hour
)

func (p *namespace) DatastoreId() string {
	return p.store.Id()
}

func (p *namespace) Id() string {
	return p.Name()
}

func (p *namespace) Name() string {
	return p.name
}

func (p *namespace) KeyspaceIds() ([]string, errors.Error) {
	return p.KeyspaceNames()
}

func (p *namespace) KeyspaceNames() ([]string, errors.Error) {
	p.refresh(true)
	rv := make([]string, 0, len(p.cbNamespace.BucketMap))
	for name, _ := range p.cbNamespace.BucketMap {
		rv = append(rv, name)
	}
	return rv, nil
}

func (p *namespace) KeyspaceByName(name string) (datastore.Keyspace, errors.Error) {
	var err errors.Error

	// make sure that no one is deleting the keyspace as we check
	p.lock.RLock()
	entry, ok := p.keyspaceCache[name]
	p.lock.RUnlock()
	if ok && entry.cbKeyspace != nil {
		return entry.cbKeyspace, nil
	}

	// MB-19601 we haven't found the keyspace, so we have to load it,
	// however, there might be a flood of other requests coming in, all
	// wanting to do use the same keyspace and all needing to load it.
	// In the previous implementation all requests would first create
	// and refresh the keyspace, refreshing the indexes, etc
	// In YCSB enviroments this resulted in thousends of requests
	// flooding ns_server with buckets and ddocs load at the same time.
	// What we want instead is for one request to do the work, and all the
	// others waiting and benefiting from that work.
	// This is the exact scenario for using Shared Optimistic Locks, but,
	// sadly, they are patented by IBM, so clearly it's no go for us.
	// What we do is create the keyspace entry, and record that we are priming
	// it by locking that entry.
	// Everyone else will have to wait on the lock, and once they get it,
	// they can check on the keyspace again - if all is fine, just continue
	// if not try and load again.
	// Shared Optimistic Locks by stealth, although not as efficient (there
	// might be sequencing of would be loaders on the keyspace lock after
	// the initial keyspace loading has been done).
	// If we fail, again! then there's something wrong with the keyspace,
	// which means that retrying over and over again, we'll be loading ns_server
	// so what we do is throttle the reloads and log errors, so that the
	// powers that be are alerted that there's some resource issue.
	// Finally, since we are having to use two locks rather than one, make sure
	// that the locking sequence is predictable.
	// keyspace lock is always locked outside of the keyspace cache lock.

	// 1) create the entry if necessary, record time of loading attempt
	p.lock.Lock()
	entry, ok = p.keyspaceCache[name]
	if !ok {
		entry = &keyspaceEntry{}
		p.keyspaceCache[name] = entry
	}
	entry.lastUse = time.Now()
	p.lock.Unlock()

	// 2) serialize the loading by locking the entry
	entry.Lock()
	defer entry.Unlock()

	// 3) check if somebody has done the job for us in the interim
	if entry.cbKeyspace != nil {
		return entry.cbKeyspace, nil
	}

	// 4) if previous loads resulted in errors, throttle requests
	if entry.errCount > 0 && time.Since(entry.lastUse) < _THROTTLING_TIMEOUT {
		time.Sleep(_THROTTLING_TIMEOUT)
	}

	// 5) try the loading
	k, err := newKeyspace(p, name)
	if err != nil {

		// We try not to flood the log with errors
		if entry.errCount == 0 {
			entry.errTime = time.Now()
		} else if time.Since(entry.errTime) > _MIN_ERR_INTERVAL {
			entry.errTime = time.Now()
		}
		entry.errCount++
		return nil, err
	}
	entry.errCount = 0

	// this is the only place where entry.cbKeyspace is set
	// it is never unset - so it's safe to test cbKeyspace != nil
	entry.cbKeyspace = k
	return k, nil
}

// compare the list of node addresses
// Assumption: the list of node addresses in each list are sorted
func compareNodeAddress(a, b []string) bool {

	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (p *namespace) KeyspaceById(id string) (datastore.Keyspace, errors.Error) {
	return p.KeyspaceByName(id)
}

func (p *namespace) MetadataVersion() uint64 {
	return p.version
}

func (p *namespace) setPool(cbpool cb.Pool) {
	p.nslock.Lock()
	defer p.nslock.Unlock()
	p.cbNamespace = cbpool
}

func (p *namespace) getPool() cb.Pool {
	p.nslock.RLock()
	defer p.nslock.RUnlock()
	return p.cbNamespace
}

func (p *namespace) refresh(changed bool) {
	// trigger refresh of this pool
	logging.Debugf("Refreshing pool %s", p.name)

	newpool, err := p.store.client.GetPool(p.name)
	if err != nil {

		var client cb.Client

		logging.Errorf("Error updating pool name %s: Error %v", p.name, err)
		url := p.store.URL()

		/*
			transport := cbauth.WrapHTTPTransport(cb.HTTPTransport, nil)
			cb.HTTPClient.Transport = transport
		*/

		if p.store.CbAuthInit == true {
			client, err = cb.ConnectWithAuth(url, cbauth.NewAuthHandler(nil))
		} else {
			client, err = cb.Connect(url)
		}
		if err != nil {
			logging.Errorf("Error connecting to URL %s", url)
			return
		}
		// check if the default pool exists
		newpool, err = client.GetPool(p.name)
		if err != nil {
			logging.Errorf("Retry Failed Error updating pool name <ud>%s</ud>: Error %v", p.name, err)
			return
		}
		p.store.client = client

	}

	p.lock.Lock()
	defer p.lock.Unlock()
	for name, ks := range p.keyspaceCache {
		logging.Debugf(" Checking keyspace %s", name)
		if ks.cbKeyspace == nil {
			if time.Since(ks.lastUse) > _CLEANUP_INTERVAL {
				delete(p.keyspaceCache, name)
			}
			continue
		}
		newbucket, err := newpool.GetBucket(name)
		if err != nil {
			changed = true
			ks.cbKeyspace.(*keyspace).deleted = true
			logging.Errorf(" Error retrieving bucket %s", name)
			delete(p.keyspaceCache, name)

		} else if ks.cbKeyspace.(*keyspace).cbbucket.UUID != newbucket.UUID {

			logging.Debugf(" UUid of keyspace %v uuid now %v", ks.cbKeyspace.(*keyspace).cbbucket.UUID, newbucket.UUID)
			// UUID has changed. Update the keyspace struct with the newbucket
			ks.cbKeyspace.(*keyspace).cbbucket = newbucket
		}
		// Not deleted. Check if GSI indexer is available
		if ks.cbKeyspace.(*keyspace).gsiIndexer == nil {
			ks.cbKeyspace.(*keyspace).refreshIndexer(p.store.URL(), p.Name())
		}
	}

	if changed == true {
		p.setPool(newpool)
		p.version++
	}
}

type keyspace struct {
	namespace   *namespace
	name        string
	cbbucket    *cb.Bucket
	deleted     bool
	viewIndexer datastore.Indexer // View index provider
	gsiIndexer  datastore.Indexer // GSI index provider
}

//
// Inferring schemas sometimes requires getting a sample of random documents
// from a keyspace. Ideally this should come through a random traversal of the
// primary index, but until that is available, we need to use the Bucket's
// connection pool of memcached.Clients to request random documents from
// the KV store.
//

func (k *keyspace) GetRandomEntry() (string, value.Value, errors.Error) {
	resp, err := k.cbbucket.GetRandomDoc()

	if err != nil {
		return "", nil, errors.NewCbGetRandomEntryError(err)
	}

	return fmt.Sprintf("%s", resp.Key), value.NewValue(resp.Body), nil
}

func newKeyspace(p *namespace, name string) (datastore.Keyspace, errors.Error) {

	cbNamespace := p.getPool()
	cbbucket, err := cbNamespace.GetBucket(name)

	if err != nil {
		logging.Infof(" keyspace %s not found %v", name, err)
		// go-couchbase caches the buckets
		// to be sure no such bucket exists right now
		// we trigger a refresh
		p.refresh(true)
		cbNamespace = p.getPool()

		// and then check one more time
		logging.Infof(" Retrying bucket %s", name)
		cbbucket, err = cbNamespace.GetBucket(name)
		if err != nil {
			// really no such bucket exists
			return nil, errors.NewCbKeyspaceNotFoundError(err, "keyspace "+name)
		}
	}

	if strings.EqualFold(cbbucket.Type, "memcached") {
		return nil, errors.NewCbBucketTypeNotSupportedError(nil, cbbucket.Type)
	}

	rv := &keyspace{
		namespace: p,
		name:      name,
		cbbucket:  cbbucket,
	}

	// Initialize index providers
	rv.viewIndexer = newViewIndexer(rv)

	logging.Infof("Created New Bucket %s", name)

	//discover existing indexes
	if ierr := rv.loadIndexes(); ierr != nil {
		logging.Warnf("Error loading indexes for keyspace %s, Error %v", name, ierr)
	}

	var qerr errors.Error
	rv.gsiIndexer, qerr = gsi.NewGSIIndexer(p.store.URL(), p.Name(), name)
	if qerr != nil {
		logging.Warnf("Error loading GSI indexes for keyspace %s. Error %v", name, qerr)
	}

	// Create a bucket updater that will keep the couchbase bucket fresh.
	cbbucket.RunBucketUpdater(p.KeyspaceDeleteCallback)

	return rv, nil
}

// Called by go-couchbase if a configured keyspace is deleted
func (p *namespace) KeyspaceDeleteCallback(name string, err error) {

	p.lock.Lock()
	defer p.lock.Unlock()

	ks, ok := p.keyspaceCache[name]
	if ok && ks.cbKeyspace != nil {
		logging.Infof("Keyspace %v being deleted", name)
		ks.cbKeyspace.(*keyspace).deleted = true
		delete(p.keyspaceCache, name)

	} else {
		logging.Warnf("Keyspace %v not configured on this server", name)
	}
}

func (b *keyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *keyspace) Namespace() datastore.Namespace {
	return b.namespace
}

func (b *keyspace) Id() string {
	return b.Name()
}

func (b *keyspace) Name() string {
	return b.name
}

func (b *keyspace) Count(context datastore.QueryContext) (int64, errors.Error) {

	var staterr error
	var totalCount int64

	// this is not an ideal implementation. We will change this when
	// gocouchbase implements a mechanism to detect cluster changes

	ns := b.namespace.getPool()
	cbBucket, err := ns.GetBucket(b.Name())
	if err != nil {
		return 0, errors.NewCbKeyspaceNotFoundError(nil, b.Name())
	}
	defer cbBucket.Close()

	statsMap := cbBucket.GetStats("")
	for _, stats := range statsMap {

		itemCount := stats["curr_items"]
		count, err := strconv.Atoi(itemCount)
		if err != nil {
			staterr = err
			break
		} else {
			totalCount = totalCount + int64(count)
		}
	}

	if staterr == nil {
		return totalCount, nil
	}

	return 0, errors.NewCbKeyspaceCountError(nil, "keyspace "+b.Name()+"Error "+staterr.Error())

}

func (b *keyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	switch name {
	case datastore.GSI, datastore.DEFAULT:
		if b.gsiIndexer != nil {
			return b.gsiIndexer, nil
		}
		return nil, errors.NewCbIndexerNotImplementedError(nil, fmt.Sprintf("GSI may not be enabled"))
	case datastore.VIEW:
		return b.viewIndexer, nil
	default:
		return nil, errors.NewCbIndexerNotImplementedError(nil, fmt.Sprintf("Type %s", name))
	}
}

func (b *keyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	indexers := make([]datastore.Indexer, 0, 2)
	if b.gsiIndexer != nil {
		indexers = append(indexers, b.gsiIndexer)
	}

	indexers = append(indexers, b.viewIndexer)
	return indexers, nil
}

func (b *keyspace) Fetch(keys []string, context datastore.QueryContext, subPaths []string) ([]value.AnnotatedPair, []errors.Error) {
	var bulkResponse map[string]*gomemcached.MCResponse
	var mcr *gomemcached.MCResponse
	var keyCount map[string]int
	var err error

	_subPaths := subPaths
	noVirtualDocAttr := false

	if len(_subPaths) > 0 && _subPaths[0] != "$document" {
		_subPaths = append([]string{"$document"}, _subPaths...)
		noVirtualDocAttr = true
	}

	l := len(keys)
	if l == 0 {
		return nil, nil
	}

	if l == 1 {
		mcr, err = b.cbbucket.GetsMC(keys[0], context.GetReqDeadline(), _subPaths)
	} else {
		bulkResponse, keyCount, err = b.cbbucket.GetBulk(keys, context.GetReqDeadline(), _subPaths)
		defer b.cbbucket.ReleaseGetBulkPools(keyCount, bulkResponse)
	}

	if err != nil {
		// Ignore "Not found" keys
		if !isNotFoundError(err) {
			if cb.IsReadTimeOutError(err) {
				logging.Errorf(err.Error())
			}
			return nil, []errors.Error{errors.NewCbBulkGetError(err, "")}
		}
	}

	i := 0
	rv := make([]value.AnnotatedPair, 0, l)
	if l == 1 {
		if mcr != nil && err == nil {
			if len(_subPaths) > 0 {
				rv = append(rv, getSubDocFetchResults(keys[0], mcr, _subPaths, noVirtualDocAttr))
			} else {
				rv = append(rv, doFetch(keys[0], mcr))
			}
			i = 1
		}
	} else {
		if len(_subPaths) > 0 {
			for k, v := range bulkResponse {
				for j := 0; j < keyCount[k]; j++ {
					rv = append(rv, getSubDocFetchResults(k, v, _subPaths, noVirtualDocAttr))
					i++
				}
			}
		} else {
			for k, v := range bulkResponse {
				for j := 0; j < keyCount[k]; j++ {
					rv = append(rv, doFetch(k, v))
					i++
				}
			}
		}

	}

	logging.Debugf("Fetched %d keys ", i)

	return rv, nil
}

func doFetch(k string, v *gomemcached.MCResponse) value.AnnotatedPair {

	var doc value.AnnotatedPair
	doc.Name = k

	val := value.NewAnnotatedValue(value.NewParsedValue(v.Body, false))
	flags := binary.BigEndian.Uint32(v.Extras[0:4])

	expiration := uint32(0)
	if len(v.Extras) >= 8 {
		expiration = binary.BigEndian.Uint32(v.Extras[4:8])
	}

	meta_type := "json"
	if val.Type() == value.BINARY {
		meta_type = "base64"
	}

	val.SetAttachment("meta", map[string]interface{}{
		"id":         k,
		"cas":        v.Cas,
		"type":       meta_type,
		"flags":      flags,
		"expiration": expiration,
	})

	// Uncomment when needed
	//logging.Debugf("CAS Value for key %v is %v flags %v", k, uint64(v.Cas), meta_flags)

	doc.Value = val
	return doc
}

func getSubDocFetchResults(k string, v *gomemcached.MCResponse, subPaths []string, noVirtualDocAttr bool) value.AnnotatedPair {
	var doc value.AnnotatedPair
	doc.Name = k

	responseIter := 0
	i := 0
	xVal := map[string]interface{}{}

	for i < len(subPaths) {
		// For the xattr contents - $document
		xattrError := gomemcached.Status(binary.BigEndian.Uint16(v.Body[responseIter+0:]))
		xattrValueLen := int(binary.BigEndian.Uint32(v.Body[responseIter+2:]))

		xattrValue := v.Body[responseIter+6 : responseIter+6+xattrValueLen]

		// When xattr value not defined for a doc, set missing
		tmpVal := value.NewValue(xattrValue)

		if xattrError != gomemcached.SUBDOC_PATH_NOT_FOUND {
			xVal[subPaths[i]] = tmpVal.Actual()
		}

		// Calculate actual doc value
		responseIter = responseIter + 6 + xattrValueLen
		i = i + 1
	}

	// For the actual document contents -
	respError := gomemcached.Status(binary.BigEndian.Uint16(v.Body[responseIter+0:]))
	respValueLen := int(binary.BigEndian.Uint32(v.Body[responseIter+2:]))

	respValue := v.Body[responseIter+6 : responseIter+6+respValueLen]

	// For deleted documents with respError path not found set to null
	var val value.AnnotatedValue

	// For non deleted documents
	if respError == gomemcached.SUBDOC_PATH_NOT_FOUND {
		// Final Doc value
		val = value.NewAnnotatedValue(nil)
	} else {
		val = value.NewAnnotatedValue(value.NewParsedValue(respValue, false))
	}

	// type
	meta_type := "json"
	if val.Type() == value.BINARY {
		meta_type = "base64"
	}

	// Get flags and expiration from the $document virtual xattrs
	docMeta := xVal["$document"].(map[string]interface{})

	// Convert unmarshalled int64 values to uint32
	flags := uint32(value.NewValue(docMeta["flags"]).(value.NumberValue).Int64())
	exptime := uint32(value.NewValue(docMeta["exptime"]).(value.NumberValue).Int64())

	if noVirtualDocAttr {
		delete(xVal, "$document")
	}

	a := map[string]interface{}{
		"id":         k,
		"cas":        v.Cas,
		"type":       meta_type,
		"flags":      flags,
		"expiration": exptime,
	}

	if len(xVal) > 0 {
		a["xattrs"] = xVal
	}

	val.SetAttachment("meta", a)

	doc.Value = val
	return doc
}

const (
	INSERT = 0x01
	UPDATE = 0x02
	UPSERT = 0x04
)

func opToString(op int) string {

	switch op {
	case INSERT:
		return "insert"
	case UPDATE:
		return "update"
	case UPSERT:
		return "upsert"
	}

	return "unknown operation"
}

func isNotFoundError(err error) bool {
	return cb.IsKeyNoEntError(err)
}

func isEExistError(err error) bool {
	return cb.IsKeyEExistsError(err)
}

func getMeta(key string, meta map[string]interface{}) (cas uint64, flags uint32, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Recovered in f", r)
		}
	}()

	if _, ok := meta["cas"]; ok {
		cas = meta["cas"].(uint64)
	} else {
		return 0, 0, fmt.Errorf("Cas value not found for key %v", key)
	}

	if _, ok := meta["flags"]; ok {
		flags = meta["flags"].(uint32)
	} else {
		return 0, 0, fmt.Errorf("Flags value not found for key %v", key)
	}

	return cas, flags, nil

}

func (b *keyspace) performOp(op int, inserts []value.Pair) ([]value.Pair, errors.Error) {

	if len(inserts) == 0 {
		return nil, nil
	}

	insertedKeys := make([]value.Pair, 0, len(inserts))
	var err error

	for _, kv := range inserts {
		key := kv.Name
		val := kv.Value.ActualForIndex()

		//mv := kv.Value.GetAttachment("meta")

		// TODO Need to also set meta
		switch op {

		case INSERT:
			var added bool
			// add the key to the backend
			added, err = b.cbbucket.Add(key, 0, val)
			if added == false {
				// false & err == nil => given key aready exists in the bucket
				if err != nil {
					err = errors.NewError(err, "Key "+key)
				} else {
					err = errors.NewError(nil, "Duplicate Key "+key)
				}
			}
		case UPDATE:
			// check if the key exists and if so then use the cas value
			// to update the key
			var meta map[string]interface{}
			var cas uint64
			var flags uint32

			an := kv.Value.(value.AnnotatedValue)
			meta = an.GetAttachment("meta").(map[string]interface{})

			cas, flags, err = getMeta(key, meta)
			if err != nil {
				// Don't perform the update if the meta values are not found
				logging.Errorf("Failed to get meta values for key <ud>%v</ud>, error %v", key, err)
			} else {

				logging.Debugf("CAS Value (Update) for key <ud>%v</ud> is %v flags <ud>%v</ud> value <ud>%v</ud>", key, uint64(cas), flags, val)
				_, _, err = b.cbbucket.CasWithMeta(key, int(flags), 0, uint64(cas), val)
			}

		case UPSERT:
			err = b.cbbucket.Set(key, 0, val)
		}

		if err != nil {
			if isEExistError(err) {
				logging.Errorf("Failed to perform update on key <ud>%s</ud>. CAS mismatch due to concurrent modifications", key)
			} else {
				logging.Errorf("Failed to perform <ud>%s</ud> on key <ud>%s</ud> for Keyspace %s.", opToString(op), key, b.Name())
			}
		} else {
			insertedKeys = append(insertedKeys, kv)
		}
	}

	if len(insertedKeys) == 0 {
		return nil, errors.NewCbDMLError(err, "Failed to perform "+opToString(op))
	}

	return insertedKeys, nil

}

func (b *keyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	return b.performOp(INSERT, inserts)

}

func (b *keyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	return b.performOp(UPDATE, updates)
}

func (b *keyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	return b.performOp(UPSERT, upserts)
}

func (b *keyspace) Delete(deletes []string, context datastore.QueryContext) ([]string, errors.Error) {

	failedDeletes := make([]string, 0)
	actualDeletes := make([]string, 0)
	var err error
	for _, key := range deletes {
		if err = b.cbbucket.Delete(key); err != nil {
			if !isNotFoundError(err) {
				logging.Infof("Failed to delete key <ud>%s</ud> Error %s", key, err)
				failedDeletes = append(failedDeletes, key)
			}
		} else {
			actualDeletes = append(actualDeletes, key)
		}
	}

	if len(failedDeletes) > 0 {
		return actualDeletes, errors.NewCbDeleteFailedError(err, "Some keys were not deleted "+fmt.Sprintf("%v", failedDeletes))
	}

	return actualDeletes, nil
}

func (b *keyspace) Release() {
	b.deleted = true
	b.cbbucket.Close()
}

func (b *keyspace) refreshIndexer(url string, poolName string) {
	var err error
	b.gsiIndexer, err = gsi.NewGSIIndexer(url, poolName, b.Name())
	if err == nil {
		logging.Infof(" GSI Indexer loaded ")
	}
}

func (b *keyspace) loadIndexes() (err errors.Error) {
	viewIndexer := b.viewIndexer.(*viewIndexer)
	if err1 := viewIndexer.loadViewIndexes(); err1 != nil {
		err = err1
	}
	return
}

// primaryIndex performs full keyspace scans.
type primaryIndex struct {
	viewIndex
}

func (pi *primaryIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *primaryIndex) Id() string {
	return pi.Name()
}

func (pi *primaryIndex) Name() string {
	return pi.name
}

func (pi *primaryIndex) Type() datastore.IndexType {
	return pi.viewIndex.Type()
}

func (pi *primaryIndex) SeekKey() expression.Expressions {
	return pi.viewIndex.SeekKey()
}

func (pi *primaryIndex) RangeKey() expression.Expressions {
	return pi.viewIndex.RangeKey()
}

func (pi *primaryIndex) Condition() expression.Expression {
	return pi.viewIndex.Condition()
}

func (pi *primaryIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return pi.viewIndex.State()
}

func (pi *primaryIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return pi.viewIndex.Statistics(requestId, span)
}

func (pi *primaryIndex) Drop(requestId string) errors.Error {
	return pi.viewIndex.Drop(requestId)
}

func (pi *primaryIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	pi.viewIndex.Scan(requestId, span, distinct, limit, cons, vector, conn)
}

func (pi *primaryIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	pi.viewIndex.ScanEntries(requestId, limit, cons, vector, conn)
}
