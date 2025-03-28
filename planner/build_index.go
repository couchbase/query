//  Copyright 2014-Present Couchbase, Inc.  //
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package planner

import (
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
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

	return plan.NewQueryPlan(plan.NewCreatePrimaryIndex(keyspace, stmt)), nil
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
	if stmt.Keys().HasVector() || stmt.Include() != nil {
		if _, ok := indexer.(datastore.Indexer6); !ok {
			var cause string
			version := datastore.INDEXER6_VERSION
			if stmt.Include() != nil {
				cause = "Include clause present"
			} else {
				cause = "Index key has vector attribute"
			}
			return nil, errors.NewIndexerVersionError(version, cause)
		}
	}

	if stmt.Partition() != nil {
		if _, ok := indexer.(datastore.Indexer3); !ok {
			return nil, errors.NewPartitionIndexNotSupportedError()
		}
	}

	return plan.NewQueryPlan(plan.NewCreateIndex(keyspace, stmt)), nil
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

	index, ierr := indexer.IndexByName(stmt.Name())

	return plan.NewQueryPlan(plan.NewDropIndex(index, ierr, indexer, stmt)), nil
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

	index, ierr := indexer.IndexByName(stmt.Name())

	return plan.NewQueryPlan(plan.NewAlterIndex(index, ierr, indexer, stmt, keyspace)), nil
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

	return plan.NewQueryPlan(plan.NewBuildIndexes(keyspace, stmt)), nil
}

func (this *builder) getNameKeyspace(ks *algebra.KeyspaceRef, dynamic bool) (datastore.Keyspace, error) {
	path := ks.Path()
	if path == nil {
		if dynamic {
			return nil, nil
		}
		return nil, errors.NewPlanNoPlaceholderError()
	}
	start := util.Now()
	keyspace, err := datastore.GetKeyspace(path.Parts()...)
	this.recordSubTime("keyspace.metadata", util.Since(start))

	if err != nil && this.indexAdvisor && !ks.IsSystem() &&
		(strings.Contains(err.TranslationKey(), "bucket_not_found") ||
			strings.Contains(err.TranslationKey(), "scope_not_found") ||
			strings.Contains(err.TranslationKey(), "keyspace_not_found")) {
		virtualKeyspace, err1 := this.getVirtualKeyspace(ks.Path().Namespace(), ks.Path().Parts())
		if err1 == nil {
			return virtualKeyspace, nil
		}
	}

	if err != nil {
		parts := path.Parts()
		err2 := datastore.CheckBucketAccess(this.context.Credentials(), err, parts)

		if err2 != nil {
			return keyspace, err2
		}
	}

	if err == nil && this.indexAdvisor {
		this.setKeyspaceFound()
	}

	return keyspace, err
}

func (this *builder) getVirtualKeyspace(namespaceStr string, path []string) (datastore.Keyspace, error) {
	ds := this.datastore
	namespace, err := ds.NamespaceById(namespaceStr)
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
