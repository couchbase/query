//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package couchbase

import "github.com/couchbaselabs/query/datastore"
import "github.com/couchbaselabs/query/errors"
import "github.com/couchbaselabs/query/expression"
import "github.com/couchbaselabs/query/logging"
import "github.com/couchbaselabs/go-couchbase"

// load 2i indexes and remember them as part of keyspace.indexes.
func (b *keyspace) load2iIndexes() errors.Error {
	indexes, err := getCoordinatorIndexes(b.namespace.site.client)
	if err != nil {
		return errors.NewError(err, "Error loading indexes")
	}
	for _, index := range indexes {
		logging.Infof("found index on keyspace %s", index.Name())
		name := index.Name()
		b.indexes[name] = index
	}
	return nil
}

// get the list of indexes from coordinator.
func getCoordinatorIndexes(
	client couchbase.Client) (map[string]datastore.Index, error) {

	// TODO: actually fetch the list from coordinator.
	return nil, nil
}

// create a new 2i index.
func new2iPrimaryIndex(b *keyspace) (*secondaryIndex, errors.Error) {
	index := &secondaryIndex{
		name:      PRIMARY_INDEX,
		defnID:    "", // computed by coordinator
		keySpace:  b,
		isPrimary: true,
		using:     datastore.LSM,
		// remote node hosting this index.
		hosts: nil, // to becomputed by coordinator
	}
	// TODO: publish this to coordinator.
	// TODO: fetch the new index topology from coordinator.
	//       until then localhost:9998 will be used as indexer.
	index.setHost([]string{"localhost:9998"})
	return index, nil
}

// create a new 2i index.
func new2iIndex(
	name string,
	equalKey, rangeKey expression.Expressions, where expression.Expression,
	using datastore.IndexType,
	b *keyspace) (*secondaryIndex, errors.Error) {

	var partnStr string
	if equalKey != nil && len(equalKey) > 0 {
		partnStr = expression.NewStringer().Visit(equalKey[0])
	}

	var whereStr string
	if where != nil {
		whereStr = expression.NewStringer().Visit(where)
	}

	secStrs := make([]string, len(rangeKey))
	for i, key := range rangeKey {
		s := expression.NewStringer().Visit(key)
		secStrs[i] = s
	}

	index := &secondaryIndex{
		name:      name,
		defnID:    "", // computed by coordinator
		keySpace:  b,
		isPrimary: false,
		using:     using,
		partnExpr: partnStr,
		secExprs:  secStrs,
		whereExpr: whereStr,
		// remote node hosting this index.
		hosts: nil, // to becomputed by coordinator
	}
	// TODO: publish this to coordinator.
	// TODO: fetch the new index topology from coordinator.
	//       until then localhost:9998 will be used as indexer.
	index.setHost([]string{"localhost:9998"})
	return index, nil
}
