//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package system

import (
	"fmt"

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type poolbucket struct {
	pool    *pool
	name    string
	indexes map[string]catalog.Index
	primary catalog.PrimaryIndex
}

func (b *poolbucket) Release() {
}

func (b *poolbucket) PoolId() string {
	return b.pool.Id()
}

func (b *poolbucket) Id() string {
	return b.Name()
}

func (b *poolbucket) Name() string {
	return b.name
}

func (b *poolbucket) Count() (int64, err.Error) {
	poolIds, excp := b.pool.site.actualSite.PoolIds()
	if excp == nil {
		return int64(len(poolIds)), nil
	}
	return 0, err.NewError(excp, "")
}

func (b *poolbucket) IndexIds() ([]string, err.Error) {
	return b.IndexNames()
}

func (b *poolbucket) IndexNames() ([]string, err.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *poolbucket) IndexById(id string) (catalog.Index, err.Error) {
	return b.IndexByName(id)
}

func (b *poolbucket) IndexByName(name string) (catalog.Index, err.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, err.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *poolbucket) IndexByPrimary() (catalog.PrimaryIndex, err.Error) {
	return b.primary, nil
}

func (b *poolbucket) Indexes() ([]catalog.Index, err.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *poolbucket) Fetch(keys []string) ([]catalog.Pair, err.Error) {
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

func (b *poolbucket) FetchOne(key string) (value.Value, err.Error) {
	pool, excp := b.pool.site.actualSite.PoolById(key)
	if pool != nil {
		doc := value.NewValue(map[string]interface{}{
			"id":      pool.Id(),
			"name":    pool.Name(),
			"site_id": b.pool.site.actualSite.Id(),
		})
		return doc, nil
	}
	return nil, err.NewError(excp, "Not Found")
}

func (b *poolbucket) Insert(inserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *poolbucket) Update(updates []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *poolbucket) Upsert(upserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *poolbucket) Delete(deletes []string) err.Error {
	// FIXME
	return err.NewError(nil, "Not yet implemented.")
}

func (b *poolbucket) CreatePrimaryIndex() (catalog.PrimaryIndex, err.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, err.NewError(nil, "Not supported.")
}

func (b *poolbucket) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, err.Error) {
	return nil, err.NewError(nil, "Not supported.")
}

func newPoolsBucket(p *pool) (*poolbucket, err.Error) {
	b := new(poolbucket)
	b.pool = p
	b.name = BUCKET_NAME_POOLS

	b.primary = &poolIndex{name: "primary", bucket: b}

	return b, nil
}

type poolIndex struct {
	name   string
	bucket *poolbucket
}

func (pi *poolIndex) BucketId() string {
	return pi.bucket.Id()
}

func (pi *poolIndex) Id() string {
	return pi.Name()
}

func (pi *poolIndex) Name() string {
	return pi.name
}

func (pi *poolIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *poolIndex) Drop() err.Error {
	return err.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *poolIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *poolIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *poolIndex) Condition() expression.Expression {
	return nil
}

func (pi *poolIndex) Statistics(span *catalog.Span) (catalog.Statistics, err.Error) {
	return nil, nil
}

func (pi *poolIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	poolIds, err := pi.bucket.pool.site.actualSite.PoolIds()
	if err == nil {
		for i, poolId := range poolIds {
			if limit > 0 && int64(i) > limit {
				break
			}

			entry := catalog.IndexEntry{PrimaryKey: poolId}
			conn.EntryChannel() <- &entry
		}
	}
}

func (pi *poolIndex) Scan(span *catalog.Span, distinct bool, limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	val := ""

	a := span.Equal[0].Actual()
	switch a := a.(type) {
	case string:
		val = a
	default:
		conn.SendError(err.NewError(nil, fmt.Sprintf("Invalid equality value %v of type %T.", a, a)))
		return
	}

	pool, _ := pi.bucket.pool.site.actualSite.PoolById(val)
	if pool != nil {
		entry := catalog.IndexEntry{PrimaryKey: pool.Id()}
		conn.EntryChannel() <- &entry
	}
}
