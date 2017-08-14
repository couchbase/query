//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitCreatePrimaryIndex(stmt *algebra.CreatePrimaryIndex) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	return plan.NewCreatePrimaryIndex(keyspace, stmt), nil
}

func (this *builder) VisitCreateIndex(stmt *algebra.CreateIndex) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	indexer, er := keyspace.Indexer(stmt.Using())
	if er != nil {
		return nil, er
	}

	er = indexer.Refresh()
	if er != nil {
		return nil, er
	}

	if stmt.Keys().HasDescending() {
		if _, ok := indexer.(datastore.Indexer2); !ok {
			return nil, errors.NewIndexerDescCollationError()
		}
	}

	// Check that the index does not already exist.
	index, _ := indexer.IndexByName(stmt.Name())
	if index != nil {
		return nil, errors.NewIndexAlreadyExistsError(stmt.Name())
	}

	return plan.NewCreateIndex(keyspace, stmt), nil
}

func (this *builder) VisitDropIndex(stmt *algebra.DropIndex) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	indexer, er := keyspace.Indexer(stmt.Using())
	if er != nil {
		return nil, er
	}

	er = indexer.Refresh()
	if er != nil {
		return nil, er
	}

	index, er := indexer.IndexByName(stmt.Name())
	if er != nil {
		return nil, er
	}

	return plan.NewDropIndex(index, stmt), nil
}

func (this *builder) VisitAlterIndex(stmt *algebra.AlterIndex) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	indexer, er := keyspace.Indexer(stmt.Using())
	if er != nil {
		return nil, er
	}

	er = indexer.Refresh()
	if er != nil {
		return nil, er
	}

	index, er := indexer.IndexByName(stmt.Name())
	if er == nil {
		return nil, er
	}

	if _, ok := index.(datastore.AlterIndex); !ok {
		return nil, errors.NewAlterIndexError()
	}

	return plan.NewAlterIndex(index, stmt, keyspace), nil
}

func (this *builder) VisitBuildIndexes(stmt *algebra.BuildIndexes) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	indexer, er := keyspace.Indexer(stmt.Using())
	if er != nil {
		return nil, er
	}

	er = indexer.Refresh()
	if er != nil {
		return nil, er
	}

	for _, name := range stmt.Names() {
		_, er := indexer.IndexByName(name)
		if er != nil {
			return nil, er
		}
	}

	return plan.NewBuildIndexes(keyspace, stmt), nil
}

func (this *builder) getNameKeyspace(ns, ks string) (datastore.Keyspace, error) {
	if ns == "" {
		ns = this.namespace
	}

	//if strings.ToLower(ns) == "#system" {
	//return nil, fmt.Errorf("Index operations not allowed on system namespace.")
	//}

	datastore := this.datastore
	if strings.ToLower(ns) == "#system" {
		datastore = this.systemstore
	}
	namespace, err := datastore.NamespaceByName(ns)
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(ks)
	if err != nil {
		return nil, err
	}

	return keyspace, nil
}
