//  Copyright (c) 2016 Couchbase, Inc.
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

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type userInfoKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *userInfoKeyspace) Release(close bool) {
}

func (b *userInfoKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *userInfoKeyspace) Id() string {
	return b.Name()
}

func (b *userInfoKeyspace) Name() string {
	return b.name
}

func getUserInfoList(s *store) ([]interface{}, errors.Error) {
	val, err := s.UserInfo()
	if err != nil {
		return nil, err
	}
	// Expected data format:
	//   [{"id":"ivanivanov","name":"Ivan Ivanov","roles":[{"role":"cluster_admin"},{"bucket_name":"default","role":"bucket_admin"}]},
	//    {"id":"petrpetrov","name":"Petr Petrov","roles":[{"role":"replication_admin"}]}]
	data := val.Actual()
	sliceOfUsers, ok := data.([]interface{})
	if !ok {
		return nil, errors.NewInvalidValueError(fmt.Sprintf("Unexpected format for user_info received from server: %v", data))
	}

	return sliceOfUsers, nil
}

func (b *userInfoKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	uil, err := getUserInfoList(b.namespace.store)
	if err != nil {
		return 0, err
	}
	return int64(len(uil)), nil
}

func (b *userInfoKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *userInfoKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *userInfoKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *userInfoKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {
	sliceOfUsers, err := getUserInfoList(b.namespace.store)
	if err != nil {
		return []errors.Error{err}
	}
	newMap, err := userInfoListToMap(sliceOfUsers)
	if err != nil {
		return []errors.Error{err}
	}

	for _, k := range keys {
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

func (b *userInfoKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *userInfoKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *userInfoKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *userInfoKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func newUserInfoKeyspace(p *namespace) (*userInfoKeyspace, errors.Error) {
	b := new(userInfoKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_USER_INFO)

	primary := &userInfoIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

type userInfoIndex struct {
	indexBase
	name     string
	keyspace *userInfoKeyspace
}

func (pi *userInfoIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *userInfoIndex) Id() string {
	return pi.Name()
}

func (pi *userInfoIndex) Name() string {
	return pi.name
}

func (pi *userInfoIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *userInfoIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *userInfoIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *userInfoIndex) Condition() expression.Expression {
	return nil
}

func (pi *userInfoIndex) IsPrimary() bool {
	return true
}

func (pi *userInfoIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *userInfoIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *userInfoIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *userInfoIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

func userInfoListToMap(sliceOfUsers []interface{}) (map[string]value.Value, errors.Error) {
	newMap := make(map[string]value.Value, len(sliceOfUsers))
	for i, u := range sliceOfUsers {
		userAsMap, ok := u.(map[string]interface{})
		if !ok {
			return nil, errors.NewInvalidValueError(fmt.Sprintf("Unexpected format for user_info at position %d: %v", i, u))
		}
		auth.ConvertRolesToAliases(userAsMap)
		id, present := userAsMap["id"]
		if !present {
			return nil, errors.NewInvalidValueError(fmt.Sprintf("Could not find id in user_info data at position %d: %v", i, u))
		}
		idAsString, ok := id.(string)
		if !ok {
			return nil, errors.NewInvalidValueError(fmt.Sprintf("Field id of unexpected type in user_info data at position %d: %v", i, u))
		}
		domain, present := userAsMap["domain"]
		if !present {
			return nil, errors.NewInvalidValueError(fmt.Sprintf("Could not find domain in user_info data at position %d: %v", i, u))
		}
		domainAsString, ok := domain.(string)
		if !ok {
			return nil, errors.NewInvalidValueError(
				fmt.Sprintf("Field domain of unexpected type in user_info data at position %d: %v", i, u))
		}
		userKey := fmt.Sprintf("%s:%s", domainAsString, idAsString)
		newMap[userKey] = value.NewValue(u)
	}
	return newMap, nil
}

func (pi *userInfoIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
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
