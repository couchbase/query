//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
)

const (
	KS_PLAN_DONE        = 1 << iota // planning is done for this keyspace
	KS_ONCLAUSE_ONLY                // use ON-clause only for planning
	KS_IS_UNNEST                    // unnest alias
	KS_IN_CORR_SUBQ                 // in correlated subquery
	KS_HAS_DOC_COUNT                // docCount retrieved for keyspace
	KS_PRIMARY_TERM                 // primary term
	KS_OUTER_FILTERS                // OUTER filters have been classified
	KS_INDEX_HINT_ERROR             // index hint error
	KS_JOIN_HINT_ERROR              // join hint error
)

type BaseKeyspace struct {
	name          string
	keyspace      string
	filters       Filters
	joinfilters   Filters
	dnfPred       expression.Expression
	origPred      expression.Expression
	onclause      expression.Expression
	outerlevel    int32
	ksFlags       uint32
	docCount      int64
	unnests       map[string]string
	unnestIndexes map[datastore.Index][]string
	node          algebra.SimpleFromTerm
	optBit        int32
	indexHints    []algebra.OptimHint
	joinHints     []algebra.OptimHint
}

func NewBaseKeyspace(name string, path *algebra.Path, node algebra.SimpleFromTerm,
	optBit int32) *BaseKeyspace {

	var keyspace string

	// for expression scans we don't have a keyspace and leave it empty
	if path != nil {

		// we use the full name, except for buckets, where we look for the underlying default collection
		// this has to be done for CBO, so that we can use the same distributions for buckets and
		// default collections, when explicitly referenced
		if path.IsCollection() {
			keyspace = path.SimpleString()
		} else {
			ks, _ := datastore.GetKeyspace(path.Parts()...)

			// if we can't find it, we use a token full name
			if ks != nil {
				keyspace = ks.QualifiedName()
			} else {
				keyspace = path.SimpleString()
			}
		}
	}

	return &BaseKeyspace{
		name:     name,
		keyspace: keyspace,
		node:     node,
		optBit:   optBit,
	}
}

func (this *BaseKeyspace) PlanDone() bool {
	return (this.ksFlags & KS_PLAN_DONE) != 0
}

func (this *BaseKeyspace) SetPlanDone() {
	this.ksFlags |= KS_PLAN_DONE
}

func (this *BaseKeyspace) UnsetPlanDone() {
	this.ksFlags &^= KS_PLAN_DONE
}

func (this *BaseKeyspace) OnclauseOnly() bool {
	return (this.ksFlags & KS_ONCLAUSE_ONLY) != 0
}

func (this *BaseKeyspace) SetOnclauseOnly() {
	this.ksFlags |= KS_ONCLAUSE_ONLY
}

func (this *BaseKeyspace) IsUnnest() bool {
	return (this.ksFlags & KS_IS_UNNEST) != 0
}

func (this *BaseKeyspace) SetUnnest() {
	this.ksFlags |= KS_IS_UNNEST
}

func (this *BaseKeyspace) IsInCorrSubq() bool {
	return (this.ksFlags & KS_IN_CORR_SUBQ) != 0
}

func (this *BaseKeyspace) SetInCorrSubq() {
	this.ksFlags |= KS_IN_CORR_SUBQ
}

func (this *BaseKeyspace) HasDocCount() bool {
	return (this.ksFlags & KS_HAS_DOC_COUNT) != 0
}

func (this *BaseKeyspace) SetHasDocCount() {
	this.ksFlags |= KS_HAS_DOC_COUNT
}

func (this *BaseKeyspace) IsPrimaryTerm() bool {
	return (this.ksFlags & KS_PRIMARY_TERM) != 0
}

func (this *BaseKeyspace) SetPrimaryTerm() {
	this.ksFlags |= KS_PRIMARY_TERM
}

func (this *BaseKeyspace) HasOuterFilters() bool {
	return (this.ksFlags & KS_OUTER_FILTERS) != 0
}

func (this *BaseKeyspace) SetOuterFilters() {
	this.ksFlags |= KS_OUTER_FILTERS
}

func (this *BaseKeyspace) IsAnsiJoin() bool {
	return this.node != nil && this.node.IsAnsiJoin()
}

func (this *BaseKeyspace) IsAnsiNest() bool {
	return this.node != nil && this.node.IsAnsiNest()
}

func (this *BaseKeyspace) IsCommaJoin() bool {
	return this.node != nil && this.node.IsCommaJoin()
}

func (this *BaseKeyspace) IsLateralJoin() bool {
	return this.node != nil && this.node.IsLateralJoin()
}

func CopyBaseKeyspaces(src map[string]*BaseKeyspace) map[string]*BaseKeyspace {
	return copyBaseKeyspaces(src, false)
}

func CopyBaseKeyspacesWithFilters(src map[string]*BaseKeyspace) map[string]*BaseKeyspace {
	return copyBaseKeyspaces(src, true)
}

func copyBaseKeyspaces(src map[string]*BaseKeyspace, copyFilter bool) map[string]*BaseKeyspace {
	dest := make(map[string]*BaseKeyspace, len(src))

	for _, kspace := range src {
		dest[kspace.name] = &BaseKeyspace{
			name:       kspace.name,
			keyspace:   kspace.keyspace,
			ksFlags:    kspace.ksFlags,
			outerlevel: kspace.outerlevel,
			docCount:   kspace.docCount,
			node:       kspace.node,
			optBit:     kspace.optBit,
		}
		if len(kspace.unnests) > 0 {
			dest[kspace.name].unnests = make(map[string]string, len(kspace.unnests))
			for a, k := range kspace.unnests {
				dest[kspace.name].unnests[a] = k
			}
			if len(kspace.unnestIndexes) > 0 {
				dest[kspace.name].unnestIndexes = make(map[datastore.Index][]string, len(kspace.unnestIndexes))
				for i, a := range kspace.unnestIndexes {
					a2 := make([]string, len(a))
					copy(a2, a)
					dest[kspace.name].unnestIndexes[i] = a2
				}
			}
		}
		// The optimizer hints kept in BaseKeyspace is a slice of pointers that points to
		// the "original" hints in a statement. The optimizer/planner subsequently
		// modify the hints (e.g. change hint state) based on plans being generated.
		// Thus when BaseKeyspace is being copied we copy the slice but the pointers
		// in the slice remains the original pointers, such that any modification
		// through BaseKeyspace is reflected in the original hint.
		if len(kspace.indexHints) > 0 {
			indexHints := make([]algebra.OptimHint, 0, len(kspace.indexHints))
			for _, hint := range kspace.indexHints {
				indexHints = append(indexHints, hint)
			}
			dest[kspace.name].indexHints = indexHints
		}
		if len(kspace.joinHints) > 0 {
			joinHints := make([]algebra.OptimHint, 0, len(kspace.joinHints))
			for _, hint := range kspace.joinHints {
				joinHints = append(joinHints, hint)
			}
			dest[kspace.name].joinHints = joinHints
		}
		if copyFilter {
			if len(kspace.filters) > 0 {
				dest[kspace.name].filters = kspace.filters.Copy()
			}
			if len(kspace.joinfilters) > 0 {
				dest[kspace.name].joinfilters = kspace.joinfilters.Copy()
			}
		}
	}

	return dest
}

func (this *BaseKeyspace) Name() string {
	return this.name
}

func (this *BaseKeyspace) Keyspace() string {
	return this.keyspace
}

func (this *BaseKeyspace) Filters() Filters {
	return this.filters
}

func (this *BaseKeyspace) JoinFilters() Filters {
	return this.joinfilters
}

func (this *BaseKeyspace) AddFilter(filter *Filter) {
	found := false
	for i := range this.filters {
		if filter.EquivalentTo(this.filters[i]) {
			found = true
			break
		}
	}
	if !found {
		this.filters = append(this.filters, filter)
	}
}

func (this *BaseKeyspace) AddJoinFilter(joinfilter *Filter) {
	found := false
	for i := range this.joinfilters {
		if joinfilter.EquivalentTo(this.joinfilters[i]) {
			found = true
			break
		}
	}
	if !found {
		this.joinfilters = append(this.joinfilters, joinfilter)
	}
}

func (this *BaseKeyspace) AddFilters(filters Filters) {
	this.filters = append(this.filters, filters...)
}

func (this *BaseKeyspace) AddJoinFilters(joinfilters Filters) {
	this.joinfilters = append(this.joinfilters, joinfilters...)
}

func (this *BaseKeyspace) SetFilters(filters, joinfilters Filters) {
	this.filters = filters
	this.joinfilters = joinfilters
}

func (this *BaseKeyspace) DnfPred() expression.Expression {
	return this.dnfPred
}

func (this *BaseKeyspace) OrigPred() expression.Expression {
	return this.origPred
}

func (this *BaseKeyspace) Onclause() expression.Expression {
	return this.onclause
}

func (this *BaseKeyspace) SetPreds(dnfPred, origPred, onclause expression.Expression) {
	this.dnfPred = dnfPred
	this.origPred = origPred
	this.onclause = onclause
}

func (this *BaseKeyspace) Outerlevel() int32 {
	return this.outerlevel
}

func (this *BaseKeyspace) SetOuterlevel(outerlevel int32) {
	this.outerlevel = outerlevel
}

func (this *BaseKeyspace) IsOuter() bool {
	return (this.outerlevel > 0)
}

// document count for keyspaces, 0 for others (ExpressionTerm, SubqueryTerm)
func (this *BaseKeyspace) DocCount() int64 {
	return this.docCount
}

func (this *BaseKeyspace) SetDocCount(docCount int64) {
	this.docCount = docCount
}

func (this *BaseKeyspace) Node() algebra.SimpleFromTerm {
	return this.node
}

func (this *BaseKeyspace) SetNode(node algebra.SimpleFromTerm) {
	this.node = node
}

func (this *BaseKeyspace) OptBit() int32 {
	return this.optBit
}

// unnests is only populated for the primary keyspace term
func (this *BaseKeyspace) AddUnnestAlias(alias, keyspace string, size int) {
	if this.unnests == nil {
		this.unnests = make(map[string]string, size)
	}
	this.unnests[alias] = keyspace
}

func (this *BaseKeyspace) GetUnnests() map[string]string {
	return this.unnests
}

// if an UNNEST SCAN is used, this.unnestIndexes is a map that points to
// the UNNEST aliases for the UNNEST SCAN. In case of multiple levels of
// UNNEST with a nested array index key, the array of UNNEST aliases is
// populated in an inside-out fashion. E.g.:
//
//	ALL ARRAY (ALL ARRAY u FOR u IN v.arr2 END) FOR v IN arr1 END
//	... UNNEST d.arr1 AS a UNNEST a.arr2 AS b
//
// the array of aliases will be ["b", "a"]
func (this *BaseKeyspace) AddUnnestIndex(index datastore.Index, alias string) {
	if this.unnestIndexes == nil {
		this.unnestIndexes = make(map[datastore.Index][]string, len(this.unnests))
	}
	aliases, _ := this.unnestIndexes[index]
	for _, a := range aliases {
		if a == alias {
			// already exists
			return
		}
	}
	this.unnestIndexes[index] = append(aliases, alias)
}

func (this *BaseKeyspace) GetUnnestIndexes() map[datastore.Index][]string {
	return this.unnestIndexes
}

func (this *BaseKeyspace) AddIndexHint(indexHint algebra.OptimHint) {
	this.indexHints = append(this.indexHints, indexHint)
}

func (this *BaseKeyspace) IndexHints() []algebra.OptimHint {
	return this.indexHints
}

func (this *BaseKeyspace) AddJoinHint(joinHint algebra.OptimHint) {
	this.joinHints = append(this.joinHints, joinHint)
}

func (this *BaseKeyspace) JoinHints() []algebra.OptimHint {
	return this.joinHints
}

func (this *BaseKeyspace) HasIndexHintError() bool {
	return (this.ksFlags & KS_INDEX_HINT_ERROR) != 0
}

func (this *BaseKeyspace) SetIndexHintError() {
	this.ksFlags |= KS_INDEX_HINT_ERROR
}

func (this *BaseKeyspace) UnsetIndexHintError() {
	this.ksFlags &^= KS_INDEX_HINT_ERROR
}

func (this *BaseKeyspace) HasJoinHintError() bool {
	return (this.ksFlags & KS_JOIN_HINT_ERROR) != 0
}

func (this *BaseKeyspace) SetJoinHintError() {
	this.ksFlags |= KS_JOIN_HINT_ERROR
}

func (this *BaseKeyspace) UnsetJoinHintError() {
	this.ksFlags &^= KS_JOIN_HINT_ERROR
}

func (this *BaseKeyspace) MarkHashUnavailable() {
	for _, hint := range this.joinHints {
		if hint.Type() == algebra.HINT_HASH {
			hint.SetError(algebra.HASH_JOIN_NOT_AVAILABLE)
		}
	}
}

func GetKeyspaceName(baseKeyspaces map[string]*BaseKeyspace, alias string) string {
	if baseKeyspace, ok := baseKeyspaces[alias]; ok {
		return baseKeyspace.Keyspace()
	}

	return ""
}
