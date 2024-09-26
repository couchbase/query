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

type bucketInfoKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *bucketInfoKeyspace) Release(close bool) {
}

func (b *bucketInfoKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *bucketInfoKeyspace) Id() string {
	return b.Name()
}

func (b *bucketInfoKeyspace) Name() string {
	return b.name
}

func getBucketInfoList(s *store) ([]interface{}, errors.Error) {
	val, err := s.BucketInfo()
	if err != nil {
		return nil, err
	}
	// Expected data format:
	//   [{"name":"b1",...},
	//    {"name":"b2",...}]
	data := val.Actual()
	sliceOfBuckets, ok := data.([]interface{})
	if !ok {
		return nil, errors.NewInvalidValueError(fmt.Sprintf("Unexpected format for bucket_info received from server: %v", data))
	}

	return sliceOfBuckets, nil
}

func (b *bucketInfoKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	uil, err := getBucketInfoList(b.namespace.store)
	if err != nil {
		return 0, err
	}
	return int64(len(uil)), nil
}

func (b *bucketInfoKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *bucketInfoKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *bucketInfoKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *bucketInfoKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {
	sliceOfBuckets, err := getBucketInfoList(b.namespace.store)
	if err != nil {
		return []errors.Error{err}
	}
	newMap, err := bucketInfoListToMap(sliceOfBuckets)
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

func newBucketInfoKeyspace(p *namespace) (*bucketInfoKeyspace, errors.Error) {
	b := new(bucketInfoKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_BUCKET_INFO, KEYSPACE_NAME_DATABASE_INFO)

	primary := &bucketInfoIndex{name: PRIMARY_INDEX_NAME, keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

type bucketInfoIndex struct {
	indexBase
	name     string
	keyspace *bucketInfoKeyspace
}

func (pi *bucketInfoIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *bucketInfoIndex) Id() string {
	return pi.Name()
}

func (pi *bucketInfoIndex) Name() string {
	return pi.name
}

func (pi *bucketInfoIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *bucketInfoIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *bucketInfoIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *bucketInfoIndex) Condition() expression.Expression {
	return nil
}

func (pi *bucketInfoIndex) IsPrimary() bool {
	return true
}

func (pi *bucketInfoIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *bucketInfoIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *bucketInfoIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *bucketInfoIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

func bucketInfoListToMap(sliceOfBuckets []interface{}) (map[string]value.Value, errors.Error) {
	newMap := make(map[string]value.Value, len(sliceOfBuckets))
	for i, b := range sliceOfBuckets {
		bucketAsMap, ok := b.(map[string]interface{})
		if !ok {
			return nil, errors.NewInvalidValueError(fmt.Sprintf("Unexpected format for bucket_info at position %d: %v", i, b))
		}
		auth.ConvertRolesToAliases(bucketAsMap)
		name, present := bucketAsMap["name"]
		if !present {
			return nil, errors.NewInvalidValueError(fmt.Sprintf("Could not find id in bucket_info data at position %d: %v", i, b))
		}
		namesString, ok := name.(string)
		if !ok {
			return nil, errors.NewInvalidValueError(
				fmt.Sprintf("Field id of unexpected type in bucket_info data at position %d: %v", i, b))
		}
		// replace unwieldy array with a string representation of the array
		vBucketServerMapField, present := bucketAsMap["vBucketServerMap"]
		if !present {
			return nil, errors.NewInvalidValueError(
				fmt.Sprintf("Could not find vBucketServerMap in bucket_info data at position %d: %v", i, b))
		}
		vBucketServerMap, ok := vBucketServerMapField.(map[string]interface{})
		if !ok {
			return nil, errors.NewInvalidValueError(
				fmt.Sprintf("Field vBucketServerMap of unexpected type in bucket_info data at position %d: %v", i, b))
		}
		vBucketMapField, present := vBucketServerMap["vBucketMap"]
		if !present {
			return nil, errors.NewInvalidValueError(
				fmt.Sprintf("Could not find vBucketServerMap.vBucketMap in bucket_info data at position %d: %v", i, b))
		}
		vBucketMap, ok := vBucketMapField.([]interface{})
		if !ok {
			return nil, errors.NewInvalidValueError(
				fmt.Sprintf("Field vBucketServerMap.vBucketMap of unexpected type in bucket_info data at position %d: %v", i, b))
		}
		s := ""
		for i := range vBucketMap {
			m := fmt.Sprintf("%v", vBucketMap[i])
			s += "|" + m[1:len(m)-1]
		}
		if len(vBucketMap) == 0 {
			vBucketServerMap["vBucketMap"] = ""
		} else {
			vBucketServerMap["vBucketMap"] = s[1:]
		}

		newMap[namesString] = value.NewValue(b)
	}
	return newMap, nil
}

func (pi *bucketInfoIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	sliceOfBuckets, err := getBucketInfoList(pi.keyspace.namespace.store)
	if err != nil {
		conn.Fatal(err)
		return
	}
	mapOfBuckets, err := bucketInfoListToMap(sliceOfBuckets)
	if err != nil {
		conn.Fatal(err)
		return
	}

	var numProduced int64
	for k, _ := range mapOfBuckets {
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
