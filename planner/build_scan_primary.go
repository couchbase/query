//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
)

func (this *builder) buildPrimaryScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	indexes []datastore.Index, id expression.Expression, force, exact, hasDeltaKeyspace bool) (
	plan.Operator, error) {
	primary, err := buildPrimaryIndex(keyspace, indexes, node, force, this.context)
	if primary == nil || err != nil {
		return nil, err
	}

	this.resetProjection()
	if this.group != nil {
		this.resetPushDowns()
	} else if !exact {
		this.resetOffsetLimit()
	}

	var indexOrder plan.IndexKeyOrders

	if this.order != nil {
		keys := datastore.IndexKeys{&datastore.IndexKey{id, datastore.IK_NONE}}
		entry := newIndexEntry(primary, keys, nil, len(keys), nil, 1, 1, 1, nil, nil,
			_EXACT_VALUED_SPANS, exact, []bool{true})
		ok := true
		if ok, indexOrder, _ = this.useIndexOrder(entry, entry.idxKeys); ok {
			this.maxParallelism = 1
		} else {
			this.resetPushDowns()
		}
	}

	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
	if primary.Type() != datastore.SYSTEM {
		this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
	}
	if primary3, ok := primary.(datastore.PrimaryIndex3); ok && useIndex3API(primary, this.context.IndexApiVersion()) {
		cost := OPT_COST_NOT_AVAIL
		cardinality := OPT_CARD_NOT_AVAIL
		size := OPT_SIZE_NOT_AVAIL
		frCost := OPT_COST_NOT_AVAIL
		if this.useCBO && this.keyspaceUseCBO(node.Alias()) {
			cost, cardinality, size, frCost = primaryIndexScanCost(primary, this.context.RequestId(), this.context)
		}

		skipNewKeys := false
		if primary3.Type() == datastore.SEQ_SCAN {
			skipNewKeys = this.skipKeyspace != "" && baseKeyspace.Keyspace() == this.skipKeyspace
			if skipNewKeys {
				this.mustSkipKeys = true
			}
			node.SetExtraPrivilege(auth.PRIV_QUERY_SEQ_SCAN)
		}

		return plan.NewPrimaryScan3(primary3, keyspace, node, this.offset, this.limit,
			plan.NewIndexProjection(0, true), indexOrder, nil, cost, cardinality,
			size, frCost, hasDeltaKeyspace, skipNewKeys), nil
	}

	var limit expression.Expression
	if exact {
		limit = offsetPlusLimit(this.offset, this.limit)
		this.resetOffset()
	}

	return plan.NewPrimaryScan(primary, keyspace, node, limit, hasDeltaKeyspace), nil
}

func buildPrimaryIndex(keyspace datastore.Keyspace, indexes []datastore.Index, node *algebra.KeyspaceTerm, force bool,
	context *PrepareContext) (primary datastore.PrimaryIndex, err error) {

	credentials := context.Credentials()
	inclSeqScan := keyspace.IsSystemCollection() || util.IsFeatureEnabled(context.FeatureControls(), util.N1QL_SEQ_SCAN)
	ok := false

	// Prefer hints
	for _, index := range indexes {
		if !index.IsPrimary() {
			continue
		} else if !inclSeqScan && index.Type() == datastore.SEQ_SCAN {
			continue
		}

		primary, ok = index.(datastore.PrimaryIndex)
		if ok {
			return
		} else {
			return nil, errors.NewPlanInternalError(fmt.Sprintf("buildPrimaryIndex: Unable to cast index %s to primary index",
				index.Name()))
		}
	}

	if force {
		return
	}

	// Return first online primary index
	indexers, er := keyspace.Indexers()
	if er != nil {
		return nil, er
	}

	for _, indexer := range indexers {
		if !inclSeqScan && indexer.Name() == datastore.SEQ_SCAN {
			continue
		}
		primaries, er := indexer.PrimaryIndexes()
		if er != nil {
			return nil, er
		}

		for _, primary = range primaries {
			state, _, er := primary.State()
			if er != nil {
				return nil, er
			}

			if state == datastore.ONLINE {
				return
			}
		}
	}

	if primary == nil {

		err := datastore.CheckBucketAccess(credentials, nil, node.Path().Parts())

		if err != nil {
			return nil, err
		}
		return nil, errors.NewWrapPlanError(errors.NewNoPrimaryIndexError(node.PathString()))
	}

	return nil, errors.NewWrapPlanError(errors.NewPrimaryIndexOfflineError(primary.Name()))
}
