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

type sitebucket struct {
	pool    *pool
	name    string
	indexes map[string]catalog.Index
	primary catalog.PrimaryIndex
}

func (b *sitebucket) Release() {
}

func (b *sitebucket) PoolId() string {
	return b.pool.Id()
}

func (b *sitebucket) Id() string {
	return b.Name()
}

func (b *sitebucket) Name() string {
	return b.name
}

func (b *sitebucket) Count() (int64, err.Error) {
	return 1, nil
}

func (b *sitebucket) IndexIds() ([]string, err.Error) {
	return b.IndexNames()
}

func (b *sitebucket) IndexNames() ([]string, err.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *sitebucket) IndexById(id string) (catalog.Index, err.Error) {
	return b.IndexByName(id)
}

func (b *sitebucket) IndexByName(name string) (catalog.Index, err.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, err.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *sitebucket) IndexByPrimary() (catalog.PrimaryIndex, err.Error) {
	return b.primary, nil
}

func (b *sitebucket) Indexes() ([]catalog.Index, err.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *sitebucket) CreatePrimaryIndex() (catalog.PrimaryIndex, err.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, err.NewError(nil, "Not supported.")
}

func (b *sitebucket) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, err.Error) {
	return nil, err.NewError(nil, "Not supported.")
}

func (b *sitebucket) Fetch(keys []string) ([]catalog.Pair, err.Error) {
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

func (b *sitebucket) FetchOne(key string) (value.Value, err.Error) {
	if key == b.pool.site.actualSite.Id() {
		doc := value.NewValue(map[string]interface{}{
			"id":  b.pool.site.actualSite.Id(),
			"url": b.pool.site.actualSite.URL(),
		})
		return doc, nil
	}
	return nil, err.NewError(nil, "Not Found")
}

func (b *sitebucket) Insert(inserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *sitebucket) Update(updates []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *sitebucket) Upsert(upserts []catalog.Pair) ([]catalog.Pair, err.Error) {
	// FIXME
	return nil, err.NewError(nil, "Not yet implemented.")
}

func (b *sitebucket) Delete(deletes []string) err.Error {
	// FIXME
	return err.NewError(nil, "Not yet implemented.")
}

func newSitesBucket(p *pool) (*sitebucket, err.Error) {
	b := new(sitebucket)
	b.pool = p
	b.name = BUCKET_NAME_SITES

	b.primary = &siteIndex{name: "primary", bucket: b}

	return b, nil
}

type siteIndex struct {
	name   string
	bucket *sitebucket
}

func (pi *siteIndex) BucketId() string {
	return pi.name
}

func (pi *siteIndex) Id() string {
	return pi.Name()
}

func (pi *siteIndex) Name() string {
	return pi.name
}

func (pi *siteIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *siteIndex) Drop() err.Error {
	return err.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *siteIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *siteIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *siteIndex) Condition() expression.Expression {
	return nil
}

func (pi *siteIndex) Statistics(span *catalog.Span) (catalog.Statistics, err.Error) {
	return nil, nil
}

func (pi *siteIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	entry := catalog.IndexEntry{PrimaryKey: pi.bucket.pool.site.actualSite.Id()}
	conn.EntryChannel() <- &entry
}

func (pi *siteIndex) Scan(span *catalog.Span, limit int64, conn *catalog.IndexConnection) {
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

	if strings.EqualFold(val, pi.bucket.pool.site.actualSite.Id()) {
		entry := catalog.IndexEntry{PrimaryKey: pi.bucket.pool.site.actualSite.Id()}
		conn.EntryChannel() <- &entry
	}
}
