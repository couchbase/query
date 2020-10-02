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
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type applicableRolesKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *applicableRolesKeyspace) Release(close bool) {
}

func (b *applicableRolesKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *applicableRolesKeyspace) Id() string {
	return b.Name()
}

func (b *applicableRolesKeyspace) Name() string {
	return b.name
}

func (b *applicableRolesKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	users, err := datastore.GetDatastore().GetUserInfoAll()
	if err != nil {
		return 0, errors.NewSystemUnableToRetrieveError(err)
	}

	numFound := 0
	for _, user := range users {
		numFound += len(user.Roles)
	}
	return int64(numFound), nil
}

func (b *applicableRolesKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *applicableRolesKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *applicableRolesKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *applicableRolesKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {

	for _, key := range keys {
		err, grantee, role, target := splitAppRolesKey(key)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		valMap := make(map[string]interface{}, 3)
		valMap["grantee"] = grantee
		valMap["role"] = auth.RoleToAlias(role)
		if target != "" {
			object := strings.SplitN(target, ".", 3)
			valMap["bucket_name"] = object[0]
			if len(object) > 1 {
				valMap["scope_name"] = object[1]
			}
			if len(object) > 2 {
				valMap["collection_name"] = object[2]
			}
		}
		val := value.NewValue(valMap)
		item := value.NewAnnotatedValue(val)
		item.NewMeta()["keyspace"] = b.fullName
		item.SetId(key)

		keysMap[key] = item
	}
	return
}

func (b *applicableRolesKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *applicableRolesKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *applicableRolesKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *applicableRolesKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func newApplicableRolesKeyspace(p *namespace) (*applicableRolesKeyspace, errors.Error) {
	b := new(applicableRolesKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_APPLICABLE_ROLES)

	primary := &applicableRolesIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

type applicableRolesIndex struct {
	indexBase
	name     string
	keyspace *applicableRolesKeyspace
}

func (pi *applicableRolesIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *applicableRolesIndex) Id() string {
	return pi.Name()
}

func (pi *applicableRolesIndex) Name() string {
	return pi.name
}

func (pi *applicableRolesIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *applicableRolesIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *applicableRolesIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *applicableRolesIndex) Condition() expression.Expression {
	return nil
}

func (pi *applicableRolesIndex) IsPrimary() bool {
	return true
}

func (pi *applicableRolesIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *applicableRolesIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *applicableRolesIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *applicableRolesIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	if span == nil {
		pi.scanEntries(limit, conn, nil)
	} else {
		compSpan, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}
		pi.scanEntries(limit, conn, compSpan)
	}
}

func (pi *applicableRolesIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	pi.scanEntries(limit, conn, nil)
}

// sample key: "ivanivanov/bucket_admin/testbucket"
func makeAppRolesKey(id, roleName, target string) string {
	return fmt.Sprintf("%s/%s/%s", id, roleName, target)
}

func splitAppRolesKey(key string) (err errors.Error, id, roleName, target string) {
	fields := strings.Split(key, "/")
	if len(fields) != 3 {
		err = errors.NewSystemMalformedKeyError(key, "system:applicable_roles")
		return
	}
	id = fields[0]
	roleName = fields[1]
	target = fields[2]
	return
}

func (pi *applicableRolesIndex) scanEntries(limit int64, conn *datastore.IndexConnection, compSpan *compiledSpan) {
	users, err := datastore.GetDatastore().GetUserInfoAll()
	if err != nil {
		conn.Error(errors.NewSystemUnableToRetrieveError(err))
		return
	}

	numProduced := int64(0)
	for _, user := range users {
		for _, role := range user.Roles {
			if numProduced >= limit {
				return
			}
			key := makeAppRolesKey(user.Id, role.Name, role.Target)
			if compSpan == nil || compSpan.evaluate(key) {
				entry := datastore.IndexEntry{PrimaryKey: key}
				if !sendSystemKey(conn, &entry) {
					return
				}
				numProduced++
			}
		}
	}
}
