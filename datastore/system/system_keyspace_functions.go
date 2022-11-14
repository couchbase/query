//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package system

import (
	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

const _FUNC_PATH = "/query/functions/"

type functionsKeyspace struct {
	keyspaceBase
	name string
	si   datastore.Indexer
}

func (b *functionsKeyspace) Release(close bool) {
}

func (b *functionsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *functionsKeyspace) Id() string {
	return b.Name()
}

func (b *functionsKeyspace) Name() string {
	return b.name
}

func (b *functionsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	children, err := metakv.ListAllChildren(_FUNC_PATH)
	if err != nil {
		return int64(len(children)), nil
	} else {
		return 0, errors.NewMetaKVError("Count", err)
	}
}

func (b *functionsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *functionsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.si, nil
}

func (b *functionsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.si}, nil
}

func (b *functionsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {
	for _, k := range keys {
		item, e := b.fetchOne(k)
		if e != nil {
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, e)
			continue
		}

		if item != nil {
			item.SetAttachment("meta", map[string]interface{}{
				"id": k,
			})
			item.SetId(k)
		}
		keysMap[k] = item
	}

	return
}

func (b *functionsKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	body, _, err := metakv.Get(_FUNC_PATH + key)

	// get does not return is not found, but nil, nil instead
	if err == nil && body == nil {
		return nil, errors.NewSystemDatastoreError(nil, "Key Not Found "+key)
	}
	if err != nil {
		return nil, errors.NewMetaKVError("Fetch", err)
	}
	return value.NewAnnotatedValue(value.NewParsedValue(body, false)), nil
}

// dodgy, but the not found error is not exported in metakv
func isNotFoundError(err error) bool {
	return err != nil && err.Error() == "Not found"
}

func (b *functionsKeyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *functionsKeyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *functionsKeyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *functionsKeyspace) Delete(deletes []string, context datastore.QueryContext) ([]string, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func newFunctionsKeyspace(p *namespace) (*functionsKeyspace, errors.Error) {
	b := new(functionsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p)
	b.name = KEYSPACE_NAME_FUNCTIONS

	primary := &functionsIndex{name: "#primary", keyspace: b}
	b.si = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.si)

	return b, nil
}

type functionsIndex struct {
	indexBase
	name     string
	keyspace *functionsKeyspace
}

func (pi *functionsIndex) KeyspaceId() string {
	return pi.name
}

func (pi *functionsIndex) Id() string {
	return pi.Name()
}

func (pi *functionsIndex) Name() string {
	return pi.name
}

func (pi *functionsIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *functionsIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *functionsIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *functionsIndex) Condition() expression.Expression {
	return nil
}

func (pi *functionsIndex) IsPrimary() bool {
	return true
}

func (pi *functionsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *functionsIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *functionsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, pi.Name())
}

func (pi *functionsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	if span == nil {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
		} else if spanEvaluator.evaluate(pi.keyspace.namespace.store.actualStore.Id()) {
			entry := datastore.IndexEntry{PrimaryKey: pi.keyspace.namespace.store.actualStore.Id()}
			sendSystemKey(conn, &entry)
		}
		conn.Sender().Close()
	}
}

func (pi *functionsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	err := metakv.IterateChildren(_FUNC_PATH, func(path string, value []byte, rev interface{}) error {
		entry := datastore.IndexEntry{PrimaryKey: path[len(_FUNC_PATH):]}
		sendSystemKey(conn, &entry)
		return nil
	})
	if err != nil {
		conn.Error(errors.NewMetaKVIndexError(err))
	}
}
