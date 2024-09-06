//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"sort"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
)

const (
	KS_PLAN_DONE            = 1 << iota // planning is done for this keyspace
	KS_ONCLAUSE_ONLY                    // use ON-clause only for planning
	KS_IS_UNNEST                        // unnest alias
	KS_IN_CORR_SUBQ                     // in correlated subquery
	KS_HAS_DOC_COUNT                    // docCount retrieved for keyspace
	KS_PRIMARY_TERM                     // primary term
	KS_OUTER_FILTERS                    // OUTER filters have been classified
	KS_INDEX_HINT_ERROR                 // index hint error
	KS_JOIN_HINT_ERROR                  // join hint error
	KS_JOIN_FLTR_HINT_ERROR             // join filter hint error
	KS_IS_SYSTEM                        // system keyspace
	KS_IS_KEYSPACETERM                  // KeyspaceTerm
	KS_IS_EXPRTERM                      // ExprTerm
	KS_IS_SUBQTERM                      // SubqTerm
)

type BaseKeyspace struct {
	name          string
	keyspace      string
	filters       Filters
	joinfilters   Filters
	vectorfilters Filters
	dnfPred       expression.Expression
	origPred      expression.Expression
	onclause      expression.Expression
	outerlevel    int32
	ksFlags       uint32
	docCount      int64
	unnests       map[string]string
	unnestIndexes map[datastore.Index]*UnnestIndexInfo
	node          algebra.SimpleFromTerm
	optBit        int32
	indexHints    []algebra.OptimHint
	joinHints     []algebra.OptimHint
	joinFltrHints []algebra.OptimHint
	bfSource      map[string]*BFSource
	cardinality   float64
	size          int64
	projection    []string
}

func NewBaseKeyspace(name string, path *algebra.Path, node algebra.SimpleFromTerm, optBit int32) (*BaseKeyspace, time.Duration) {

	var keyspace string
	var duration time.Duration
	var ksFlags uint32

	// for expression scans we don't have a keyspace and leave it empty
	if path != nil {

		// we use the full name, except for buckets, where we look for the underlying default collection
		// this has to be done for CBO, so that we can use the same distributions for buckets and
		// default collections, when explicitly referenced
		if path.IsCollection() {
			keyspace = path.SimpleString()
		} else {
			start := util.Now()
			ks, _ := datastore.GetKeyspace(path.Parts()...)
			duration = util.Since(start)

			// if we can't find it, we use a token full name
			if ks != nil {
				keyspace = ks.QualifiedName()
			} else {
				keyspace = path.SimpleString()
			}
		}
		if path.IsSystem() {
			ksFlags |= KS_IS_SYSTEM
		}
	}

	switch term := node.(type) {
	case *algebra.KeyspaceTerm:
		ksFlags |= KS_IS_KEYSPACETERM
	case *algebra.ExpressionTerm:
		if term.IsKeyspace() {
			ksFlags |= KS_IS_KEYSPACETERM
			node = term.KeyspaceTerm()
		} else {
			ksFlags |= KS_IS_EXPRTERM
		}
	case *algebra.SubqueryTerm:
		ksFlags |= KS_IS_SUBQTERM
	}

	return &BaseKeyspace{
		name:     name,
		keyspace: keyspace,
		ksFlags:  ksFlags,
		node:     node,
		optBit:   optBit,
	}, duration
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

func (this *BaseKeyspace) IsSystem() bool {
	return (this.ksFlags & KS_IS_SYSTEM) != 0
}

func (this *BaseKeyspace) IsKeyspaceTerm() bool {
	return (this.ksFlags & KS_IS_KEYSPACETERM) != 0
}

func (this *BaseKeyspace) IsExpressionTerm() bool {
	return (this.ksFlags & KS_IS_EXPRTERM) != 0
}

func (this *BaseKeyspace) IsSubqueryTerm() bool {
	return (this.ksFlags & KS_IS_SUBQTERM) != 0
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

func (this *BaseKeyspace) HasInferJoinHint() bool {
	return this.node != nil && this.node.HasInferJoinHint()
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
				dest[kspace.name].unnestIndexes = make(map[datastore.Index]*UnnestIndexInfo, len(kspace.unnestIndexes))
				for i, idxInfo := range kspace.unnestIndexes {
					if idxInfo != nil {
						a2 := make([]string, len(idxInfo.aliases))
						copy(a2, idxInfo.aliases)
						dest[kspace.name].unnestIndexes[i] = &UnnestIndexInfo{
							selec:   idxInfo.selec,
							aliases: a2,
						}
					}
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
		if len(kspace.joinFltrHints) > 0 {
			joinFltrHints := make([]algebra.OptimHint, 0, len(kspace.joinFltrHints))
			for _, hint := range kspace.joinFltrHints {
				joinFltrHints = append(joinFltrHints, hint)
			}
			dest[kspace.name].joinFltrHints = joinFltrHints
		}
		if len(kspace.projection) > 0 {
			dest[kspace.name].projection = make([]string, len(kspace.projection))
			copy(dest[kspace.name].projection, kspace.projection)
		}
		if copyFilter {
			if len(kspace.filters) > 0 {
				dest[kspace.name].filters = kspace.filters.Copy()
			}
			if len(kspace.joinfilters) > 0 {
				dest[kspace.name].joinfilters = kspace.joinfilters.Copy()
			}
		}
		if len(kspace.vectorfilters) > 0 {
			dest[kspace.name].vectorfilters = kspace.vectorfilters.Copy()
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

func (this *BaseKeyspace) VectorFilters() Filters {
	return this.vectorfilters
}

func (this *BaseKeyspace) AddFilter(filter *Filter) {
	this.filters = append(this.filters, filter)
}

func (this *BaseKeyspace) AddJoinFilter(joinfilter *Filter) {
	this.joinfilters = append(this.joinfilters, joinfilter)
}

func (this *BaseKeyspace) AddVectorFilter(vectorfilter *Filter) {
	this.vectorfilters = append(this.vectorfilters, vectorfilter)
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
	switch term := node.(type) {
	case *algebra.KeyspaceTerm:
		this.ksFlags |= KS_IS_KEYSPACETERM
	case *algebra.ExpressionTerm:
		if term.IsKeyspace() {
			this.ksFlags |= KS_IS_KEYSPACETERM
			node = term.KeyspaceTerm()
		} else {
			this.ksFlags |= KS_IS_EXPRTERM
		}
	case *algebra.SubqueryTerm:
		this.ksFlags |= KS_IS_SUBQTERM
	}
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

func (this *BaseKeyspace) HasUnnest() bool {
	return len(this.unnests) > 0
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
		this.unnestIndexes = make(map[datastore.Index]*UnnestIndexInfo, len(this.unnests))
	}
	if idxInfo, ok := this.unnestIndexes[index]; ok && idxInfo != nil {
		for _, a := range idxInfo.aliases {
			if a == alias {
				// already exists
				return
			}
		}
		idxInfo.aliases = append(idxInfo.aliases, alias)
	} else {
		this.unnestIndexes[index] = &UnnestIndexInfo{
			aliases: []string{alias},
		}
	}
}

func (this *BaseKeyspace) UpdateUnnestIndexSelec(index datastore.Index, selec float64) {
	if idxInfo, ok := this.unnestIndexes[index]; ok {
		idxInfo.selec = selec
	}
}

func (this *BaseKeyspace) GetUnnestIndexes() map[datastore.Index]*UnnestIndexInfo {
	return this.unnestIndexes
}

func (this *BaseKeyspace) GetUnnestIndexAliases(index datastore.Index) []string {
	if idxInfo, ok := this.unnestIndexes[index]; ok {
		return idxInfo.aliases
	}
	return nil
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

func (this *BaseKeyspace) AddJoinFltrHint(joinFltrHint algebra.OptimHint) {
	this.joinFltrHints = append(this.joinFltrHints, joinFltrHint)
}

func (this *BaseKeyspace) JoinFltrHints() []algebra.OptimHint {
	return this.joinFltrHints
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

func (this *BaseKeyspace) HasJoinFltrHintError() bool {
	return (this.ksFlags & KS_JOIN_FLTR_HINT_ERROR) != 0
}

func (this *BaseKeyspace) SetJoinFltrHintError() {
	this.ksFlags |= KS_JOIN_FLTR_HINT_ERROR
}

func (this *BaseKeyspace) UnsetJoinFltrHintError() {
	this.ksFlags &^= KS_JOIN_FLTR_HINT_ERROR
}

func (this *BaseKeyspace) MarkHashUnavailable() {
	for _, hint := range this.joinHints {
		if hint.Type() == algebra.HINT_HASH {
			hint.SetError(algebra.HASH_JOIN_NOT_AVAILABLE)
		}
	}
}

func (this *BaseKeyspace) JoinHint() algebra.JoinHint {
	for _, hint := range this.joinHints {
		switch hint.State() {
		case algebra.HINT_STATE_ERROR, algebra.HINT_STATE_INVALID:
			// ignore
		default:
			// there should be only a single join hint
			switch hint := hint.(type) {
			case *algebra.HintHash:
				switch hint.Option() {
				case algebra.HASH_OPTION_BUILD:
					return algebra.USE_HASH_BUILD
				case algebra.HASH_OPTION_PROBE:
					return algebra.USE_HASH_PROBE
				default:
					return algebra.USE_HASH_EITHER
				}
			case *algebra.HintNL:
				return algebra.USE_NL
			case *algebra.HintNoHash:
				return algebra.NO_USE_HASH
			case *algebra.HintNoNL:
				return algebra.NO_USE_NL
			}
		}
	}
	return algebra.JOIN_HINT_NONE
}

func (this *BaseKeyspace) HasJoinFilterHint() bool {
	for _, hint := range this.joinFltrHints {
		switch hint.State() {
		case algebra.HINT_STATE_ERROR, algebra.HINT_STATE_INVALID:
			// ignore
		default:
			// there should be only a single join filter hint
			switch hint.(type) {
			case *algebra.HintJoinFilter:
				return true
			}
		}
	}
	return false
}

func (this *BaseKeyspace) HasNoJoinFilterHint() bool {
	for _, hint := range this.joinFltrHints {
		switch hint.State() {
		case algebra.HINT_STATE_ERROR, algebra.HINT_STATE_INVALID:
			// ignore
		default:
			// there should be only a single join filter hint
			switch hint.(type) {
			case *algebra.HintNoJoinFilter:
				return true
			}
		}
	}
	return false
}

func (this *BaseKeyspace) HasIndexAllHint() bool {
	for _, hint := range this.indexHints {
		switch hint.State() {
		case algebra.HINT_STATE_ERROR, algebra.HINT_STATE_INVALID:
			// ignore
		default:
			// there should be only a single index_all hint
			switch hint.(type) {
			case *algebra.HintIndexAll:
				return true
			}
		}
	}
	return false
}

func (this *BaseKeyspace) MarkIndexHintError(err string) {
	for _, hint := range this.indexHints {
		if hint.State() == algebra.HINT_STATE_UNKNOWN {
			hint.SetError(err)
		}
	}
}

func (this *BaseKeyspace) MarkJoinHintError(err string) {
	for _, hint := range this.joinHints {
		if hint.State() == algebra.HINT_STATE_UNKNOWN {
			hint.SetError(err)
		}
	}
}

func (this *BaseKeyspace) MarkJoinFltrHintError(err string) {
	for _, hint := range this.joinFltrHints {
		if hint.State() == algebra.HINT_STATE_UNKNOWN {
			hint.SetError(err)
		}
	}
}

func (this *BaseKeyspace) Cardinality() float64 {
	return this.cardinality
}

func (this *BaseKeyspace) SetCardinality(cardinality float64) {
	this.cardinality = cardinality
}

func (this *BaseKeyspace) Size() int64 {
	return this.size
}

func (this *BaseKeyspace) SetSize(size int64) {
	this.size = size
}

func (this *BaseKeyspace) GetAllBFSource() map[string]*BFSource {
	return this.bfSource
}

func (this *BaseKeyspace) GetBFSource(joinAlias string) *BFSource {
	if bfSource, ok := this.bfSource[joinAlias]; ok {
		return bfSource
	}
	return nil
}

func (this *BaseKeyspace) AddBFSource(joinKeyspace *BaseKeyspace, index datastore.Index,
	self, other expression.Expression, fltr *Filter) {

	joinAlias := joinKeyspace.Name()
	if this.bfSource == nil {
		this.bfSource = make(map[string]*BFSource, 4)
	}
	selfStr := self.String()
	bfSource, ok := this.bfSource[joinAlias]
	if !ok {
		bfInfos := make(map[string]*BFInfo)
		bfInfos[selfStr] = newBFInfo(index, self, other, fltr)
		bfSource = newBFSource(joinKeyspace, bfInfos)
		this.bfSource[joinAlias] = bfSource
	} else {
		bfInfos := bfSource.bfInfos
		selec := fltr.selec
		if curInfo, ok := bfInfos[selfStr]; ok {
			if other.EquivalentTo(curInfo.other) {
				if _, ok := curInfo.indexes[index]; !ok {
					curInfo.indexes[index] = true
				}
			} else if (selec > 0.0) && (curInfo.fltr.selec <= 0.0 || selec < curInfo.fltr.selec) {
				// don't expect this to happen often, this will be something like:
				// r.c1 = t.c1 and r.c1 = t.c2
				curInfo.fltr = fltr
				curInfo.other = other
			}
			// otherwise just ignore
		} else {
			bfInfos[selfStr] = newBFInfo(index, self, other, fltr)
		}
	}
}

func (this *BaseKeyspace) RemoveBFSource(joinAlias string, index datastore.Index) {
	if bfSource, ok := this.bfSource[joinAlias]; ok {
		for exp, bfinfo := range bfSource.BFInfos() {
			if bfinfo.HasIndex(index) {
				bfinfo.DropIndex(index)
				if bfinfo.Empty() {
					bfSource.DropBFInfo(exp)
					if bfSource.Empty() {
						delete(this.bfSource, joinAlias)
					}
				}
			}
		}
	}
	return
}

func (this *BaseKeyspace) GetAllJoinFilterExprs(index datastore.Index) expression.Expressions {
	allExprs := make(expression.Expressions, 0, len(this.bfSource))
	for _, bfSource := range this.bfSource {
		for _, bfinfo := range bfSource.BFInfos() {
			if _, ok := bfinfo.indexes[index]; ok {
				allExprs = append(allExprs, bfinfo.fltr.FltrExpr())
			}
		}
	}
	return allExprs
}

func (this *BaseKeyspace) GetAllJoinFilterBits() int32 {
	allBits := int32(0)
	for _, bfSource := range this.bfSource {
		allBits |= bfSource.joinKeyspace.optBit
	}
	return allBits
}

func (this *BaseKeyspace) CheckJoinFilterIndexes(ops []plan.Operator) {
	indexes := make(map[datastore.Index]bool)
	gatherIndexes(indexes, ops...)
	for _, bfSource := range this.bfSource {
		for index, _ := range bfSource.selecs {
			if _, ok := indexes[index]; !ok {
				delete(bfSource.selecs, index)
			}
		}
	}
}

func (this *BaseKeyspace) HasEarlyProjection() bool {
	return len(this.projection) > 0
}

func (this *BaseKeyspace) SetEarlyProjection(names map[string]bool) {
	this.projection = make([]string, 0, len(names))
	for n, _ := range names {
		this.projection = append(this.projection, n)
	}
	// sort for explain stability
	sort.Strings(this.projection)
}

func (this *BaseKeyspace) EarlyProjection() []string {
	return this.projection
}

func (this *BaseKeyspace) GetVectorPred() expression.Expression {
	var vpred expression.Expression
	for _, fl := range this.vectorfilters {
		if vpred == nil {
			vpred = fl.fltrExpr
		} else {
			vpred = expression.NewAnd(vpred, fl.fltrExpr)
		}
	}
	return vpred
}

func GetKeyspaceName(baseKeyspaces map[string]*BaseKeyspace, alias string) string {
	if baseKeyspace, ok := baseKeyspaces[alias]; ok {
		return baseKeyspace.Keyspace()
	}

	return ""
}

type UnnestIndexInfo struct {
	selec   float64
	aliases []string
}

func (this *UnnestIndexInfo) GetSelec() float64 {
	return this.selec
}

func (this *UnnestIndexInfo) SetSelec(selec float64) {
	this.selec = selec
}

func (this *UnnestIndexInfo) GetAliases() []string {
	return this.aliases
}
