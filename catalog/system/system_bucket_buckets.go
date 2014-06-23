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
	"strings"

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type bucketbucket struct {
	pool    *pool
	name    string
	indexes map[string]catalog.Index
	primary catalog.PrimaryIndex
}

func (b *bucketbucket) Release() {
}

func (b *bucketbucket) PoolId() string {
	return b.pool.Id()
}

func (b *bucketbucket) Id() string {
	return b.Name()
}

func (b *bucketbucket) Name() string {
	return b.name
}

func (b *bucketbucket) Count() (int64, err.Error) {
	count := int64(0)
	poolIds, excp := b.pool.site.actualSite.PoolIds()
	if excp == nil {
		for _, poolId := range poolIds {
			pool, excp := b.pool.site.actualSite.PoolById(poolId)
			if excp == nil {
				bucketIds, excp := pool.BucketIds()
				if excp == nil {
					count += int64(len(bucketIds))
				} else {
					return 0, err.NewError(excp, "")
				}
			} else {
				return 0, err.NewError(excp, "")
			}
		}
		return count, nil
	}
	return 0, err.NewError(excp, "")
}

func (b *bucketbucket) IndexIds() ([]string, err.Error) {
	return b.IndexNames()
}

func (b *bucketbucket) IndexNames() ([]string, err.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *bucketbucket) IndexById(id string) (catalog.Index, err.Error) {
	return b.IndexByName(id)
}

func (b *bucketbucket) IndexByName(name string) (catalog.Index, err.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, err.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *bucketbucket) IndexByPrimary() (catalog.PrimaryIndex, err.Error) {
	return b.primary, nil
}

func (b *bucketbucket) Indexes() ([]catalog.Index, err.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *bucketbucket) CreatePrimaryIndex() (catalog.PrimaryIndex, err.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, err.NewError(nil, "Not supported.")
}

func (b *bucketbucket) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, err.Error) {
	return nil, err.NewError(nil, "Not supported.")
}

func (b *bucketbucket) Fetch(keys []string) ([]catalog.Pair, err.Error) {
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

func (b *bucketbucket) FetchOne(key string) (value.Value, err.Error) {
	ids := strings.SplitN(key, "/", 2)

	pool, err := b.pool.site.actualSite.PoolById(ids[0])
	if pool != nil {
		bucket, _ := pool.BucketById(ids[1])
		if bucket != nil {
			doc := value.NewValue(map[string]interface{}{
				"id":      bucket.Id(),
				"name":    bucket.Name(),
				"pool_id": pool.Id(),
				"site_id": b.pool.site.actualSite.Id(),
			})
			return doc, nil
		}
	}
	return nil, err
}

func (b *bucketbucket) Insert(inserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *bucketbucket) Update(updates []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *bucketbucket) Upsert(upserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *bucketbucket) Delete(deletes []string) err.Error {
	// FIXME
	return err.NewError(nil, "Not yet implemented.")
}

func newBucketsBucket(p *pool) (*bucketbucket, err.Error) {
	b := new(bucketbucket)
	b.pool = p
	b.name = BUCKET_NAME_BUCKETS

	b.primary = &bucketIndex{name: "primary", bucket: b}

	return b, nil
}

type bucketIndex struct {
	name   string
	bucket *bucketbucket
}

func (pi *bucketIndex) BucketId() string {
	return pi.bucket.Id()
}

func (pi *bucketIndex) Id() string {
	return pi.Name()
}

func (pi *bucketIndex) Name() string {
	return pi.name
}

func (pi *bucketIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *bucketIndex) Drop() err.Error {
	return err.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *bucketIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *bucketIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *bucketIndex) Condition() expression.Expression {
	return nil
}

func (pi *bucketIndex) Statistics(span *catalog.Span) (catalog.Statistics, err.Error) {
	return nil, nil
}

func (pi *bucketIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	poolIds, err := pi.bucket.pool.site.actualSite.PoolIds()
	if err == nil {
		for _, poolId := range poolIds {
			pool, err := pi.bucket.pool.site.actualSite.PoolById(poolId)
			if err == nil {
				bucketIds, err := pool.BucketIds()
				if err == nil {
					for i, bucketId := range bucketIds {
						if limit > 0 && int64(i) > limit {
							break
						}
						entry := catalog.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s", poolId, bucketId)}
						conn.EntryChannel() <- &entry
					}
				}
			}
		}
	}
}

func (pi *bucketIndex) Scan(span catalog.Span, limit int64, conn *catalog.IndexConnection) {
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

	ids := strings.SplitN(val, "/", 2)
	if len(ids) != 2 {
		return
	}

	pool, _ := pi.bucket.pool.site.actualSite.PoolById(ids[0])
	if pool == nil {
		return
	}

	bucket, _ := pool.BucketById(ids[1])
	if bucket != nil {
		entry := catalog.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s", pool.Id(), bucket.Id())}
		conn.EntryChannel() <- &entry
	}
}
