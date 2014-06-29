//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package file provides a file-based implementation of the catalog
package.

*/
package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

// site is the root for the file-based Site.
type site struct {
	path      string
	pools     map[string]*pool
	poolNames []string
}

func (s *site) Id() string {
	return s.path
}

func (s *site) URL() string {
	return "file://" + s.path
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
	p, ok := s.pools[strings.ToUpper(name)]
	if !ok {
		e = err.NewError(nil, "Pool "+name+" not found.")
	}

	return
}

// NewSite creates a new file-based site for the given filepath.
func NewSite(path string) (s catalog.Site, e err.Error) {
	path, er := filepath.Abs(path)
	if er != nil {
		return nil, err.NewError(er, "")
	}

	fs := &site{path: path}

	e = fs.loadPools()
	if e != nil {
		return
	}

	s = fs
	return
}

func (s *site) loadPools() (e err.Error) {
	dirEntries, er := ioutil.ReadDir(s.path)
	if er != nil {
		return err.NewError(er, "")
	}

	s.pools = make(map[string]*pool, len(dirEntries))
	s.poolNames = make([]string, 0, len(dirEntries))

	var p *pool
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			s.poolNames = append(s.poolNames, dirEntry.Name())
			diru := strings.ToUpper(dirEntry.Name())
			if _, ok := s.pools[diru]; ok {
				return err.NewError(nil, "Duplicate pool name "+dirEntry.Name())
			}

			p, e = newPool(s, dirEntry.Name())
			if e != nil {
				return
			}

			s.pools[diru] = p
		}
	}

	return
}

// pool represents a file-based Pool.
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
	b, ok := p.buckets[strings.ToUpper(name)]
	if !ok {
		e = err.NewError(nil, "Bucket "+name+" not found.")
	}

	return
}

func (p *pool) path() string {
	return filepath.Join(p.site.path, p.name)
}

// newPool creates a new pool.
func newPool(s *site, dir string) (p *pool, e err.Error) {
	p = new(pool)
	p.site = s
	p.name = dir

	e = p.loadBuckets()
	return
}

func (p *pool) loadBuckets() (e err.Error) {
	dirEntries, er := ioutil.ReadDir(p.path())
	if er != nil {
		return err.NewError(er, "")
	}

	p.buckets = make(map[string]*bucket, len(dirEntries))
	p.bucketNames = make([]string, 0, len(dirEntries))

	var b *bucket
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			diru := strings.ToUpper(dirEntry.Name())
			if _, ok := p.buckets[diru]; ok {
				return err.NewError(nil, "Duplicate bucket name "+dirEntry.Name())
			}

			b, e = newBucket(p, dirEntry.Name())
			if e != nil {
				return
			}

			p.buckets[diru] = b
			p.bucketNames = append(p.bucketNames, b.Name())
		}
	}

	return
}

// bucket is a file-based bucket.
type bucket struct {
	pool    *pool
	name    string
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
	dirEntries, er := ioutil.ReadDir(b.path())
	if er != nil {
		return 0, err.NewError(er, "")
	}
	return int64(len(dirEntries)), nil
}

func (b *bucket) IndexIds() ([]string, err.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
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
	path := filepath.Join(b.path(), key+".json")
	item, e := fetch(path)
	if e != nil {
		item = nil
	}

	return item, e
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

func (b *bucket) path() string {
	return filepath.Join(b.pool.path(), b.name)
}

// newBucket creates a new bucket.
func newBucket(p *pool, dir string) (b *bucket, e err.Error) {
	b = new(bucket)
	b.pool = p
	b.name = dir

	fi, er := os.Stat(b.path())
	if er != nil {
		return nil, err.NewError(er, "")
	}

	if !fi.IsDir() {
		return nil, err.NewError(nil, "Bucket path must be a directory.")
	}

	b.indexes = make(map[string]catalog.Index, 1)
	pi := new(primaryIndex)
	b.primary = pi
	pi.bucket = b
	pi.name = "#primary"
	b.indexes[pi.name] = pi

	return
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
	// FIXME
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

	dirEntries, er := ioutil.ReadDir(pi.bucket.path())
	if er != nil {
		conn.SendError(err.NewError(er, ""))
		return
	}

	var n int64 = 0
	for _, dirEntry := range dirEntries {
		if limit > 0 && n > limit {
			break
		}

		id := documentPathToId(dirEntry.Name())

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

		if !dirEntry.IsDir() {
			entry := catalog.IndexEntry{PrimaryKey: id}
			conn.EntryChannel() <- &entry
			n++
		}
	}
}

func (pi *primaryIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	dirEntries, er := ioutil.ReadDir(pi.bucket.path())
	if er != nil {
		conn.SendError(err.NewError(er, ""))
		return
	}

	for i, dirEntry := range dirEntries {
		if limit > 0 && int64(i) > limit {
			break
		}
		if !dirEntry.IsDir() {
			entry := catalog.IndexEntry{PrimaryKey: documentPathToId(dirEntry.Name())}
			conn.EntryChannel() <- &entry
		}
	}
}

func fetch(path string) (item value.Value, e err.Error) {
	bytes, er := ioutil.ReadFile(path)
	if er != nil {
		if os.IsNotExist(er) {
			// file doesn't exist should simply return nil, nil
			return
		}
		return nil, err.NewError(er, "")
	}

	doc := value.NewAnnotatedValue(value.NewValueFromBytes(bytes))
	doc.SetAttachment("meta", map[string]interface{}{"id": documentPathToId(path)})
	item = doc

	return
}

func documentPathToId(p string) string {
	_, file := filepath.Split(p)
	ext := filepath.Ext(file)
	return file[0 : len(file)-len(ext)]
}
