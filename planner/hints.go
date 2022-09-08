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
				var gsi, fts bool
				gsiIndexes := make([]string, 0, len(indexes))
				ftsIndexes := make([]string, 0, len(indexes))
				for _, idx := range indexes {
					switch idx.Using() {
					case datastore.DEFAULT, datastore.GSI:
						gsi = true
						if idx.Name() != "" {
							gsiIndexes = append(gsiIndexes, idx.Name())
						}
					case datastore.FTS:
						fts = true
						if idx.Name() != "" {
							ftsIndexes = append(ftsIndexes, idx.Name())
						}
					}
				}
				if gsi {
					newHint := algebra.NewDerivedIndexHint(alias, gsiIndexes)
					baseKeyspace.AddIndexHint(newHint)
					newHints = append(newHints, newHint)
				}
				if fts {
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

	numJoinFilterHint := 0

	// For INDEX/INDEX_FTS/NO_INDEX/NO_INDEX_FTS/USE_NL/USE_HASH/NO_USE_NL/NO_USE_HASH hints,
	// check for duplicated hint specification and add hints to baseKeyspace.
	// Note we don't allow mixing of USE style hints (specified after a keyspace in query text)
	// with same type of hint specified up front
	for _, hint := range optimHints.Hints() {
		if hint.Derived() {
			continue
		}

		var keyspace string
		var joinHint algebra.JoinHint
		var indexHint, joinFilterHint, negative bool

		switch hint := hint.(type) {
		case *algebra.HintIndex:
			keyspace = hint.Keyspace()
			indexHint = true
		case *algebra.HintFTSIndex:
			keyspace = hint.Keyspace()
			indexHint = true
		case *algebra.HintNoIndex:
			keyspace = hint.Keyspace()
			negative = true
			indexHint = true
		case *algebra.HintNoFTSIndex:
			keyspace = hint.Keyspace()
			negative = true
			indexHint = true
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
		case *algebra.HintNoNL:
			keyspace = hint.Keyspace()
			joinHint = algebra.NO_USE_NL
			negative = true
		case *algebra.HintNoHash:
			keyspace = hint.Keyspace()
			joinHint = algebra.NO_USE_HASH
			negative = true
		case *algebra.HintJoinFilter:
			keyspace = hint.Keyspace()
			joinFilterHint = true
		case *algebra.HintNoJoinFilter:
			keyspace = hint.Keyspace()
			joinFilterHint = true
			negative = true
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
			} else if baseKeyspace.HasJoinFilterHint() {
				if joinHint == algebra.USE_HASH_BUILD {
					hint.SetError(algebra.HASH_JOIN_BUILD_JOIN_FILTER)
					for _, curHint := range baseKeyspace.JoinFltrHints() {
						curHint.SetError(algebra.HASH_JOIN_BUILD_JOIN_FILTER)
					}
				} else if joinHint == algebra.USE_NL {
					hint.SetError(algebra.NL_JOIN_JOIN_FILTER)
					for _, curHint := range baseKeyspace.JoinFltrHints() {
						curHint.SetError(algebra.NL_JOIN_JOIN_FILTER)
					}
				}
			}
			baseKeyspace.AddJoinHint(hint)
		} else if indexHint {
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
				}
			}
			if negative {
				checkDupIndexes(baseKeyspace.IndexHints(), hint)

				switch hint := hint.(type) {
				case *algebra.HintNoIndex:
					if len(hint.Indexes()) == 0 {
						hint.SetNotFollowed()
					}
				case *algebra.HintNoFTSIndex:
					if len(hint.Indexes()) == 0 {
						hint.SetNotFollowed()
					}
				}
			}
			baseKeyspace.AddIndexHint(hint)
		} else if joinFilterHint {
			curHints := baseKeyspace.JoinFltrHints()
			if len(curHints) > 0 {
				// duplicated join hint
				hint.SetError(algebra.DUPLICATED_JOIN_FLTR_HINT + keyspace)
				for _, curHint := range curHints {
					curHint.SetError(algebra.DUPLICATED_JOIN_FLTR_HINT + keyspace)
				}
			} else if !negative {
				if baseKeyspace.JoinHint() == algebra.USE_HASH_BUILD {
					hint.SetError(algebra.HASH_JOIN_BUILD_JOIN_FILTER)
					for _, curHint := range baseKeyspace.JoinHints() {
						curHint.SetError(algebra.HASH_JOIN_BUILD_JOIN_FILTER)
					}
				} else if baseKeyspace.JoinHint() == algebra.USE_NL {
					hint.SetError(algebra.NL_JOIN_JOIN_FILTER)
					for _, curHint := range baseKeyspace.JoinHints() {
						curHint.SetError(algebra.NL_JOIN_JOIN_FILTER)
					}
				}
				numJoinFilterHint++
			}
			baseKeyspace.AddJoinFltrHint(hint)
		}
	}

	if numJoinFilterHint == len(baseKeyspaces) {
		for _, baseKeyspace := range baseKeyspaces {
			for _, hint := range baseKeyspace.JoinFltrHints() {
				hint.SetError(algebra.ALL_TERM_JOIN_FILTER)
			}
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

// If an index is specified in both INDEX/INDEX_FTS and the corresponding NO_INDEX/NO_INDEX_FTS hint,
// then both hints are ignored, but the index is considered during planning, i.e. the INDEX/INDEX_FTS
// hint overrides the NO_INDEX/NO_INDEX_FTS hint. In this case, remove this index from the
// NO_INDEX/NO_INDEX_FTS hint.
func checkDupIndexes(hints []algebra.OptimHint, hint algebra.OptimHint) {
	var indexes []string
	fts := false
	switch hint := hint.(type) {
	case *algebra.HintNoIndex:
		indexes = hint.Indexes()
	case *algebra.HintNoFTSIndex:
		indexes = hint.Indexes()
		fts = true
	}
	if len(indexes) == 0 {
		return
	}

	newIndexes := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		found := false
		for _, other := range hints {
			if other.State() != algebra.HINT_STATE_UNKNOWN {
				continue
			}
			var otherIndexes []string
			switch other := other.(type) {
			case *algebra.HintIndex:
				if !fts {
					otherIndexes = other.Indexes()
				}
			case *algebra.HintFTSIndex:
				if fts {
					otherIndexes = other.Indexes()
				}
			}
			for _, oidx := range otherIndexes {
				if idx == oidx {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			newIndexes = append(newIndexes, idx)
		}
	}

	// if some indexes removed, reset the indexes array, we cannot change the original
	// hint pointer since that's what's kept in stmt.OptimHints()
	if len(indexes) != len(newIndexes) {
		switch hint := hint.(type) {
		case *algebra.HintNoIndex:
			hint.SetIndexes(newIndexes)
		case *algebra.HintNoFTSIndex:
			hint.SetIndexes(newIndexes)
		}
	}
	return
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

func markDMLOrderedHintError(optHints *algebra.OptimHints) {
	if optHints != nil {
		for _, hint := range optHints.Hints() {
			if hint.Type() == algebra.HINT_ORDERED {
				hint.SetError(algebra.DML_ORDERED_HINT_ERR)
			}
		}
	}
}

func setDuplicateIndexHintError(hint algebra.OptimHint, keyspace string) {
	switch hint := hint.(type) {
	case *algebra.HintIndex, *algebra.HintNoIndex:
		hint.SetError(algebra.DUPLICATED_INDEX_HINT + keyspace)
	case *algebra.HintFTSIndex, *algebra.HintNoFTSIndex:
		hint.SetError(algebra.DUPLICATED_INDEX_FTS_HINT + keyspace)
	}
}

func setNonKeyspaceIndexHintError(hint algebra.OptimHint, keyspace string) {
	switch hint := hint.(type) {
	case *algebra.HintIndex, *algebra.HintNoIndex:
		hint.SetError(algebra.NON_KEYSPACE_INDEX_HINT + keyspace)
	case *algebra.HintFTSIndex, *algebra.HintNoFTSIndex:
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
func (this *builder) markOptimHints(alias string, includeJoin bool) (err error) {
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

	if !includeJoin {
		return nil
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

func (this *builder) MarkJoinFilterHints() (err error) {
	if len(this.subChildren) > 0 {
		err = checkJoinFilterHint(this.baseKeyspaces, this.subChildren...)
		if err != nil {
			return err
		}
	}
	if len(this.children) > 0 {
		err = checkJoinFilterHint(this.baseKeyspaces, this.children...)
		if err != nil {
			return err
		}
	}

	for _, baseKeyspace := range this.baseKeyspaces {
		joinFltrHintError := baseKeyspace.HasJoinFltrHintError()
		for _, hint := range baseKeyspace.JoinFltrHints() {
			switch hint.State() {
			case algebra.HINT_STATE_ERROR, algebra.HINT_STATE_INVALID, algebra.HINT_STATE_FOLLOWED, algebra.HINT_STATE_NOT_FOLLOWED:
			// nothing to do
			case algebra.HINT_STATE_UNKNOWN:
				if joinFltrHintError {
					hint.SetNotFollowed()
				} else {
					hint.SetFollowed()
				}
			default:
				return errors.NewPlanInternalError("markJoinFilterHints: invalid hint state")
			}
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
