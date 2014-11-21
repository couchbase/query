//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package file provides a couchbase-server implementation of the datasite
package.

*/

package couchbase

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"sync"
	"time"

	cb "github.com/couchbaselabs/go-couchbase"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/logging"
	"github.com/couchbaselabs/query/value"
)

const (
	PRIMARY_INDEX = "#primary"
	ALLDOCS_INDEX = "#alldocs"
)

// datasite is the root for the couchbase datasite
type site struct {
	mu             sync.Mutex
	client         cb.Client             // instance of go-couchbase client
	namespaceCache map[string]*namespace // map of pool-names and IDs
}

// NewDatastore creates a new Couchbase site for the given url.
func NewDatastore(url string) (datastore.Datastore, errors.Error) {
	client, err := cb.Connect(url)
	if err != nil {
		return nil, errors.NewError(err, "Cannot connect to url "+url)
	}

	s := &site{
		client:         client,
		namespaceCache: make(map[string]*namespace),
	}
	defaultPool, Err := loadNamespace(s, "default")
	if Err != nil {
		logging.Errorf("Cannot connect to default pool")
		return nil, Err
	}

	s.SetNamespace("default", defaultPool)
	logging.Infof("New site created with url %s", url)
	return s, nil
}

func (s *site) SetClient(client cb.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.client = client
	s.namespaceCache = make(map[string]*namespace)
}

func (s *site) GetPool(name string) (cb.Pool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.client.GetPool(name)
}

func (s *site) GetNamespace(name string) (*namespace, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, ok := s.namespaceCache[name]
	return n, ok
}

func (s *site) SetNamespace(name string, n *namespace) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.namespaceCache[name] = n
}

func (s *site) DelNamespace(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.namespaceCache, name)
}

func (s *site) Id() string {
	return s.URL()
}

func (s *site) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.client.BaseURL.String()
}

func (s *site) NamespaceIds() ([]string, errors.Error) {
	return s.NamespaceNames()
}

func (s *site) NamespaceNames() ([]string, errors.Error) {
	return []string{"default"}, nil
}

func (s *site) NamespaceById(id string) (p datastore.Namespace, e errors.Error) {
	return s.NamespaceByName(id)
}

func (s *site) NamespaceByName(name string) (datastore.Namespace, errors.Error) {
	var err errors.Error

	p, exists := s.GetNamespace(name)
	if !exists {
		if p, err = loadNamespace(s, name); err != nil {
			return nil, err
		}
		s.SetNamespace(name, p)
	}
	return p, nil
}

// a namespace represents a couchbase pool
type namespace struct {
	mu            sync.Mutex
	site          *site
	name          string
	cbNamespace   cb.Pool
	keyspaceCache map[string]datastore.Keyspace
}

func loadNamespace(s *site, name string) (*namespace, errors.Error) {
	msg := fmt.Sprintf("Pool %v not found.", name)
	cbpool, err := s.GetPool(name)
	if err != nil {
		if name == "default" {
			// if default pool is not available, try reconnecting to the server
			client, err := cb.Connect(s.URL())
			if err != nil {
				return nil, errors.NewError(nil, msg)
			}
			// check if the default pool exists
			if cbpool, err = client.GetPool(name); err != nil {
				return nil, errors.NewError(nil, msg)
			}
			s.SetClient(client)

		} else {
			return nil, errors.NewError(nil, msg)
		}
	}

	rv := namespace{
		site:          s,
		name:          name,
		cbNamespace:   cbpool,
		keyspaceCache: make(map[string]datastore.Keyspace),
	}
	go keepPoolFresh(&rv)
	return &rv, nil
}

func (p *namespace) GetCbNamespace() cb.Pool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.cbNamespace
}

func (p *namespace) SetCbNamespace(pool cb.Pool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.cbNamespace = pool
	p.keyspaceCache = make(map[string]datastore.Keyspace)
}

func (p *namespace) GetKeyspace(name string) (datastore.Keyspace, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ks, ok := p.keyspaceCache[name]
	return ks, ok
}

func (p *namespace) SetKeyspace(name string, ks datastore.Keyspace) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.keyspaceCache[name] = ks
}

func (p *namespace) DelKeyspace(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.keyspaceCache, name)
}

func (p *namespace) DatastoreId() string {
	return p.site.Id()
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
	p.mu.Lock()
	defer p.mu.Unlock()

	rv := make([]string, 0, len(p.keyspaceCache))
	for name := range p.keyspaceCache {
		rv = append(rv, name)
	}
	return rv, nil
}

func (p *namespace) KeyspaceByName(
	name string) (b datastore.Keyspace, e errors.Error) {

	b, ok := p.GetKeyspace(name)
	if !ok {
		var err errors.Error
		b, err = newKeyspace(p, name)
		if err != nil {
			return nil, errors.NewError(err, "Keyspace "+name+" name not found")
		}
		p.SetKeyspace(name, b)
	}
	return b, nil
}

func (p *namespace) KeyspaceById(id string) (datastore.Keyspace, errors.Error) {
	return p.KeyspaceByName(id)
}

func (p *namespace) refresh() {
	// trigger refresh of this pool
	logging.Infof(" Refreshing pool %s", p.name)

	pool, err := p.site.GetPool(p.name) // pool gets refreshed here.
	if err != nil {
		logging.Errorf(" Error updating pool name %s: Error %v", p.name, err)
		url := p.site.URL()
		client, err := cb.Connect(url)
		if err != nil {
			logging.Errorf(" Error connecting to URL %s", url)
			return
		}
		// check if the default pool exists
		pool, err = client.GetPool(p.name)
		if err != nil {
			msg := " Retry Failed Error updating pool name %s: Error %v"
			logging.Errorf(msg, p.name, err)
			return
		}
		p.site.SetClient(client)
		p.SetCbNamespace(pool)
	}

	// keyspaces in the pool
	for name := range pool.BucketMap {
		logging.Infof(" Checking keyspace %s", name)
		if _, exists := p.GetKeyspace(name); !exists {
			if b, err := newKeyspace(p, name); err == nil {
				p.SetKeyspace(name, b)
			} else {
				logging.Errorf(" Error creating keyspace %s", name)
			}

		}
	}
}

type keyspace struct {
	mu               sync.Mutex
	namespace        *namespace
	name             string
	cbbucket         *cb.Bucket
	indexes          map[string]datastore.Index
	primary          datastore.PrimaryIndex
	nonUsableIndexes []string // indexes that cannot be used
}

func newKeyspace(p *namespace, name string) (datastore.Keyspace, errors.Error) {
	logging.Infof(" Created New Bucket %s", name)
	pool := p.GetCbNamespace()
	cbbucket, err := pool.GetBucket(name)
	if err != nil {
		// go-couchbase caches the buckets
		// to be sure no such bucket exists right now
		// we trigger a refresh
		p.refresh()
		// and then check one more time
		cbbucket, err = pool.GetBucket(name)
		if err != nil {
			// really no such bucket exists
			msg := fmt.Sprintf("Bucket %v not found.", name)
			return nil, errors.NewError(nil, msg)
		}
	}
	rv := &keyspace{
		namespace:        p,
		name:             name,
		cbbucket:         cbbucket,
		indexes:          make(map[string]datastore.Index),
		nonUsableIndexes: make([]string, 0),
	}
	//discover existing indexes
	indexes, err := rv.loadIndexes()
	if err != nil {
		logging.Warnf("Error loading indexes for keyspace %s: %v", name, err)
	}
	for name, index := range indexes {
		rv.SetIndex(name, index)
	}

	go keepIndexesFresh(rv)
	return rv, nil
}

func (b *keyspace) GetBucket() *cb.Bucket {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.cbbucket
}

func (b *keyspace) SetBucket(cbbucket *cb.Bucket) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cbbucket = cbbucket
}

func (b *keyspace) GetIndex(name string) (datastore.Index, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	index, ok := b.indexes[name]
	return index, ok
}

func (b *keyspace) SetIndex(name string, index datastore.Index) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.indexes[name] = index
	// if index is primary set primary index.
	if name == PRIMARY_INDEX {
		b.primary = index.(datastore.PrimaryIndex)
	}
	logging.Infof("Primary index %T", index)
}

func (b *keyspace) DelIndex(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.indexes, name)
}

func (b *keyspace) GetPrimaryIndex() datastore.PrimaryIndex {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.primary
}

func (b *keyspace) refresh() {
	// trigger refresh of this pool
	logging.Infof(" Refreshing Indexes in keyspace %s", b.name)
	indexes, err := b.loadIndexes()
	if err != nil {
		logging.Errorf(" Error loading indexes for bucket %s", b.name)

	} else {
		for name, index := range indexes {
			b.SetIndex(name, index)
			logging.Infof(" Found index %s on keyspace %s", name, b.name)
		}
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
	statsMap := b.GetBucket().GetStats("")
	for _, stats := range statsMap {
		itemCount := stats["curr_items_tot"]
		if totalCount, err := strconv.Atoi(itemCount); err == nil {
			return int64(totalCount), nil
		}
	}
	return 0, errors.NewError(nil, fmt.Sprintf("Unable to STAT %v", b.name))
}

func (b *keyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *keyspace) IndexNames() ([]string, errors.Error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	rv := make([]string, 0, len(b.indexes))
	for name := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *keyspace) IndexById(id string) (datastore.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *keyspace) IndexByName(name string) (datastore.Index, errors.Error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *keyspace) IndexByPrimary() (datastore.PrimaryIndex, errors.Error) {
	if b.GetPrimaryIndex() == nil {
		indexes, err := b.Indexes()
		if err != nil {
			return nil, err
		}
		if len(indexes) == 0 {
			indexes, err := b.loadIndexes()
			if err != nil {
				return nil, errors.NewError(err, "No indexes found. Please create a primary index")
			}
			for name, index := range indexes {
				b.SetIndex(name, index)
			}
		}
		idx, ok := b.GetIndex(PRIMARY_INDEX)
		if ok {
			primary := idx.(datastore.PrimaryIndex)
			return primary, nil
		}
		all, ok := b.GetIndex(ALLDOCS_INDEX)
		if ok {
			primary := all.(datastore.PrimaryIndex)
			return primary, nil
		}
	}
	return b.GetPrimaryIndex(), nil
}

func (b *keyspace) Indexes() ([]datastore.Index, errors.Error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	rv := make([]datastore.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *keyspace) CreatePrimaryIndex(using datastore.IndexType) (datastore.PrimaryIndex, errors.Error) {
	if _, exists := b.GetIndex(PRIMARY_INDEX); exists {
		return nil, errors.NewError(nil, "Primary index already exists")
	}
	switch using {
	case datastore.VIEW:
		idx, err := newViewPrimaryIndex(b)
		if err != nil {
			return nil, errors.NewError(err, "Error creating primary index")
		}
		b.SetIndex(idx.Name(), idx)
		return idx, nil

	case datastore.LSM:
		idx, err := create2iPrimaryIndex(b, using)
		if err != nil {
			return nil, errors.NewError(err, "")
		}
		logging.Debugf("Created Primary index using 2i `%s`", idx.Name())
		b.SetIndex(idx.Name(), idx)
		return idx, nil

	default:
		return nil, errors.NewError(nil, "Not yet implemented.")
	}
}

func (b *keyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions,
	where expression.Expression, using datastore.IndexType) (datastore.Index, errors.Error) {

	if using == "" {
		// current default is VIEW
		using = datastore.VIEW
	}

	if _, exists := b.GetIndex(name); exists {
		return nil, errors.NewError(nil, fmt.Sprintf("Index already exists: %s", name))
	}

	// if the name matches any of the unusable indexes, return an error
	// TODO: what is non-usable index ?
	for _, iname := range b.nonUsableIndexes {
		if name == iname {
			return nil, errors.NewError(nil, fmt.Sprintf("Index already exists: %s", name))
		}
	}

	switch using {
	case datastore.VIEW:
		idx, err := newViewIndex(name, datastore.IndexKey(rangeKey), where, b)
		if err != nil {
			return nil, errors.NewError(err, fmt.Sprintf("Error creating index: %s", name))
		}
		b.SetIndex(idx.Name(), idx)
		return idx, nil

	case datastore.LSM:
		idx, err := create2iIndex(name, equalKey, rangeKey, where, using, b)
		if err != nil {
			return nil, errors.NewError(err, fmt.Sprintf("Error creating 2i index: %q", name))
		}
		logging.Debugf("Created 2i index %q", idx.Name())
		b.SetIndex(idx.Name(), idx)
		return idx, nil

	default:
		return nil, errors.NewError(nil, "Not yet implemented.")
	}
}

func (b *keyspace) Fetch(keys []string) ([]datastore.AnnotatedPair, errors.Error) {

	if len(keys) == 0 {
		return nil, errors.NewError(nil, "No keys to fetch")
	}

	bulkResponse, err := b.cbbucket.GetBulk(keys)
	if err != nil {
		return nil, errors.NewError(err, "Error doing bulk get")
	}

	i := 0
	rv := make([]datastore.AnnotatedPair, len(bulkResponse))
	for k, v := range bulkResponse {

		var doc datastore.AnnotatedPair
		doc.Key = k

		Value := value.NewAnnotatedValue(value.NewValue(v.Body))

		meta_flags := binary.BigEndian.Uint32(v.Extras[0:4])
		meta_type := "json"
		if Value.Type() == value.BINARY {
			meta_type = "base64"
		}
		Value.SetAttachment("meta", map[string]interface{}{
			"id":    k,
			"cas":   float64(v.Cas),
			"type":  meta_type,
			"flags": float64(meta_flags),
		})

		doc.Value = Value
		rv[i] = doc
		i++

	}

	logging.Debugf("Fetched %d keys ", i)

	return rv, nil
}

func (b *keyspace) FetchOne(key string) (value.AnnotatedValue, errors.Error) {

	item, e := b.Fetch([]string{key})
	if e != nil {
		return nil, e
	}
	// not found
	if len(item) == 0 {
		return nil, nil
	}

	return item[0].Value, e
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

func (b *keyspace) performOp(op int, inserts []datastore.Pair) ([]datastore.Pair, errors.Error) {

	if len(inserts) == 0 {
		return nil, errors.NewError(nil, "No keys to insert")
	}

	insertedKeys := make([]datastore.Pair, 0)
	var err error

	for _, kv := range inserts {
		key := kv.Key
		value := kv.Value.Actual()

		// TODO Need to also set meta
		switch op {

		case INSERT:
			var added bool
			// add the key to the backend
			added, err = b.cbbucket.Add(key, 0, value)
			if added == false {
				err = errors.NewError(nil, "Key "+key+" Exists")
			}
		case UPDATE:
			// check if the key exists and if so then use the cas value
			// to update the key
			rv := map[string]interface{}{}
			var cas uint64

			err = b.cbbucket.Gets(key, &rv, &cas)
			if err == nil {
				err = b.cbbucket.Set(key, 0, value)
			} else {
				logging.Errorf("Failed to insert. Key exists %s", key)
			}
		case UPSERT:
			err = b.cbbucket.Set(key, 0, value)
		}

		if err != nil {
			logging.Errorf("Failed to perform %s on key %s Error %v", opToString(op), key, err)
		} else {
			insertedKeys = append(insertedKeys, kv)
		}
	}

	if len(insertedKeys) == 0 {
		return nil, errors.NewError(err, "Failed to perform "+opToString(op))
	}

	return insertedKeys, nil

}

func (b *keyspace) Insert(inserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	return b.performOp(INSERT, inserts)

}

func (b *keyspace) Update(updates []datastore.Pair) ([]datastore.Pair, errors.Error) {
	return b.performOp(UPDATE, updates)
}

func (b *keyspace) Upsert(upserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	return b.performOp(UPSERT, upserts)
}

func (b *keyspace) Delete(deletes []string) errors.Error {

	failedDeletes := make([]string, 0)
	var err error
	for _, key := range deletes {
		if err = b.cbbucket.Delete(key); err != nil {
			logging.Infof("Failed to delete key %s", key)
			failedDeletes = append(failedDeletes, key)
		}
	}

	if len(failedDeletes) > 0 {
		return errors.NewError(err, "Some keys were not deleted "+fmt.Sprintf("%v", failedDeletes))
	}

	return nil
}

func (b *keyspace) Release() {
	b.cbbucket.Close()
}

func (b *keyspace) loadIndexes() (map[string]datastore.Index, errors.Error) {
	indexes := make(map[string]datastore.Index)
	xs, err := loadViewIndexes(b)
	if err != nil {
		return nil, errors.NewError(nil, err.Error())
	}
	for _, index := range xs {
		indexes[index.Name()] = index
	}

	if xs, err = b.load2iIndexes(); err != nil {
		return nil, errors.NewError(nil, err.Error())
	}
	for _, index := range xs {
		name := index.Name()
		if _, ok := indexes[name]; ok {
			logging.Errorf("Index %q already loaded from view", name)
		} else {
			indexes[name] = index
		}
	}
	return indexes, nil
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

func (pi *primaryIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *primaryIndex) RangeKey() expression.Expressions {
	// FIXME
	return nil
}

func (pi *primaryIndex) Condition() expression.Expression {
	return nil
}

func (pi *primaryIndex) State() (datastore.IndexState, errors.Error) {
	return pi.viewIndex.State()
}

func (pi *primaryIndex) Statistics(span *datastore.Span) (datastore.Statistics, errors.Error) {
	return pi.viewIndex.Statistics(span)
}

func (pi *primaryIndex) Drop() errors.Error {
	return pi.viewIndex.Drop()
}

func (pi *primaryIndex) Scan(span *datastore.Span, distinct bool, limit int64, conn *datastore.IndexConnection) {
	pi.viewIndex.Scan(span, distinct, limit, conn)
}

func (pi *primaryIndex) ScanEntries(limit int64, conn *datastore.IndexConnection) {
	pi.viewIndex.ScanEntries(limit, conn)
}

// go-routine, keep refreshing bucket (aka keyspace).
func keepIndexesFresh(b *keyspace) {
	for _ = range time.Tick(1 * time.Minute) {
		b.refresh()
	}
}

// go-routine, keep refreshing pool (aka namespace).
func keepPoolFresh(p *namespace) {
	for _ = range time.Tick(1 * time.Minute) {
		p.refresh()
	}
}
