//  Copyright 2021-Present Couchbase, Inc.
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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
)

// derive new OptimHints based on USE INDEX and USE NL/USE HASH specified in the query
func deriveOptimHints(baseKeyspaces map[string]*base.BaseKeyspace, optimHints *algebra.OptimHints) *algebra.OptimHints {
	var newHints []algebra.OptimHint
	var derivedHints []algebra.OptimHint

	if optimHints != nil {
		derivedHints = make([]algebra.OptimHint, 0, len(optimHints.Hints()))
		for _, hint := range optimHints.Hints() {
			if hint.Derived() {
				derivedHints = append(derivedHints, hint)
			}
		}
	}

	for alias, baseKeyspace := range baseKeyspaces {
		node := baseKeyspace.Node()

		if node == nil {
			// unnest alias
			continue
		}

		var newHint algebra.OptimHint
		var found bool

		joinHint := node.JoinHint()
		if joinHint != algebra.JOIN_HINT_NONE {
			switch joinHint {
			case algebra.USE_HASH_BUILD:
				newHint, found = algebra.GetDerivedHashHint(derivedHints, alias, algebra.HASH_OPTION_BUILD)
			case algebra.USE_HASH_PROBE:
				newHint, found = algebra.GetDerivedHashHint(derivedHints, alias, algebra.HASH_OPTION_PROBE)
			case algebra.USE_NL:
				newHint, found = algebra.GetDerivedNLHint(derivedHints, alias)
			}
			if newHint != nil {
				baseKeyspace.AddJoinHint(newHint)
				if !found {
					newHints = append(newHints, newHint)
				}
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
					newHint, found = algebra.GetDerivedIndexHint(derivedHints, alias, gsiIndexes)
					if newHint != nil {
						baseKeyspace.AddIndexHint(newHint)
						if !found {
							newHints = append(newHints, newHint)
						}
					}
				}
				if fts {
					newHint, found = algebra.GetDerivedFTSIndexHint(derivedHints, alias, ftsIndexes)
					if newHint != nil {
						baseKeyspace.AddIndexHint(newHint)
						if !found {
							newHints = append(newHints, newHint)
						}
					}
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

func (this *builder) processOptimHints(optimHints *algebra.OptimHints) {
	if optimHints == nil {
		return
	}

	numJoinFilterHint := 0

	// For INDEX/INDEX_FTS/NO_INDEX/NO_INDEX_FTS/INDEX_ALL/USE_NL/USE_HASH/NO_USE_NL/NO_USE_HASH
	// hints, check for duplicated hint specification and add hints to baseKeyspace.
	// Note we don't allow mixing of USE style hints (specified after a keyspace in query text)
	// with same type of hint specified up front
	for _, hint := range optimHints.Hints() {
		if hint.Derived() {
			continue
		}

		var keyspace string
		var joinHint algebra.JoinHint
		var indexHint, indexAll, joinFilterHint, negative bool

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
		case *algebra.HintIndexAll:
			keyspace = hint.Keyspace()
			indexAll = true
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

		baseKeyspace, ok := this.baseKeyspaces[keyspace]
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
			indexHints := baseKeyspace.IndexHints()
			if hasIndexAllHint(indexHints) {
				setMixedIndexAllHintError(hint, keyspace)
				for _, curHint := range indexHints {
					if curHint.Type() == algebra.HINT_INDEX_ALL {
						setMixedIndexAllHintError(curHint, keyspace)
					}
				}
			} else if hasDerivedHint(indexHints) {
				setDuplicateIndexHintError(hint, keyspace)
				for _, curHint := range indexHints {
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
		} else if indexAll {
			indexHints := baseKeyspace.IndexHints()
			if len(indexHints) > 0 {
				for _, curHint := range indexHints {
					if curHint.Type() == algebra.HINT_INDEX_ALL {
						setDuplicateIndexAllHintError(curHint, keyspace)
						setDuplicateIndexAllHintError(hint, keyspace)
					} else {
						setMixedIndexAllHintError(curHint, keyspace)
						setMixedIndexAllHintError(hint, keyspace)
					}
				}
			} else {
				ksterm := algebra.GetKeyspaceTerm(node)
				if ksterm == nil {
					setNonKeyspaceIndexHintError(hint, keyspace)
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

	if numJoinFilterHint == len(this.baseKeyspaces) {
		for _, baseKeyspace := range this.baseKeyspaces {
			for _, hint := range baseKeyspace.JoinFltrHints() {
				hint.SetError(algebra.ALL_TERM_JOIN_FILTER)
			}
		}
	}

	for _, baseKeyspace := range this.baseKeyspaces {
		ksterm := algebra.GetKeyspaceTerm(baseKeyspace.Node())
		if ksterm == nil {
			continue
		}

		indexHints := baseKeyspace.IndexHints()
		if len(indexHints) == 0 {
			continue
		}

		var hasGsi, hasFts bool
		for _, hint := range indexHints {
			switch hint.(type) {
			case *algebra.HintIndex, *algebra.HintNoIndex, *algebra.HintIndexAll:
				hasGsi = true
			case *algebra.HintFTSIndex, *algebra.HintNoFTSIndex:
				hasFts = true
			}
		}

		start := util.Now()
		var hasErr bool
		var msg string
		var gsiIndexer, ftsIndexer datastore.Indexer
		var err error
		ks, _ := this.getTermKeyspace(ksterm)
		if ks == nil {
			hasErr = true
			msg = algebra.INVALID_KEYSPACE + ksterm.Alias()
		} else {
			if hasGsi {
				gsiIndexer, err = ks.Indexer(datastore.GSI)
				if err != nil {
					hasErr = true
				}
			}
			if hasFts {
				ftsIndexer, err = ks.Indexer(datastore.FTS)
				if err != nil {
					hasErr = true
				}
			}
		}

		if hasErr {
			// mark all index hints with error
			for _, hint := range indexHints {
				if hint.State() == algebra.HINT_STATE_UNKNOWN {
					if msg == "" {
						// indexer error
						switch hint.(type) {
						case *algebra.HintIndex, *algebra.HintNoIndex, *algebra.HintIndexAll:
							msg = algebra.GSI_INDEXER_NOT_AVAIL
						case *algebra.HintFTSIndex, *algebra.HintNoFTSIndex:
							msg = algebra.FTS_INDEXER_NOT_AVAIL
						default:
							msg = fmt.Sprintf("Unexpected hint in index hints: %T", hint)
						}
					}
					hint.SetError(msg)
				}
			}
		} else {
			for _, hint := range indexHints {
				var gsiIndexNames, ftsIndexNames []string
				switch hint := hint.(type) {
				case *algebra.HintIndex:
					gsiIndexNames = hint.Indexes()
				case *algebra.HintNoIndex:
					gsiIndexNames = hint.Indexes()
				case *algebra.HintFTSIndex:
					ftsIndexNames = hint.Indexes()
				case *algebra.HintNoFTSIndex:
					ftsIndexNames = hint.Indexes()
				case *algebra.HintIndexAll:
					gsiIndexNames = hint.Indexes()
				}

				var errIndexes string
				for _, idx := range gsiIndexNames {
					// allow `#sequentialscan` as index name
					if idx == "#sequentialscan" {
						continue
					}
					_, err = gsiIndexer.IndexByName(idx)
					if err != nil {
						if errIndexes != "" {
							errIndexes += ", "
						}
						errIndexes += idx
					}
				}
				if errIndexes != "" {
					hint.SetError(algebra.INVALID_GSI_INDEX + errIndexes)
					errIndexes = ""
				}

				for _, idx := range ftsIndexNames {
					_, err = ftsIndexer.IndexByName(idx)
					if err != nil {
						if errIndexes != "" {
							errIndexes += ", "
						}
						errIndexes += idx
					}
				}
				if errIndexes != "" {
					hint.SetError(algebra.INVALID_FTS_INDEX + errIndexes)
				}
			}
		}
		if hasGsi || hasFts {
			this.recordSubTime("index.metadata", util.Since(start))
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

func hasIndexAllHint(hints []algebra.OptimHint) bool {
	for _, hint := range hints {
		if hint.Type() == algebra.HINT_INDEX_ALL {
			return true
		}
	}
	return false
}

func setDuplicateIndexAllHintError(hint algebra.OptimHint, keyspace string) {
	hint.SetError(algebra.DUPLICATED_INDEX_ALL_HINT + keyspace)
}

func setMixedIndexAllHintError(hint algebra.OptimHint, keyspace string) {
	switch hint := hint.(type) {
	case *algebra.HintIndex, *algebra.HintNoIndex:
		hint.SetError(algebra.MIXED_INDEX_WITH_INDEX_ALL + keyspace)
	case *algebra.HintFTSIndex, *algebra.HintNoFTSIndex:
		hint.SetError(algebra.MIXED_INDEX_FTS_WITH_INDEX_ALL + keyspace)
	case *algebra.HintIndexAll:
		hint.SetError(algebra.MIXED_INDEX_ALL_WITH_INDEX + keyspace)
	}
}

func setNonKeyspaceIndexHintError(hint algebra.OptimHint, keyspace string) {
	switch hint := hint.(type) {
	case *algebra.HintIndex, *algebra.HintNoIndex:
		hint.SetError(algebra.NON_KEYSPACE_INDEX_HINT + keyspace)
	case *algebra.HintFTSIndex, *algebra.HintNoFTSIndex:
		hint.SetError(algebra.NON_KEYSPACE_INDEX_FTS_HINT + keyspace)
	case *algebra.HintIndexAll:
		hint.SetError(algebra.NON_KEYSPACE_INDEX_ALL_HINT + keyspace)
	}
}

// if an INDEX_ALL hint is specified, make sure all listed indexes are in sargables
func checkIndexAllSargable(baseKeyspace *base.BaseKeyspace, sargables map[datastore.Index]*indexEntry) bool {
	var indexNames []string
	for _, hint := range baseKeyspace.IndexHints() {
		if indexAll, ok := hint.(*algebra.HintIndexAll); ok && hint.State() == algebra.HINT_STATE_UNKNOWN {
			indexNames = indexAll.Indexes()
			break
		}
	}

	if len(indexNames) == 0 {
		return false
	}

	indexes := make(map[string]datastore.Index, len(sargables))
	for index, _ := range sargables {
		indexes[index.Name()] = index
	}

	// match indexNames (in hint) with sargable indexes
	for _, idxName := range indexNames {
		if _, ok := indexes[idxName]; !ok {
			return false
		}
	}

	return true
}

// for lookup join and index join
func (this *builder) markJoinIndexAllHint(alias string) error {
	baseKeyspace, ok := this.baseKeyspaces[alias]
	if !ok {
		return errors.NewPlanInternalError(fmt.Sprintf("markJoinIndexAllHint: baseKeyspace for %s not found", alias))
	}
	for _, hint := range baseKeyspace.IndexHints() {
		if hint.Type() == algebra.HINT_INDEX_ALL {
			hint.SetError(algebra.INDEX_ALL_LEGACY_JOIN + alias)
		}
	}
	return nil
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

func (this *builder) MarkJoinFilterHints(children, subChildren []plan.Operator) (err error) {
	if len(subChildren) > 0 {
		err = checkJoinFilterHint(this.baseKeyspaces, subChildren...)
		if err != nil {
			return err
		}
	}
	if len(children) > 0 {
		err = checkJoinFilterHint(this.baseKeyspaces, children...)
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
