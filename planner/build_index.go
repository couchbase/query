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
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"strings"
)

func (this *builder) VisitCreatePrimaryIndex(stmt *algebra.CreatePrimaryIndex) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref, false)
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

	if stmt.Partition() != nil {
		if _, ok := indexer.(datastore.Indexer3); !ok {
			return nil, errors.NewPartitionIndexNotSupportedError()
		}
	}

	return plan.NewCreatePrimaryIndex(keyspace, stmt), nil
}

func (this *builder) VisitCreateIndex(stmt *algebra.CreateIndex) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref, false)
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

	if stmt.Partition() != nil {
		if _, ok := indexer.(datastore.Indexer3); !ok {
			return nil, errors.NewPartitionIndexNotSupportedError()
		}
	}

	// Check that the index does not already exist.
	index, _ := indexer.IndexByName(stmt.Name())
	if index != nil {
		return nil, errors.NewIndexAlreadyExistsError(stmt.Name())
	}

	// Make sure you dont have multiple xattrs
	_, names := expression.XattrsNames(stmt.Expressions(), "")
	if ok := isValidXattrs(names); !ok {
		return nil, errors.NewPlanInternalError("Only a single user or system xattribute can be indexed.")
	}

	return plan.NewCreateIndex(keyspace, stmt), nil
}

func (this *builder) VisitDropIndex(stmt *algebra.DropIndex) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref, false)
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

	return plan.NewDropIndex(index, indexer, stmt), nil
}

func (this *builder) VisitAlterIndex(stmt *algebra.AlterIndex) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref, false)
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

	if _, ok := index.(datastore.Index3); !ok {
		return nil, errors.NewAlterIndexError()
	}

	return plan.NewAlterIndex(index, indexer, stmt, keyspace), nil
}

func (this *builder) VisitBuildIndexes(stmt *algebra.BuildIndexes) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref, false)
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

	return plan.NewBuildIndexes(keyspace, stmt), nil
}

func (this *builder) getNameKeyspace(ks *algebra.KeyspaceRef, dynamic bool) (datastore.Keyspace, error) {
	path := ks.Path()
	if path == nil {
		if dynamic {
			return nil, nil
		}
		return nil, errors.NewError(nil, "placeholder is not allowed in keyspace")
	}
	keyspace, err := datastore.GetKeyspace(path.Parts()...)

	if err != nil && this.indexAdvisor && ks.Path().Namespace() != "#system" &&
		(strings.Contains(err.TranslationKey(), "bucket_not_found") ||
			strings.Contains(err.TranslationKey(), "scope_not_found") ||
			strings.Contains(err.TranslationKey(), "keyspace_not_found")) {
		virtualKeyspace, err1 := this.getVirtualKeyspace(ks.Path().Namespace(), ks.Path().Parts())
		if err1 == nil {
			return virtualKeyspace, nil
		}
	}

	if err == nil && this.indexAdvisor {
		this.setKeyspaceFound()
	}

	return keyspace, err
}

func (this *builder) getVirtualKeyspace(namespaceStr string, path []string) (datastore.Keyspace, error) {
	ds := this.datastore
	namespace, err := ds.NamespaceByName(namespaceStr)
	if err != nil {
		return nil, err
	}
	if v, ok := namespace.(datastore.VirtualNamespace); ok {
		if this.indexAdvisor {
			this.setKeyspaceFound()
		}

		return v.VirtualKeyspaceByName(path)
	}
	return nil, errors.NewVirtualKSNotSupportedError(nil, "Namespace "+namespaceStr)
}
