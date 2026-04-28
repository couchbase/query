//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type catalogsKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *catalogsKeyspace) Release(close bool) {
}

func (b *catalogsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *catalogsKeyspace) Id() string {
	return b.Name()
}

func (b *catalogsKeyspace) Name() string {
	return b.name
}

func getCatalogList(context datastore.QueryContext, s *store, name string, infoFlags uint64) ([]any, errors.Error) {
	cbDatastore, ok := s.actualStore.(datastore.CouchbaseDatastore)
	if !ok {
		return nil, errors.NewDatastoreNotCouchbaseError()
	}

	val, err := cbDatastore.GetCatalogs(context, name, infoFlags)
	if err != nil {
		return nil, err
	}

	data := val.Actual()
	sliceOfCatalogs, ok := data.([]any)
	if !ok {
		return nil, nil
	}

	return sliceOfCatalogs, nil
}

// canReadCatalog checks if user can read a specific catalog
func canReadCatalog(context datastore.QueryContext, catalogName string) bool {
	privs := auth.NewPrivileges()
	privs.Add(catalogName, auth.PRIV_CATALOG_SELECT, auth.PRIV_PROPS_NONE)

	err := datastore.GetDatastore().AuthorizeInternal(privs, context.Credentials())
	return err == nil
}

func (b *catalogsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	sliceOfCatalogs, err := getCatalogList(context, b.namespace.store, "", 0)
	if err != nil {
		return 0, err
	}

	canAccessAll := canAccessSystemTables(context, false)
	count := int64(0)

	mapOfCatalogs := catalogListToMap(sliceOfCatalogs)
	for catalogName := range mapOfCatalogs {
		excludeResult := !canAccessAll && !canReadCatalog(context, catalogName)
		if excludeResult {
			context.Warning(errors.NewSystemFilteredRowsWarning("system:catalogs"))
		} else {
			count++
		}
	}

	return count, nil
}

func (b *catalogsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *catalogsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *catalogsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *catalogsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {

	sliceOfCatalogs, err := getCatalogList(context, b.namespace.store, "", 0)
	if err != nil {
		return []errors.Error{err}
	}

	canAccessAll := canAccessSystemTables(context, false)
	newMap := catalogListToMap(sliceOfCatalogs)

	for _, k := range keys {
		val := newMap[k]
		if val == nil {
			continue
		}

		excludeResult := !canAccessAll && !canReadCatalog(context, k)
		if excludeResult {
			context.Warning(errors.NewSystemFilteredRowsWarning("system:catalogs"))
			continue
		}

		item := value.NewAnnotatedValue(val)
		item.SetMetaField(value.META_KEYSPACE, b.fullName)
		item.SetId(k)
		keysMap[k] = item
	}

	return
}

func catalogListToMap(sliceOfCatalogs []any) map[string]value.Value {
	newMap := make(map[string]value.Value, len(sliceOfCatalogs))
	for _, c := range sliceOfCatalogs {
		catalogAsMap, ok := c.(map[string]any)
		if !ok {
			continue
		}
		name, present := catalogAsMap["name"]
		if !present {
			continue
		}
		nameString, ok := name.(string)
		if !ok {
			continue
		}
		newMap[nameString] = value.NewValue(c)
	}
	return newMap
}

func newCatalogsKeyspace(p *namespace) (*catalogsKeyspace, errors.Error) {
	b := new(catalogsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_CATALOGS)

	primary := &catalogsIndex{name: PRIMARY_INDEX_NAME, keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

type catalogsIndex struct {
	indexBase
	name     string
	keyspace *catalogsKeyspace
}

func (pi *catalogsIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *catalogsIndex) Id() string {
	return pi.Name()
}

func (pi *catalogsIndex) Name() string {
	return pi.name
}

func (pi *catalogsIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *catalogsIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *catalogsIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *catalogsIndex) Condition() expression.Expression {
	return nil
}

func (pi *catalogsIndex) IsPrimary() bool {
	return true
}

func (pi *catalogsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *catalogsIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *catalogsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *catalogsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

func (pi *catalogsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	sliceOfCatalogs, err := getCatalogList(conn.QueryContext(), pi.keyspace.namespace.store, "", 0)
	if err != nil {
		conn.Fatal(err)
		return
	}
	mapOfCatalogs := catalogListToMap(sliceOfCatalogs)

	canAccessAll := canAccessSystemTables(conn.QueryContext(), false)

	var numProduced int64
	for k := range mapOfCatalogs {
		if limit > 0 && numProduced >= limit {
			break
		}

		excludeResult := !canAccessAll && !canReadCatalog(conn.QueryContext(), k)
		if excludeResult {
			conn.QueryContext().Warning(errors.NewSystemFilteredRowsWarning("system:catalogs"))
			continue
		}

		entry := datastore.IndexEntry{PrimaryKey: k}
		if !sendSystemKey(conn, &entry) {
			return
		}
		numProduced++
	}
}
