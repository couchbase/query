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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type indexbucket struct {
	pool    *pool
	name    string
	indexes map[string]catalog.Index
	primary catalog.PrimaryIndex
}

func (b *indexbucket) Release() {
}

func (b *indexbucket) PoolId() string {
	return b.pool.Id()
}

func (b *indexbucket) Id() string {
	return b.Name()
}

func (b *indexbucket) Name() string {
	return b.name
}

func (b *indexbucket) Count() (int64, err.Error) {
	count := int64(0)
	poolIds, excp := b.pool.site.actualSite.PoolIds()
	if excp == nil {
		for _, poolId := range poolIds {
			pool, excp := b.pool.site.actualSite.PoolById(poolId)
			if excp == nil {
				bucketIds, excp := pool.BucketIds()
				if excp == nil {
					for _, bucketId := range bucketIds {
						bucket, excp := pool.BucketById(bucketId)
						if excp == nil {
							indexIds, excp := bucket.IndexIds()
							if excp == nil {
								count += int64(len(indexIds))
							} else {
								return 0, err.NewError(excp, "")
							}
						} else {
							return 0, err.NewError(excp, "")
						}
					}
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

func (b *indexbucket) IndexIds() ([]string, err.Error) {
	return b.IndexNames()
}

func (b *indexbucket) IndexNames() ([]string, err.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *indexbucket) IndexById(id string) (catalog.Index, err.Error) {
	return b.IndexByName(id)
}

func (b *indexbucket) IndexByName(name string) (catalog.Index, err.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, err.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *indexbucket) IndexByPrimary() (catalog.PrimaryIndex, err.Error) {
	return b.primary, nil
}

func (b *indexbucket) Indexes() ([]catalog.Index, err.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *indexbucket) CreatePrimaryIndex() (catalog.PrimaryIndex, err.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, err.NewError(nil, "Not supported.")
}

func (b *indexbucket) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, err.Error) {
	return nil, err.NewError(nil, "Not supported.")
}

func (b *indexbucket) Fetch(keys []string) ([]catalog.Pair, err.Error) {
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

func (b *indexbucket) FetchOne(key string) (value.Value, err.Error) {
	ids := strings.SplitN(key, "/", 3)

	pool, err := b.pool.site.actualSite.PoolById(ids[0])
	if pool != nil {
		bucket, _ := pool.BucketById(ids[1])
		if bucket != nil {
			index, _ := bucket.IndexById(ids[2])
			if index != nil {
				doc := value.NewValue(map[string]interface{}{
					"id":         index.Id(),
					"name":       index.Name(),
					"bucket_id":  bucket.Id(),
					"pool_id":    pool.Id(),
					"site_id":    b.pool.site.actualSite.Id(),
					"index_key":  catalogObjectToJSONSafe(indexKeyToIndexKeyStringArray(index.EqualKey())),
					"index_type": catalogObjectToJSONSafe(index.Type()),
				})
				return doc, nil
			}
		}
	}
	return nil, err
}

func indexKeyToIndexKeyStringArray(key expression.Expressions) []string {
	rv := make([]string, len(key))
	for i, kp := range key {
		// TODO: Determine if Expression needs to implement fmt.Stringer per ast.Expression in dp3
		// rv[i] = kp.String()
		rv[i] = fmt.Sprintf("%v", kp)
	}
	return rv
}

func catalogObjectToJSONSafe(catobj interface{}) interface{} {
	var rv interface{}
	bytes, err := json.Marshal(catobj)
	if err == nil {
		json.Unmarshal(bytes, &rv)
	}
	return rv
}

func newIndexesBucket(p *pool) (*indexbucket, err.Error) {
	b := new(indexbucket)
	b.pool = p
	b.name = BUCKET_NAME_INDEXES

	b.primary = &indexIndex{name: "primary", bucket: b}

	return b, nil
}

func (b *indexbucket) Insert(inserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *indexbucket) Update(updates []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *indexbucket) Upsert(upserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *indexbucket) Delete(deletes []string) err.Error {
	// FIXME
	return err.NewError(nil, "Not yet implemented.")
}

type indexIndex struct {
	name   string
	bucket *indexbucket
}

func (pi *indexIndex) BucketId() string {
	return pi.bucket.Id()
}

func (pi *indexIndex) Id() string {
	return pi.Name()
}

func (pi *indexIndex) Name() string {
	return pi.name
}

func (pi *indexIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *indexIndex) Drop() err.Error {
	return err.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *indexIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *indexIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *indexIndex) Condition() expression.Expression {
	return nil
}

func (pi *indexIndex) Statistics(span *catalog.Span) (catalog.Statistics, err.Error) {
	return nil, nil
}

func (pi *indexIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	poolIds, err := pi.bucket.pool.site.actualSite.PoolIds()
	if err == nil {
		for _, poolId := range poolIds {
			pool, err := pi.bucket.pool.site.actualSite.PoolById(poolId)
			if err == nil {
				bucketIds, err := pool.BucketIds()
				if err == nil {
					for _, bucketId := range bucketIds {
						bucket, err := pool.BucketById(bucketId)
						if err == nil {
							indexIds, err := bucket.IndexIds()
							if err == nil {
								for i, indexId := range indexIds {
									if limit > 0 && int64(i) > limit {
										break
									}

									entry := catalog.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s/%s", poolId, bucketId, indexId)}
									conn.EntryChannel() <- &entry
								}
							}
						}
					}
				}
			}
		}
	}
}

func (pi *indexIndex) Scan(span catalog.Span, limit int64, conn *catalog.IndexConnection) {
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

	ids := strings.SplitN(val, "/", 3)
	if len(ids) != 3 {
		return
	}

	pool, _ := pi.bucket.pool.site.actualSite.PoolById(ids[0])
	if pool == nil {
		return
	}

	bucket, _ := pool.BucketById(ids[1])
	if bucket == nil {
		return
	}

	index, _ := bucket.IndexById(ids[2])
	if bucket != nil {
		entry := catalog.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s/%s", pool.Id(), bucket.Id(), index.Id())}
		conn.EntryChannel() <- &entry
	}
}
