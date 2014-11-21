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
import "github.com/couchbaselabs/query/expression/parser"
import "github.com/couchbaselabs/query/logging"
import "github.com/couchbase/indexing/secondary/queryport"

// ClusterManagerAddr is temporary hard-coded address for cluster-manager-agent
const ClusterManagerAddr = "localhost:9101"

// IndexerAddr is temporary hard-coded address for indexer node.
const IndexerAddr = "localhost:7000"

// load 2i indexes and remember them as part of keyspace.indexes.
func (b *keyspace) load2iIndexes() ([]datastore.Index, errors.Error) {
	indexes, err := getCoordinatorIndexes(b)
	if err != nil {
		return nil, errors.NewError(err, " Error loading 2i indexes")
	}
	return indexes, nil
}

// get the list of indexes from coordinator.
func getCoordinatorIndexes(b *keyspace) ([]datastore.Index, error) {
	indexes := make([]datastore.Index, 0)
	client := queryport.NewClusterClient(ClusterManagerAddr)
	infos, err := client.List()
	if err != nil {
		return nil, err
	} else if infos == nil { // empty list of indexes
		return nil, nil
	}

	var index *secondaryIndex

	for _, info := range infos {
		using := datastore.IndexType(info.Using)
		if info.Name == PRIMARY_INDEX {
			index, err = new2iPrimaryIndex(b, using, &info)
			if err != nil {
				return nil, err
			}

		} else {
			index, err = new2iIndex(b, &info)
			if err != nil {
				return nil, err
			}
		}
		indexes = append(indexes, index)
		logging.Infof(" found index on keyspace %s", index.Name())
	}
	return indexes, nil
}

// create 2i primary index
func create2iPrimaryIndex(
	b *keyspace, using datastore.IndexType) (*secondaryIndex, errors.Error) {

	bucket := b.Name()
	client := queryport.NewClusterClient(ClusterManagerAddr)
	info, err := client.CreateIndex(
		PRIMARY_INDEX, bucket, string(using), "N1QL", "", "", nil, true)
	if err != nil {
		return nil, errors.NewError(err, " Primary CreateIndex() with 2i failed")
	} else if info == nil {
		return nil, errors.NewError(nil, " primary CreateIndex() with 2i failed")
	}
	return new2iPrimaryIndex(b, using, info)
}

// new 2i index.
func new2iPrimaryIndex(
	b *keyspace, using datastore.IndexType,
	info *queryport.IndexInfo) (*secondaryIndex, errors.Error) {

	index := &secondaryIndex{
		name:      PRIMARY_INDEX,
		defnID:    info.DefnID,
		keySpace:  b,
		isPrimary: true,
		using:     datastore.LSM,
		// remote node hosting this index.
		hosts: nil, // to becomputed by coordinator
	}
	// TODO: fetch the new index topology from coordinator.
	index.setHost([]string{IndexerAddr})
	return index, nil
}

// create 2i index
func create2iIndex(
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

	bucket := b.Name()
	client := queryport.NewClusterClient(ClusterManagerAddr)
	info, err := client.CreateIndex(
		name, bucket, string(using), "N1QL", partnStr, whereStr, secStrs, false)
	if err != nil {
		return nil, errors.NewError(nil, err.Error())
	} else if info == nil {
		return nil, errors.NewError(nil, "2i CreateIndex() failed")
	}
	return new2iIndex(b, info)
}

// new 2i index.
func new2iIndex(
	b *keyspace, info *queryport.IndexInfo) (*secondaryIndex, errors.Error) {

	index := &secondaryIndex{
		name:      info.Name,
		defnID:    info.DefnID,
		keySpace:  b,
		isPrimary: info.IsPrimary,
		using:     datastore.IndexType(info.Using),
		partnExpr: info.PartnExpr,
		secExprs:  info.SecExprs,
		whereExpr: info.WhereExpr,
		// remote node hosting this index.
		hosts: nil, // to becomputed by coordinator
	}
	// TODO: fetch the new index topology from coordinator.
	index.setHost([]string{IndexerAddr})
	return index, nil
}

func parseExprs(exprs []string) (expression.Expressions, error) {
	keys := expression.Expressions(nil)
	if len(exprs) > 0 {
		for _, expr := range exprs {
			if len(expr) > 0 {
				key, err := parser.Parse(expr)
				if err != nil {
					return nil, err
				}
				keys = append(keys, key)
			}
		}
	}
	return keys, nil
}
