//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package mock provides a fake, mock 100%-in-memory implementation of
the catalog package, which can be useful for testing.  Because it is
memory-oriented, performance testing of higher layers may be easier
with this mock catalog.

*/
package mock

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

const (
	DEFAULT_NUM_POOLS   = 1
	DEFAULT_NUM_BUCKETS = 1
	DEFAULT_NUM_ITEMS   = 100000
)

// site is the root for the mock-based Site.
type site struct {
	path      string
	pools     map[string]*pool
	poolNames []string
	params    map[string]int
}

func (s *site) Id() string {
	return s.URL()
}

func (s *site) URL() string {
	return "mock:" + s.path
}

func (s *site) PoolIds() ([]string, err.Error) {
	return s.PoolNames()
}

func (s *site) PoolNames() ([]string, err.Error) {
	return s.poolNames, nil
}

func (s *site) PoolById(id string) (p catalog.Pool, e err.Error) {
	return s.PoolByName(id)
}

func (s *site) PoolByName(name string) (p catalog.Pool, e err.Error) {
	p, ok := s.pools[name]
	if !ok {
		p, e = nil, err.NewError(nil, "Pool "+name+" not found.")
	}

	return
}

// pool represents a mock-based Pool.
type pool struct {
	site        *site
	name        string
	buckets     map[string]*bucket
	bucketNames []string
}

func (p *pool) SiteId() string {
	return p.site.Id()
}

func (p *pool) Id() string {
	return p.Name()
}

func (p *pool) Name() string {
	return p.name
}

func (p *pool) BucketIds() ([]string, err.Error) {
	return p.BucketNames()
}

func (p *pool) BucketNames() ([]string, err.Error) {
	return p.bucketNames, nil
}

func (p *pool) BucketById(id string) (b catalog.Bucket, e err.Error) {
	return p.BucketByName(id)
}

func (p *pool) BucketByName(name string) (b catalog.Bucket, e err.Error) {
	b, ok := p.buckets[name]
	if !ok {
		b, e = nil, err.NewError(nil, "Bucket "+name+" not found.")
	}

	return
}

// bucket is a mock-based bucket.
type bucket struct {
	pool    *pool
	name    string
	nitems  int
	indexes map[string]catalog.Index
	primary catalog.PrimaryIndex
}

func (b *bucket) PoolId() string {
	return b.pool.Id()
}

func (b *bucket) Id() string {
	return b.Name()
}

func (b *bucket) Name() string {
	return b.name
}

func (b *bucket) Count() (int64, err.Error) {
	return int64(b.nitems), nil
}

func (b *bucket) IndexIds() ([]string, err.Error) {
	return b.IndexNames()
}

func (b *bucket) IndexNames() ([]string, err.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *bucket) IndexById(id string) (catalog.Index, err.Error) {
	return b.IndexByName(id)
}

func (b *bucket) IndexByName(name string) (catalog.Index, err.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, err.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *bucket) IndexByPrimary() (catalog.PrimaryIndex, err.Error) {
	return b.primary, nil
}

func (b *bucket) Indexes() ([]catalog.Index, err.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *bucket) CreatePrimaryIndex() (catalog.PrimaryIndex, err.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, err.NewError(nil, "Not supported.")
}

func (b *bucket) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, err.Error) {
	return nil, err.NewError(nil, "Not supported.")
}

func (b *bucket) Fetch(keys []string) ([]catalog.Pair, err.Error) {
	rv := make([]catalog.Pair, len(keys))
	for i, k := range keys {
		item, e := b.FetchOne(k)
		if e != nil {
			return nil, e
		}

		rv[i].Key = k
		rv[i].Value = item
	}

	return rv, nil
}

func (b *bucket) FetchOne(key string) (value.Value, err.Error) {
	i, e := strconv.Atoi(key)
	if e != nil {
		return nil, err.NewError(e, fmt.Sprintf("no mock item: %v", key))
	} else {
		return genItem(i, b.nitems)
	}
}

// generate a mock document - used by FetchOne to mock a document in the bucket
func genItem(i int, nitems int) (value.Value, err.Error) {
	if i < 0 || i >= nitems {
		return nil, err.NewError(nil,
			fmt.Sprintf("item out of mock range: %v [0,%v)", i, nitems))
	}
	id := strconv.Itoa(i)
	doc := value.NewAnnotatedValue(map[string]interface{}{"id": id, "i": float64(i)})
	doc.SetAttachment("meta", map[string]interface{}{"id": id})
	return doc, nil
}

func (b *bucket) Insert(inserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *bucket) Update(updates []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *bucket) Upsert(upserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *bucket) Delete(deletes []string) err.Error {
	// FIXME
	return err.NewError(nil, "Not yet implemented.")
}

func (b *bucket) Release() {
}

// NewSite creates a new mock site for the given "path".  The path has
// prefix "mock:", with the rest of the path treated as a
// comma-separated key=value params.  For example:
//     mock:pools=2,buckets=5,items=50000
// The above means 2 pools.
// And, each pool has 5 buckets.
// And, each bucket with 50000 items.
// By default, you get...
//     mock:pools=1,buckets=1,items=100000
// Which is what you'd get by specifying a path of just...
//     mock:
func NewSite(path string) (catalog.Site, err.Error) {
	if strings.HasPrefix(path, "mock:") {
		path = path[5:]
	}
	params := map[string]int{}
	for _, kv := range strings.Split(path, ",") {
		if kv == "" {
			continue
		}
		pair := strings.Split(kv, "=")
		v, e := strconv.Atoi(pair[1])
		if e != nil {
			return nil, err.NewError(e,
				fmt.Sprintf("could not parse mock param key: %s, val: %s",
					pair[0], pair[1]))
		}
		params[pair[0]] = v
	}
	npools := paramVal(params, "pools", DEFAULT_NUM_POOLS)
	nbuckets := paramVal(params, "buckets", DEFAULT_NUM_BUCKETS)
	nitems := paramVal(params, "items", DEFAULT_NUM_ITEMS)
	s := &site{path: path, params: params, pools: map[string]*pool{}, poolNames: []string{}}
	for i := 0; i < npools; i++ {
		p := &pool{site: s, name: "p" + strconv.Itoa(i), buckets: map[string]*bucket{}, bucketNames: []string{}}
		for j := 0; j < nbuckets; j++ {
			b := &bucket{pool: p, name: "b" + strconv.Itoa(j), nitems: nitems,
				indexes: map[string]catalog.Index{}}
			pi := &primaryIndex{name: "all_docs", bucket: b}
			b.primary = pi
			b.indexes["all_docs"] = pi
			p.buckets[b.name] = b
			p.bucketNames = append(p.bucketNames, b.name)
		}
		s.pools[p.name] = p
		s.poolNames = append(s.poolNames, p.name)
	}
	return s, nil
}

func paramVal(params map[string]int, key string, defaultVal int) int {
	v, ok := params[key]
	if ok {
		return v
	}
	return defaultVal
}

// primaryIndex performs full bucket scans.
type primaryIndex struct {
	name   string
	bucket *bucket
}

func (pi *primaryIndex) BucketId() string {
	return pi.bucket.Id()
}

func (pi *primaryIndex) Id() string {
	return pi.Name()
}

func (pi *primaryIndex) Name() string {
	return pi.name
}

func (pi *primaryIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *primaryIndex) Drop() err.Error {
	return err.NewError(nil, "This primary index cannot be dropped.")
}

func (pi *primaryIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *primaryIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *primaryIndex) Condition() expression.Expression {
	return nil
}

func (pi *primaryIndex) Statistics(span *catalog.Span) (catalog.Statistics, err.Error) {
	return nil, nil
}

func (pi *primaryIndex) Scan(span *catalog.Span, distinct bool, limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())
	// For primary indexes, bounds must always be strings, so we
	// can just enforce that directly
	low, high := "", ""

	// Ensure that lower bound is a string, if any
	if len(span.Range.Low) > 0 {
		a := span.Range.Low[0].Actual()
		switch a := a.(type) {
		case string:
			low = a
		default:
			conn.SendError(err.NewError(nil, fmt.Sprintf("Invalid lower bound %v of type %T.", a, a)))
			return
		}
	}

	// Ensure that upper bound is a string, if any
	if len(span.Range.High) > 0 {
		a := span.Range.High[0].Actual()
		switch a := a.(type) {
		case string:
			high = a
		default:
			conn.SendError(err.NewError(nil, fmt.Sprintf("Invalid upper bound %v of type %T.", a, a)))
			return
		}
	}

	if limit == 0 {
		limit = int64(pi.bucket.nitems)
	}
	for i := 0; i < pi.bucket.nitems && int64(i) < limit; i++ {
		id := strconv.Itoa(i)

		if low != "" &&
			(id < low ||
				(id == low && (span.Range.Inclusion&catalog.LOW == 0))) {
			continue
		}

		low = ""

		if high != "" &&
			(id > high ||
				(id == high && (span.Range.Inclusion&catalog.HIGH == 0))) {
			break
		}

		entry := catalog.IndexEntry{PrimaryKey: id}
		conn.EntryChannel() <- &entry
	}
}

func (pi *primaryIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	if limit == 0 {
		limit = int64(pi.bucket.nitems)
	}

	for i := 0; i < pi.bucket.nitems && int64(i) < limit; i++ {
		entry := catalog.IndexEntry{PrimaryKey: strconv.Itoa(i)}
		conn.EntryChannel() <- &entry
	}
}
