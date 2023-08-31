//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type myUserInfoKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *myUserInfoKeyspace) Release(close bool) {
}

func (b *myUserInfoKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *myUserInfoKeyspace) Id() string {
	return b.Name()
}

func (b *myUserInfoKeyspace) Name() string {
	return b.name
}

func (b *myUserInfoKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var authUsers []string
	creds := context.Credentials()
	if creds != nil {
		authUsers = []string(creds.AuthenticatedUsers)
	}
	approverFunc := func(id string) bool {
		for _, v := range authUsers {
			if id == v {
				return true
			}
		}
		return false
	}

	sliceOfUsers, err := getUserInfoList(b.namespace.store)
	if err != nil {
		return 0, err
	}
	userMap, err := userInfoListToMap(sliceOfUsers)
	if err != nil {
		return 0, err
	}

	var total int64
	for k := range userMap {
		if approverFunc(k) {
			total++
		}
	}

	return total, nil
}

func (b *myUserInfoKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *myUserInfoKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *myUserInfoKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *myUserInfoKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string) (errs errors.Errors) {
	authUsers := context.Credentials().AuthenticatedUsers
	approverFunc := func(id string) bool {
		for _, v := range authUsers {
			if id == v {
				return true
			}
		}
		return false
	}

	sliceOfUsers, err := getUserInfoList(b.namespace.store)
	if err != nil {
		return []errors.Error{err}
	}
	newMap, err := userInfoListToMap(sliceOfUsers)
	if err != nil {
		return []errors.Error{err}
	}

	for _, k := range keys {
		if !approverFunc(k) {
			continue
		}
		val := newMap[k]
		if val == nil {
			continue
		}

		item := value.NewAnnotatedValue(val)
		item.NewMeta()["keyspace"] = b.fullName
		item.SetId(k)

		keysMap[k] = item
	}

	return
}

func newMyUserInfoKeyspace(p *namespace) (*myUserInfoKeyspace, errors.Error) {
	b := new(myUserInfoKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_MY_USER_INFO)

	primary := &myUserInfoIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

type myUserInfoIndex struct {
	indexBase
	name     string
	keyspace *myUserInfoKeyspace
}

func (pi *myUserInfoIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *myUserInfoIndex) Id() string {
	return pi.Name()
}

func (pi *myUserInfoIndex) Name() string {
	return pi.name
}

func (pi *myUserInfoIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *myUserInfoIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *myUserInfoIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *myUserInfoIndex) Condition() expression.Expression {
	return nil
}

func (pi *myUserInfoIndex) IsPrimary() bool {
	return true
}

func (pi *myUserInfoIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *myUserInfoIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *myUserInfoIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *myUserInfoIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

func (pi *myUserInfoIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()
	sliceOfUsers, err := getUserInfoList(pi.keyspace.namespace.store)
	if err != nil {
		conn.Fatal(err)
		return
	}
	mapOfUsers, err := userInfoListToMap(sliceOfUsers)
	if err != nil {
		conn.Fatal(err)
		return
	}

	var numProduced int64
	for k, _ := range mapOfUsers {
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
