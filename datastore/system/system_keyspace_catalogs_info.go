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
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type catalogsInfoKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *catalogsInfoKeyspace) Release(close bool) {
}

func (b *catalogsInfoKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *catalogsInfoKeyspace) Id() string {
	return b.Name()
}

func (b *catalogsInfoKeyspace) Name() string {
	return b.name
}

// canConsumeCredential checks if user can consume from a credential store
func canConsumeCredential(context datastore.QueryContext, credentialId string) bool {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_CLUSTER_CREDENTIALSTORE_CONSUME, auth.PRIV_PROPS_NONE)

	err := datastore.GetDatastore().AuthorizeInternal(privs, context.Credentials())
	return err == nil
}

// canAccessCatalogInfo checks if user can access catalog info
// Returns true if user has catalog read privilege
// For catalogs with credentialId, also requires credential consume privilege
func canAccessCatalogInfo(context datastore.QueryContext, catalogName string, credentialId string) bool {
	// Must have catalog read privilege first
	privs := auth.NewPrivileges()
	privs.Add(catalogName, auth.PRIV_CATALOG_SELECT, auth.PRIV_PROPS_NONE)
	if credentialId != "" {
		// If catalog has a credential, require consume privilege as well
		privs.Add(credentialId, auth.PRIV_CLUSTER_CREDENTIALSTORE_CONSUME, auth.PRIV_PROPS_NONE)
	}

	err := datastore.GetDatastore().AuthorizeInternal(privs, context.Credentials())
	return err == nil
}

func (b *catalogsInfoKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	sliceOfCatalogs, err := getCatalogList(context, b.namespace.store, "", 0)
	if err != nil {
		return 0, err
	}

	count := int64(0)
	mapOfCatalogs := catalogListToMap(sliceOfCatalogs)

	for catalogName, catalogVal := range mapOfCatalogs {
		credentialId := extractCredentialId(catalogVal)
		if !canAccessCatalogInfo(context, catalogName, credentialId) {
			context.Warning(errors.NewSystemFilteredRowsWarning("system:catalogs_info"))
		} else {
			count++
		}
	}

	return count, nil
}

func (b *catalogsInfoKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *catalogsInfoKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *catalogsInfoKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *catalogsInfoKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {

	// Files carries per-file record counts (see LoadCatalogMetadata), which requires a
	// live PlanFiles metadata call per table, on top of schema/snapshot info.
	sliceOfCatalogs, err := getCatalogList(context, b.namespace.store, "",
		datastore.CatalogInfoSchema|datastore.CatalogInfoSnapshots|datastore.CatalogInfoFiles)
	if err != nil {
		return []errors.Error{err}
	}

	newMap := catalogListToMap(sliceOfCatalogs)

	for _, k := range keys {
		val := newMap[k]
		if val == nil {
			continue
		}

		credentialId := extractCredentialId(val)
		if !canAccessCatalogInfo(context, k, credentialId) {
			context.Warning(errors.NewSystemFilteredRowsWarning("system:catalogs_info"))
			continue
		}

		item := value.NewAnnotatedValue(val)
		item.SetMetaField(value.META_KEYSPACE, b.fullName)
		item.SetId(k)
		keysMap[k] = item
	}

	return
}

func extractCredentialId(catalogVal value.Value) string {
	if catalogVal == nil {
		return ""
	}
	data := catalogVal.Actual()
	catalogMap, ok := data.(map[string]any)
	if !ok {
		return ""
	}
	credId, ok := catalogMap["credentialId"]
	if !ok {
		return ""
	}
	credIdStr, ok := credId.(string)
	if !ok {
		return ""
	}
	return credIdStr
}

func newCatalogsInfoKeyspace(p *namespace) (*catalogsInfoKeyspace, errors.Error) {
	b := new(catalogsInfoKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_CATALOGS_INFO)

	primary := &catalogsInfoIndex{name: PRIMARY_INDEX_NAME, keyspace: b, primary: true}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	// add a secondary index on `name`
	expr, err := parser.Parse("`name`")

	if err == nil {
		key := expression.Expressions{expr}
		nameIdx := &catalogsInfoIndex{
			name:     "#names",
			keyspace: b,
			primary:  false,
			idxKey:   key,
		}
		setIndexBase(&nameIdx.indexBase, b.indexer)
		b.indexer.(*systemIndexer).AddIndex(nameIdx.name, nameIdx)
	} else {
		return nil, errors.NewSystemDatastoreError(err, "")
	}

	return b, nil
}

type catalogsInfoIndex struct {
	indexBase
	name     string
	keyspace *catalogsInfoKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *catalogsInfoIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *catalogsInfoIndex) Id() string {
	return pi.Name()
}

func (pi *catalogsInfoIndex) Name() string {
	return pi.name
}

func (pi *catalogsInfoIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *catalogsInfoIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *catalogsInfoIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *catalogsInfoIndex) Condition() expression.Expression {
	return nil
}

func (pi *catalogsInfoIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *catalogsInfoIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *catalogsInfoIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *catalogsInfoIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *catalogsInfoIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	if span == nil || pi.primary {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		defer conn.Sender().Close()

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}

		sliceOfCatalogs, err := getCatalogList(conn.QueryContext(), pi.keyspace.namespace.store, "", 0)
		if err != nil {
			conn.Fatal(err)
			return
		}
		mapOfCatalogs := catalogListToMap(sliceOfCatalogs)

		var numProduced int64
		for k, catalogVal := range mapOfCatalogs {
			if !spanEvaluator.evaluate(k) {
				continue
			}
			if limit > 0 && numProduced >= limit {
				break
			}

			credentialId := extractCredentialId(catalogVal)
			if !canAccessCatalogInfo(conn.QueryContext(), k, credentialId) {
				conn.QueryContext().Warning(errors.NewSystemFilteredRowsWarning("system:catalogs_info"))
				continue
			}

			entry := datastore.IndexEntry{PrimaryKey: k}
			if !sendSystemKey(conn, &entry) {
				return
			}
			numProduced++
		}
	}
}

func (pi *catalogsInfoIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	sliceOfCatalogs, err := getCatalogList(conn.QueryContext(), pi.keyspace.namespace.store, "", 0)
	if err != nil {
		conn.Fatal(err)
		return
	}
	mapOfCatalogs := catalogListToMap(sliceOfCatalogs)

	var numProduced int64
	for k, catalogVal := range mapOfCatalogs {
		if limit > 0 && numProduced >= limit {
			break
		}

		credentialId := extractCredentialId(catalogVal)
		if !canAccessCatalogInfo(conn.QueryContext(), k, credentialId) {
			conn.QueryContext().Warning(errors.NewSystemFilteredRowsWarning("system:catalogs_info"))
			continue
		}

		entry := datastore.IndexEntry{PrimaryKey: k}
		if !sendSystemKey(conn, &entry) {
			return
		}
		numProduced++
	}
}
