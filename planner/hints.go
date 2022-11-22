//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	base "github.com/couchbase/query/plannerbase"
)

// derive new OptimHints based on USE INDEX and USE NL/USE HASH specified in the query
func deriveOptimHints(baseKeyspaces map[string]*base.BaseKeyspace, optimHints *algebra.OptimHints) *algebra.OptimHints {
	var newHints []algebra.OptimHint

	for alias, baseKeyspace := range baseKeyspaces {
		node := baseKeyspace.Node()

		if node == nil {
			// unnest alias
			continue
		}

		joinHint := node.JoinHint()
		if joinHint != algebra.JOIN_HINT_NONE {
			var newHint algebra.OptimHint
			switch joinHint {
			case algebra.USE_HASH_BUILD:
				newHint = algebra.NewDerivedHashHint(alias, algebra.HASH_OPTION_BUILD)
			case algebra.USE_HASH_PROBE:
				newHint = algebra.NewDerivedHashHint(alias, algebra.HASH_OPTION_PROBE)
			case algebra.USE_NL:
				newHint = algebra.NewDerivedNLHint(alias)
			}
			if newHint != nil {
				baseKeyspace.AddJoinHint(newHint)
				newHints = append(newHints, newHint)
			}
		}

		ksterm := algebra.GetKeyspaceTerm(node)
		if ksterm != nil {
			indexes := ksterm.Indexes()
			if len(indexes) > 0 {
				gsiIndexes := make(algebra.IndexRefs, 0, len(indexes))
				ftsIndexes := make(algebra.IndexRefs, 0, len(indexes))
				for _, idx := range indexes {
					switch idx.Using() {
					case datastore.DEFAULT, datastore.GSI:
						gsiIndexes = append(gsiIndexes, idx)
					case datastore.FTS:
						ftsIndexes = append(ftsIndexes, idx)
					}
				}
				if len(gsiIndexes) > 0 {
					newHint := algebra.NewDerivedIndexHint(alias, gsiIndexes)
					baseKeyspace.AddIndexHint(newHint)
					newHints = append(newHints, newHint)
				}
				if len(ftsIndexes) > 0 {
					newHint := algebra.NewDerivedFTSIndexHint(alias, ftsIndexes)
					baseKeyspace.AddIndexHint(newHint)
					newHints = append(newHints, newHint)
				}
			}
		}
	}

	if len(newHints) > 0 {
		if optimHints == nil {
			optimHints = algebra.NewOptimHints(nil, false)
		}

		// sort the new hints for explain purpose
		algebra.SortOptimHints(newHints)
		optimHints.AddHints(newHints)
	}

	return optimHints
}

func processOptimHints(baseKeyspaces map[string]*base.BaseKeyspace, optimHints *algebra.OptimHints) {
	if optimHints == nil {
		return
	}

	// For INDEX/INDEX_FTS/USE_NL/USE_HASH hints, add correpsonding hint in SimpleFromTerm
	// for ease of further processing.
	// Note we don't allow mixing of USE style hints (specified after a keyspace in query text)
	// with same type of hint specified up front
	for _, hint := range optimHints.Hints() {
		if hint.Derived() {
			continue
		}

		var keyspace string
		var indexes algebra.IndexRefs
		var joinHint algebra.JoinHint

		switch hint := hint.(type) {
		case *algebra.HintIndex:
			keyspace = hint.Keyspace()
			indexes = hint.Indexes()
		case *algebra.HintFTSIndex:
			keyspace = hint.Keyspace()
			indexes = hint.Indexes()
		case *algebra.HintNL:
			keyspace = hint.Keyspace()
			joinHint = algebra.USE_NL
		case *algebra.HintHash:
			keyspace = hint.Keyspace()
			switch hint.Option() {
			case algebra.HASH_OPTION_NONE:
				joinHint = algebra.USE_HASH_EITHER
			case algebra.HASH_OPTION_BUILD:
				joinHint = algebra.USE_HASH_BUILD
			case algebra.HASH_OPTION_PROBE:
				joinHint = algebra.USE_HASH_PROBE
			}
		}

		if keyspace == "" {
			// ignore hints that's not specific to a keyspace
			continue
		}

		baseKeyspace, ok := baseKeyspaces[keyspace]
		if !ok {
			// invalid keyspace specified
			hint.SetError(algebra.INVALID_KEYSPACE + keyspace)
			continue
		}

		node := baseKeyspace.Node()
		if node == nil {
			// invalid keyspace specified
			hint.SetError(algebra.INVALID_KEYSPACE + keyspace)
			continue
		}

		if joinHint != algebra.JOIN_HINT_NONE {
			curHints := baseKeyspace.JoinHints()
			if len(curHints) > 0 {
				// duplicated join hint
				hint.SetError(algebra.DUPLICATED_JOIN_HINT + keyspace)
				for _, curHint := range curHints {
					curHint.SetError(algebra.DUPLICATED_JOIN_HINT + keyspace)
				}
			} else {
				node.SetJoinHint(joinHint)
			}
			baseKeyspace.AddJoinHint(hint)
		}
		if len(indexes) > 0 {
			if hasDerivedHint(baseKeyspace.IndexHints()) {
				setDuplicateIndexHintError(hint, keyspace)
				for _, curHint := range baseKeyspace.IndexHints() {
					if !curHint.Derived() {
						continue
					}
					setDuplicateIndexHintError(curHint, keyspace)
				}
			} else {
				ksterm := algebra.GetKeyspaceTerm(node)
				if ksterm == nil {
					setNonKeyspaceIndexHintError(hint, keyspace)
				} else {
					curIndexes := ksterm.Indexes()
					curMap := make(map[string]bool, len(curIndexes)+len(indexes))
					newIndexes := make(algebra.IndexRefs, 0, len(curIndexes)+len(indexes))
					for _, idx := range curIndexes {
						if _, ok := curMap[idx.Name()]; ok {
							continue
						}
						curMap[idx.Name()] = true
						newIndexes = append(newIndexes, idx)
					}
					for _, idx := range indexes {
						if _, ok := curMap[idx.Name()]; ok {
							continue
						}
						curMap[idx.Name()] = true
						newIndexes = append(newIndexes, idx)
					}
					ksterm.SetIndexes(newIndexes)
				}
			}
			baseKeyspace.AddIndexHint(hint)
		}
	}
}

func hasDerivedHint(hints []algebra.OptimHint) bool {
	for _, hint := range hints {
		if hint.Derived() {
			return true
		}
	}
	return false
}

func hasOrderedHint(optHints *algebra.OptimHints) bool {
	if optHints != nil {
		for _, hint := range optHints.Hints() {
			if hint.Type() == algebra.HINT_ORDERED {
				// ordered hint is currently always followed
				hint.SetFollowed()
				return true
			}
		}
	}
	return false
}

func setDuplicateIndexHintError(hint algebra.OptimHint, keyspace string) {
	switch hint := hint.(type) {
	case *algebra.HintIndex:
		hint.SetError(algebra.DUPLICATED_INDEX_HINT + keyspace)
	case *algebra.HintFTSIndex:
		hint.SetError(algebra.DUPLICATED_INDEX_FTS_HINT + keyspace)
	}
}

func setNonKeyspaceIndexHintError(hint algebra.OptimHint, keyspace string) {
	switch hint := hint.(type) {
	case *algebra.HintIndex:
		hint.SetError(algebra.NON_KEYSPACE_INDEX_HINT + keyspace)
	case *algebra.HintFTSIndex:
		hint.SetError(algebra.NON_KEYSPACE_INDEX_FTS_HINT + keyspace)
	}
}

// Based on hint error flags in BaseKeyspace, mark the original hint (as FOLLOWED or NOT_FOLLOWED).
// The optimizer hints kept in BaseKeyspace are slices of pointers to the original hints in the
// statement, thus any modifications done here are reflected in the original hints.
// Note that when BaseKeyspace is copied we only copy the hint slices but keep the original hint
// pointers, thus we should only be marking optimizer hints when it's safe to do so, i.e. when
// the planning for a specific keyspace is "final". For RBO this is done as we consider each
// keyspace, for join enumeration this is done when a final plan is chosen.
func (this *builder) markOptimHints(alias string) (err error) {
	baseKeyspace, ok := this.baseKeyspaces[alias]
	if !ok {
		return errors.NewPlanInternalError("markOptimHintErrors: invalid alias specified: " + alias)
	}

	indexHintError := baseKeyspace.HasIndexHintError()
	for _, hint := range baseKeyspace.IndexHints() {
		switch hint.State() {
		case algebra.HINT_STATE_ERROR, algebra.HINT_STATE_INVALID, algebra.HINT_STATE_FOLLOWED, algebra.HINT_STATE_NOT_FOLLOWED:
			// nothing to do
		case algebra.HINT_STATE_UNKNOWN:
			if indexHintError {
				hint.SetNotFollowed()
			} else {
				hint.SetFollowed()
			}
		default:
			return errors.NewPlanInternalError("markOptimHints: invalid hint state")
		}
	}

	joinHintError := baseKeyspace.HasJoinHintError()
	for _, hint := range baseKeyspace.JoinHints() {
		switch hint.State() {
		case algebra.HINT_STATE_ERROR, algebra.HINT_STATE_INVALID, algebra.HINT_STATE_FOLLOWED, algebra.HINT_STATE_NOT_FOLLOWED:
			// nothing to do
		case algebra.HINT_STATE_UNKNOWN:
			if joinHintError {
				hint.SetNotFollowed()
			} else {
				hint.SetFollowed()
			}
		default:
			return errors.NewPlanInternalError("markOptimHints: invalid hint state")
		}
	}

	return nil
}

func (this *builder) gatherSubqueryTermHints() []*algebra.SubqOptimHints {
	var subqTermHints []*algebra.SubqOptimHints
	for _, ks := range this.baseKeyspaces {
		node := ks.Node()
		if node != nil {
			if subqTerm, ok := node.(*algebra.SubqueryTerm); ok {
				optimHints := subqTerm.Subquery().OptimHints()
				if optimHints != nil {
					subqHints := algebra.NewSubqOptimHints(subqTerm.Alias(), optimHints)
					subqTermHints = append(subqTermHints, subqHints)
				}
			}
		}
	}
	return subqTermHints
}

func removeSubqueryTermHints(optimHints *algebra.OptimHints, alias string) {
	if optimHints != nil {
		subqTermHints := optimHints.SubqTermHints()
		for i, subqTermHint := range subqTermHints {
			if subqTermHint != nil && subqTermHint.Alias() == alias {
				subqTermHints[i] = nil
				break
			}
		}
	}
}
