//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	base "github.com/couchbase/query/plannerbase"
)

// derive new OptimHints based on USE INDEX and USE NL/USE HASH specified in the query
func deriveOptimHints(baseKeyspaces map[string]*base.BaseKeyspace, optimHints *algebra.OptimHints) *algebra.OptimHints {
	if optimHints == nil {
		optimHints = algebra.NewOptimHints(nil, false)
	}

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
				hint.SetError(algebra.DUPLICATED_INDEX_HINT + keyspace)
				for _, curHint := range baseKeyspace.IndexHints() {
					if !curHint.Derived() {
						continue
					}
					curHint.SetError(algebra.DUPLICATED_INDEX_HINT + keyspace)
				}
			} else {
				ksterm := algebra.GetKeyspaceTerm(node)
				if ksterm == nil {
					hint.SetError(algebra.NON_KEYSPACE_INDEX_HINT + keyspace)
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
