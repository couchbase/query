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
const ClusterManagerAddr = "localhost:9997"

// IndexerAddr is temporary hard-coded address for indexer node.
const IndexerAddr = "localhost:9998"

// load 2i indexes and remember them as part of keyspace.indexes.
func (b *keyspace) load2iIndexes() errors.Error {
	indexes, err := getCoordinatorIndexes(b)
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
func getCoordinatorIndexes(b *keyspace) (map[string]datastore.Index, error) {
	indexes := make(map[string]datastore.Index)
	client := queryport.NewClusterClient(ClusterManagerAddr)
	infos, err := client.List()
	if err != nil {
		return indexes, err
	}
	for _, info := range infos {
		rkeys := make(expression.Expressions, 0)
		for _, secExpr := range info.SecExprs {
			expr, err := parser.Parse(secExpr)
			if err != nil {
				return nil, err
			}
			rkeys = append(rkeys, expr)
		}
		expr, err := parser.Parse(info.PartnExpr)
		if err != nil {
			return nil, err
		}
		ekey := expression.Expressions{expr}
		wkey, err := parser.Parse(info.WhereExpr)
		if err != nil {
			return nil, err
		}
		using := datastore.IndexType(info.Using)
		index, err := new2iIndex(info.Name, ekey, rkeys, wkey, using, b)
		if err != nil {
			return nil, err
		}
		indexes[index.Name()] = index
	}
	return indexes, nil
}

// create a new 2i index.
func new2iPrimaryIndex(
	b *keyspace, using datastore.IndexType) (*secondaryIndex, errors.Error) {

	bucket := b.Name()
	client := queryport.NewClusterClient(ClusterManagerAddr)
	info, err := client.CreateIndex(
		PRIMARY_INDEX, bucket, string(using), "", "", nil, true)
	if err != nil {
		return nil, errors.NewError(nil, err.Error())
	}
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

	bucket := b.Name()
	client := queryport.NewClusterClient(ClusterManagerAddr)
	info, err := client.CreateIndex(
		name, bucket, string(using), partnStr, whereStr, secStrs, false)
	if err != nil {
		return nil, errors.NewError(nil, err.Error())
	}
	index := &secondaryIndex{
		name:      name,
		defnID:    info.DefnID,
		keySpace:  b,
		isPrimary: false,
		using:     using,
		partnExpr: partnStr,
		secExprs:  secStrs,
		whereExpr: whereStr,
		// remote node hosting this index.
		hosts: nil, // to becomputed by coordinator
	}
	// TODO: fetch the new index topology from coordinator.
	index.setHost([]string{IndexerAddr})
	return index, nil
}
