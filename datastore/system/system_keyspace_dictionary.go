//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package system

import (
	"github.com/couchbase/query/datastore"
	dictionary "github.com/couchbase/query/datastore/couchbase"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type dictionaryKeyspace struct {
	keyspaceBase
	si datastore.Indexer
}

func (b *dictionaryKeyspace) Release(close bool) {
}

func (b *dictionaryKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *dictionaryKeyspace) Id() string {
	return b.Name()
}

func (b *dictionaryKeyspace) Name() string {
	return b.name
}

func (b *dictionaryKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	count, err := dictionary.Count()
	if err == nil {
		return count, nil
	} else {
		return 0, errors.NewSystemCollectionError("Count from system collection", err)
	}
}

func (b *dictionaryKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *dictionaryKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.si, nil
}

func (b *dictionaryKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.si}, nil
}

func (b *dictionaryKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {
	for _, k := range keys {
		itemMap, e := b.fetchOne(k)
		if e != nil {
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, e)
			continue
		}

		var item value.AnnotatedValue
		if itemMap != nil {
			distributions := itemMap["distributions"]
			delete(itemMap, "distributions")
			if distributions != nil {
				dists := distributions.(map[string]interface{})
				if len(dists) > 0 {
					distKeys := make([]interface{}, 0, len(dists))
					for n, _ := range dists {
						distKeys = append(distKeys, n)
					}
					itemMap["distributionKeys"] = distKeys
				}
			}
			item = value.NewAnnotatedValue(value.NewValue(itemMap))
			meta := item.NewMeta()
			meta["keyspace"] = b.fullName
			meta["distributions"] = distributions
			item.SetId(k)
		}
		keysMap[k] = item
	}

	return
}

func (b *dictionaryKeyspace) fetchOne(key string) (map[string]interface{}, errors.Error) {
	entry, err := dictionary.Get(key)

	// get does not return is not found, but nil, nil instead
	if err == nil && entry == nil {
		return nil, errors.NewSystemDatastoreError(nil, "Key Not Found "+key)
	}
	if err != nil {
		return nil, errors.NewSystemCollectionError("Fetch from system collection", err)
	}
	itemMap := map[string]interface{}{}
	entry.Target(itemMap)
	entry.Dictionary(itemMap)
	return itemMap, nil
}

func (b *dictionaryKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *dictionaryKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *dictionaryKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *dictionaryKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	for _, pair := range deletes {
		name := pair.Name

		// if we are deleting a dictionary entry, we also must remove it
		// from all the n1ql node caches
		distributed.RemoteAccess().DoRemoteOps([]string{}, "dictionary_cache", "DELETE", name, "",
			func(warn errors.Error) {
				context.Warning(warn)
			},
			distributed.NO_CREDS, "")

		dictionary.DropDictionaryEntry(name)
	}
	return deletes, nil
}

func newDictionaryKeyspace(p *namespace, name string) (*dictionaryKeyspace, errors.Error) {
	b := new(dictionaryKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, name)

	primary := &dictionaryIndex{name: "#primary", keyspace: b}
	b.si = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.si)

	return b, nil
}

type dictionaryIndex struct {
	indexBase
	name     string
	keyspace *dictionaryKeyspace
}

func (pi *dictionaryIndex) KeyspaceId() string {
	return pi.name
}

func (pi *dictionaryIndex) Id() string {
	return pi.Name()
}

func (pi *dictionaryIndex) Name() string {
	return pi.name
}

func (pi *dictionaryIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *dictionaryIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *dictionaryIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *dictionaryIndex) Condition() expression.Expression {
	return nil
}

func (pi *dictionaryIndex) IsPrimary() bool {
	return true
}

func (pi *dictionaryIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *dictionaryIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *dictionaryIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, pi.Name())
}

func (pi *dictionaryIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

func (pi *dictionaryIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	err := dictionary.Foreach(func(path string) error {
		entry := datastore.IndexEntry{PrimaryKey: path}
		sendSystemKey(conn, &entry)
		return nil
	})
	if err != nil {
		conn.Error(errors.NewSystemCollectionError("Iterate through system collection", err))
	}
}
