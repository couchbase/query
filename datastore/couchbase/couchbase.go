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
	"strconv"
	"strings"
	"sync"

	"github.com/couchbase/cbauth"
	cb "github.com/couchbase/go-couchbase"
	gsi "github.com/couchbase/indexing/secondary/queryport/n1ql"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

const (
	PRIMARY_INDEX = "#primary"
)

// store is the root for the couchbase datastore
type store struct {
	client         cb.Client             // instance of go-couchbase client
	namespaceCache map[string]*namespace // map of pool-names and IDs
	CbAuthInit     bool                  // whether cbAuth is initialized
	inferencer     datastore.Inferencer  // what we use to infer schemas
}

func (s *store) Id() string {
	return s.URL()
}

func (s *store) URL() string {
	return s.client.BaseURL.String()
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

func doAuth(username, password, bucket string, requested datastore.Privilege) (bool, error) {

	logging.Debugf(" Authenticating for bucket %s username %s password %s", bucket, username, password)
	creds, err := cbauth.Auth(username, password)
	if err != nil {
		return false, err
	}

	if requested == datastore.PRIV_DDL {
		authResult, err := creds.IsAllowed(fmt.Sprintf("cluster.bucket[%s].views!write", bucket))
		if err != nil || authResult == false {
			return false, err
		}

	} else if requested == datastore.PRIV_WRITE {
		authResult, err := creds.IsAllowed(fmt.Sprintf("cluster.bucket[%s].data!write", bucket))
		if err != nil || authResult == false {
			return false, err
		}

	} else if requested == datastore.PRIV_READ {
		authResult, err := creds.IsAllowed(fmt.Sprintf("cluster.bucket[%s].data!read", bucket))
		if err != nil || authResult == false {
			return false, err
		}

	} else {
		return false, fmt.Errorf("Invalid Privileges")
	}

	return true, nil

}

func (s *store) Authorize(privileges datastore.Privileges, credentials datastore.Credentials) errors.Error {
	if s.CbAuthInit == false {
		// cbauth is not initialized. Access to SASL protected buckets will be
		// denied by the couchbase server
		logging.Warnf("CbAuth not intialized")
		return nil
	}

	// Add default authorization -- the privileges every user has.
	if credentials == nil {
		credentials = make(datastore.Credentials)
	}
	credentials[""] = ""

	// if the authentication fails for any of the requested privileges return an error
	for keyspace, privilege := range privileges {
		if strings.Contains(keyspace, ":") {
			q := strings.Split(keyspace, ":")
			pool := q[0]
			keyspace = q[1]

			if strings.EqualFold(pool, "#system") {
				// trying auth on system keyspace
				return nil
			}
		}

		logging.Debugf("Authenticating for keyspace %s", keyspace)

		thisBucketAuthorized := false
		var rememberedError error
		for username, password := range credentials {
			var un string
			userCreds := strings.Split(username, ":")
			if len(userCreds) == 1 {
				un = userCreds[0]
			} else {
				un = userCreds[1]
			}

			logging.Debugf(" Credentials for user %v", un)

			authResult, err := doAuth(un, password, keyspace, privilege)

			// Auth succeeded
			if authResult == true {
				thisBucketAuthorized = true
				break
			} else if err != nil {
				rememberedError = err
			}
		}

		if !thisBucketAuthorized {
			return errors.NewDatastoreAuthorizationError(rememberedError, "Keyspace "+keyspace)
		}
	}

	// If we got this far, every bucket is authorized. Success!
	return nil
}

func (s *store) SetLogLevel(level logging.Level) {
	for _, n := range s.namespaceCache {
		defer n.lock.Unlock()
		n.lock.Lock()
		for _, k := range n.keyspaceCache {
			indexers, _ := k.Indexers()
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

// NewStore creates a new Couchbase store for the given url.
func NewDatastore(u string) (s datastore.Datastore, e errors.Error) {

	var client cb.Client
	var cbAuthInit bool

	// try and initialize cbauth

	c, err := initCbAuth(u)
	if err != nil {
		logging.Errorf(" Unable to initialize cbauth. Error %v", err)
		url, err := url.Parse(u)
		if err != nil {
			return nil, errors.NewCbUrlParseError(err, "url "+u)
		}

		if url.User != nil {
			password, _ := url.User.Password()
			if password == "" {
				logging.Errorf("No password found in url %s", u)
			}

			// intialize cb_auth variables manually
			logging.Infof(" Trying to init cbauth with credentials %s %s", url.Host, url.User.Username())
			set, err := cbauth.InternalRetryDefaultInit(url.Host, url.User.Username(), password)
			if set == false || err != nil {
				logging.Errorf(" Unable to initialize cbauth variables. Error %v", err)
			} else {
				c, err = initCbAuth("http://" + url.Host)
				if err != nil {
					logging.Errorf("Unable to initliaze cbauth.  Error %v", err)
				} else {
					client = *c
					cbAuthInit = true
				}
			}
		}
	} else {
		client = *c
		cbAuthInit = true
	}

	if cbAuthInit == false {
		// connect without auth
		logging.Warnf("Unable to intialize cbAuth, access to couchbase buckets may be restricted")
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
		keyspaceCache: make(map[string]datastore.Keyspace),
	}

	return &rv, nil
}

// a namespace represents a couchbase pool
type namespace struct {
	store         *store
	name          string
	cbNamespace   cb.Pool
	keyspaceCache map[string]datastore.Keyspace
	lock          sync.Mutex   // lock to guard the keyspaceCache
	nslock        sync.RWMutex // lock for this structure
}

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

func (p *namespace) KeyspaceByName(name string) (b datastore.Keyspace, e errors.Error) {

	b, ok := p.keyspaceCache[name]
	if !ok {
		var err errors.Error
		b, err = newKeyspace(p, name)
		if err != nil {
			return nil, err
		}
		p.lock.Lock()
		defer p.lock.Unlock()
		p.keyspaceCache[name] = b
	}
	return b, nil
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
			logging.Errorf("Retry Failed Error updating pool name %s: Error %v", p.name, err)
			return
		}
		p.store.client = client

	}

	p.lock.Lock()
	defer p.lock.Unlock()
	for name, ks := range p.keyspaceCache {
		logging.Debugf(" Checking keyspace %s", name)
		newbucket, err := newpool.GetBucket(name)
		if err != nil {
			changed = true
			ks.(*keyspace).deleted = true
			logging.Errorf(" Error retrieving bucket %s", name)
			delete(p.keyspaceCache, name)

		} else if ks.(*keyspace).cbbucket.UUID != newbucket.UUID {

			logging.Debugf(" UUid of keyspace %v uuid now %v", ks.(*keyspace).cbbucket.UUID, newbucket.UUID)
			// UUID has changed. Update the keyspace struct with the newbucket
			ks.(*keyspace).cbbucket = newbucket
		}
		// Not deleted. Check if GSI indexer is available
		if ks.(*keyspace).gsiIndexer == nil {
			ks.(*keyspace).refreshIndexer(p.store.URL(), p.Name())
		}
	}

	if changed == true {
		p.setPool(newpool)
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
	if ok {
		logging.Infof("Keyspace %v being deleted", name)
		ks.(*keyspace).deleted = true
		delete(p.keyspaceCache, name)

	} else {
		logging.Warnf("Keyspace %v not configured on this server", name)
	}
}

func (b *keyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *keyspace) Id() string {
	return b.Name()
}

func (b *keyspace) Name() string {
	return b.name
}

func (b *keyspace) Count() (int64, errors.Error) {

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

func (b *keyspace) Fetch(keys []string) ([]value.AnnotatedPair, []errors.Error) {

	if len(keys) == 0 {
		return nil, nil
	}

	bulkResponse, err := b.cbbucket.GetBulkAll(keys)
	if err != nil {
		// Ignore "Not found" keys
		if !isNotFoundError(err) {
			return nil, []errors.Error{errors.NewCbBulkGetError(err, "")}
		}
	}

	i := 0
	rv := make([]value.AnnotatedPair, 0, len(keys))
	for k, av := range bulkResponse {
		for _, v := range av {
			var doc value.AnnotatedPair
			doc.Name = k

			Value := value.NewAnnotatedValue(value.NewValue(v.Body))

			meta_flags := binary.BigEndian.Uint32(v.Extras[0:4])
			meta_type := "json"
			if Value.Type() == value.BINARY {
				meta_type = "base64"
			}
			Value.SetAttachment("meta", map[string]interface{}{
				"id":    k,
				"cas":   v.Cas,
				"type":  meta_type,
				"flags": uint32(meta_flags),
			})

			// Uncomment when needed
			//logging.Debugf("CAS Value for key %v is %v flags %v", k, uint64(v.Cas), meta_flags)

			doc.Value = Value
			rv = append(rv, doc)
			i++
		}

	}

	logging.Debugf("Fetched %d keys ", i)

	return rv, nil
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
		val := kv.Value.Actual()

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
				logging.Errorf("Failed to get meta values for key %v, error %v", key, err)
			} else {

				logging.Debugf("CAS Value (Update) for key %v is %v flags %v value %v", key, uint64(cas), flags, val)
				_, _, err = b.cbbucket.CasWithMeta(key, int(flags), 0, uint64(cas), val)
			}

		case UPSERT:
			err = b.cbbucket.Set(key, 0, val)
		}

		if err != nil {
			if isEExistError(err) {
				logging.Errorf("Failed to perform update on key %s. CAS mismatch due to concurrent modifications", key)
			} else {
				logging.Errorf("Failed to perform %s on key %s for Keyspace %s Error %v", opToString(op), key, b.Name(), err)
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

func (b *keyspace) Delete(deletes []string) ([]string, errors.Error) {

	failedDeletes := make([]string, 0)
	actualDeletes := make([]string, 0)
	var err error
	for _, key := range deletes {
		if err = b.cbbucket.Delete(key); err != nil {
			if !isNotFoundError(err) {
				logging.Infof("Failed to delete key %s Error %s", key, err)
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
