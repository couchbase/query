//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"fmt"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type groupInfoKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *groupInfoKeyspace) Release(close bool) {
}

func (b *groupInfoKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *groupInfoKeyspace) Id() string {
	return b.Name()
}

func (b *groupInfoKeyspace) Name() string {
	return b.name
}

func getGroupInfoList(s *store) ([]interface{}, errors.Error) {
	val, err := s.GroupInfo()
	if err != nil {
		return nil, err
	}
	// Expected data format:
	//   [{"id":"g1","description":"one","roles":[{"Name":"query_select","Target":"default:_default:_default"}]
	//    {"id":"g2","description":"two","roles":[{"Name":"replication_admin", "Target":""}]}]
	data := val.Actual()
	sliceOfGroups, ok := data.([]interface{})
	if !ok {
		return nil, errors.NewInvalidValueError(fmt.Sprintf("Unexpected format for group_info received from server: %v", data))
	}

	return sliceOfGroups, nil
}

func (b *groupInfoKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	uil, err := getGroupInfoList(b.namespace.store)
	if err != nil {
		return 0, err
	}
	return int64(len(uil)), nil
}

func (b *groupInfoKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *groupInfoKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *groupInfoKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *groupInfoKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {
	sliceOfGroups, err := getGroupInfoList(b.namespace.store)
	if err != nil {
		return []errors.Error{err}
	}
	newMap, err := groupInfoListToMap(sliceOfGroups)
	if err != nil {
		return []errors.Error{err}
	}

	for _, k := range keys {
		val := newMap[k]
		if val == nil {
			continue
		}

		item := value.NewAnnotatedValue(val)
		item.SetMetaField(value.META_KEYSPACE, b.fullName)
		item.SetId(k)
		keysMap[k] = item
	}

	return
}

func newGroupInfoKeyspace(p *namespace) (*groupInfoKeyspace, errors.Error) {
	b := new(groupInfoKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_GROUP_INFO)

	primary := &groupInfoIndex{name: PRIMARY_INDEX_NAME, keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

type groupInfoIndex struct {
	indexBase
	name     string
	keyspace *groupInfoKeyspace
}

func (pi *groupInfoIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *groupInfoIndex) Id() string {
	return pi.Name()
}

func (pi *groupInfoIndex) Name() string {
	return pi.name
}

func (pi *groupInfoIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *groupInfoIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *groupInfoIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *groupInfoIndex) Condition() expression.Expression {
	return nil
}

func (pi *groupInfoIndex) IsPrimary() bool {
	return true
}

func (pi *groupInfoIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *groupInfoIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *groupInfoIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *groupInfoIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

func groupInfoListToMap(sliceOfGroups []interface{}) (map[string]value.Value, errors.Error) {
	newMap := make(map[string]value.Value, len(sliceOfGroups))
	for i, g := range sliceOfGroups {
		groupAsMap, ok := g.(map[string]interface{})
		if !ok {
			return nil, errors.NewInvalidValueError(fmt.Sprintf("Unexpected format for group_info at position %d: %v", i, g))
		}
		auth.ConvertRolesToAliases(groupAsMap)
		id, present := groupAsMap["id"]
		if !present {
			return nil, errors.NewInvalidValueError(fmt.Sprintf("Could not find id in group_info data at position %d: %v", i, g))
		}
		idAsString, ok := id.(string)
		if !ok {
			return nil, errors.NewInvalidValueError(
				fmt.Sprintf("Field id of unexpected type in group_info data at position %d: %v", i, g))
		}
		newMap[idAsString] = value.NewValue(g)
	}
	return newMap, nil
}

func (pi *groupInfoIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	sliceOfGroups, err := getGroupInfoList(pi.keyspace.namespace.store)
	if err != nil {
		conn.Fatal(err)
		return
	}
	mapOfGroups, err := groupInfoListToMap(sliceOfGroups)
	if err != nil {
		conn.Fatal(err)
		return
	}

	var numProduced int64
	for k, _ := range mapOfGroups {
		if limit > 0 && numProduced > limit {
			break
		}

		entry := datastore.IndexEntry{PrimaryKey: k}
		if !sendSystemKey(conn, &entry) {
			return
		}
		numProduced++
	}
}
