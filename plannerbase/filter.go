//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

const (
	FLTR_IS_JOIN            = 1 << iota // is this originally a join filter
	FLTR_IS_ONCLAUSE                    // is this an ON-clause filter for ANSI JOIN
	FLTR_IS_DERIVED                     // is this a derived filter
	FLTR_IS_UNNEST                      // is this ann unnest filter (inherited)
	FLTR_SELEC_DONE                     // calculation of selectivity is done
	FLTR_HAS_DEF_SELEC                  // has default selectivity
	FLTR_IN_INDEX_SPAN                  // used in index span
	FLTR_IN_HASH_JOIN                   // used as join filter for hash join
	FLTR_HAS_SUBQ                       // has subquery
	FLTR_HAS_ADJ_ARR_SELEC              // has adjusted selectivity
	FLTR_PRIMARY_JOIN                   // join on meta id
	FLTR_DERIVED_EQJOIN                 // derived equi-join filter
	FLTR_ADJUST_JOIN_SELEC              // join selectivity adjusted
	FLTR_SAV_INDEX_SPAN                 // saved IN_INDEX_SPAN flag
	FLTR_HAS_ADJ_BIT_SELEC              // has adjusted bit-filter selectivity
	FLTR_NOT_PUSHABLE                   // ON-clause filter that is not pushable
	FLTR_HAS_AVG_DIST_SELEC             // has selectivity of (1/distinct)
	FLTR_HAS_ADJ_DIST_SELEC             // has adjusted selectivity of (1/distinct)
)

const TEMP_PLAN_FLAGS = (FLTR_IN_INDEX_SPAN | FLTR_IN_HASH_JOIN)

type Filter struct {
	fltrExpr      expression.Expression // filter expression
	origExpr      expression.Expression // original filter expression
	keyspaces     map[string]string     // keyspace references
	origKeyspaces map[string]string     // original keyspace references
	optBits       int32                 // keyspace references in bits
	fltrFlags     uint32                // filter flags
	selec         float64               // filter selectivity
	arrSelec      float64               // filter selectivity for array index
	adjSelec      float64               // selectivity adjustment
	eqLeft        EqElem                // left-hand-side equivalent class element
	eqRight       EqElem                // right-hand-side equivalent class element
	eqfId         int32                 // equivalent filter id
}

type EqElem interface {
	HasLink(pos int32) bool
	SetLink(pos int32)
}

type Filters []*Filter

func NewFilter(fltrExpr, origExpr expression.Expression, keyspaces, origKeyspaces map[string]string,
	isOnclause bool, isJoin bool) *Filter {

	rv := &Filter{
		fltrExpr:      fltrExpr,
		origExpr:      origExpr,
		keyspaces:     keyspaces,
		origKeyspaces: origKeyspaces,
	}

	if isOnclause {
		rv.fltrFlags |= FLTR_IS_ONCLAUSE
	}
	if isJoin {
		rv.fltrFlags |= FLTR_IS_JOIN
	}

	return rv
}

func (this *Filter) Copy() *Filter {
	rv := &Filter{
		fltrExpr:  expression.Copy(this.fltrExpr),
		origExpr:  expression.Copy(this.origExpr),
		optBits:   this.optBits,
		fltrFlags: this.fltrFlags,
		selec:     this.selec,
		arrSelec:  this.arrSelec,
		adjSelec:  this.adjSelec,
		eqLeft:    this.eqLeft,
		eqRight:   this.eqRight,
		eqfId:     this.eqfId,
	}

	rv.keyspaces = make(map[string]string, len(this.keyspaces))
	for key, value := range this.keyspaces {
		rv.keyspaces[key] = value
	}

	rv.origKeyspaces = make(map[string]string, len(this.origKeyspaces))
	for key, value := range this.origKeyspaces {
		rv.origKeyspaces[key] = value
	}

	return rv
}

func (this *Filter) EquivalentTo(other *Filter) bool {
	return expression.Equivalent(this.fltrExpr, other.fltrExpr) &&
		expression.Equivalent(this.origExpr, other.origExpr) &&
		len(this.keyspaces) == len(other.keyspaces) &&
		len(this.origKeyspaces) == len(other.origKeyspaces) &&
		this.optBits == other.optBits &&
		this.fltrFlags == other.fltrFlags
}

func (this *Filter) IsOnclause() bool {
	return (this.fltrFlags & FLTR_IS_ONCLAUSE) != 0
}

func (this *Filter) IsJoin() bool {
	return (this.fltrFlags & FLTR_IS_JOIN) != 0
}

func (this *Filter) IsDerived() bool {
	return (this.fltrFlags & FLTR_IS_DERIVED) != 0
}

func (this *Filter) SetDerived() {
	this.fltrFlags |= FLTR_IS_DERIVED
}

func (this *Filter) IsUnnest() bool {
	return (this.fltrFlags & FLTR_IS_UNNEST) != 0
}

func (this *Filter) SetUnnest() {
	this.fltrFlags |= FLTR_IS_UNNEST
}

func (this *Filter) IsSelecDone() bool {
	return (this.fltrFlags & FLTR_SELEC_DONE) != 0
}

func (this *Filter) SetSelecDone() {
	this.fltrFlags |= FLTR_SELEC_DONE
}

func (this *Filter) HasDefSelec() bool {
	return (this.fltrFlags & FLTR_HAS_DEF_SELEC) != 0
}

func (this *Filter) SetDefSelec() {
	this.fltrFlags |= FLTR_HAS_DEF_SELEC
}

func (this *Filter) HasIndexFlag() bool {
	return (this.fltrFlags & FLTR_IN_INDEX_SPAN) != 0
}

func (this *Filter) HasHJFlag() bool {
	return (this.fltrFlags & FLTR_IN_HASH_JOIN) != 0
}

func (this *Filter) SetIndexFlag() {
	this.fltrFlags |= FLTR_IN_INDEX_SPAN
}

func (this *Filter) SetHJFlag() {
	this.fltrFlags |= FLTR_IN_HASH_JOIN
}

func (this *Filter) HasPlanFlags() bool {
	return (this.fltrFlags & TEMP_PLAN_FLAGS) != 0
}

func (this *Filter) SetSubq() {
	this.fltrFlags |= FLTR_HAS_SUBQ
}

func (this *Filter) HasSubq() bool {
	return (this.fltrFlags & FLTR_HAS_SUBQ) != 0
}

func (this *Filter) SetAdjustedArrSelec() {
	this.fltrFlags |= FLTR_HAS_ADJ_ARR_SELEC
}

func (this *Filter) HasAdjustedArrSelec() bool {
	return (this.fltrFlags & FLTR_HAS_ADJ_ARR_SELEC) != 0
}

func (this *Filter) SetAdjustedBitSelec() {
	this.fltrFlags |= FLTR_HAS_ADJ_BIT_SELEC
}

func (this *Filter) UnsetAdjustedBitSelec() {
	this.fltrFlags &^= FLTR_HAS_ADJ_BIT_SELEC
}

func (this *Filter) HasAdjustedBitSelec() bool {
	return (this.fltrFlags & FLTR_HAS_ADJ_BIT_SELEC) != 0
}

func (this *Filter) SetPrimaryJoin() {
	this.fltrFlags |= FLTR_PRIMARY_JOIN
}

func (this *Filter) IsPrimaryJoin() bool {
	return (this.fltrFlags & FLTR_PRIMARY_JOIN) != 0
}

func (this *Filter) SetDerivedEqJoin() {
	this.fltrFlags |= FLTR_DERIVED_EQJOIN
}

func (this *Filter) IsDerivedEqJoin() bool {
	return (this.fltrFlags & FLTR_DERIVED_EQJOIN) != 0
}

func (this *Filter) SetAdjustJoinSelec() {
	this.fltrFlags |= FLTR_ADJUST_JOIN_SELEC
}

func (this *Filter) HasAdjustJoinSelec() bool {
	return (this.fltrFlags & FLTR_ADJUST_JOIN_SELEC) != 0
}

func (this *Filter) SetNotPushable() {
	this.fltrFlags |= FLTR_NOT_PUSHABLE
}

func (this *Filter) NotPushable() bool {
	return (this.fltrFlags & FLTR_NOT_PUSHABLE) != 0
}

func (this *Filter) HasAvgDistSelec() bool {
	return (this.fltrFlags & FLTR_HAS_AVG_DIST_SELEC) != 0
}

func (this *Filter) SetAvgDistSelec() {
	this.fltrFlags |= FLTR_HAS_AVG_DIST_SELEC
}

func (this *Filter) UnsetAvgDistSelec() {
	this.fltrFlags &^= FLTR_HAS_AVG_DIST_SELEC
}

func (this *Filter) HasAdjustAvgDistSelec() bool {
	return (this.fltrFlags & FLTR_HAS_ADJ_DIST_SELEC) != 0
}

func (this *Filter) SetAdjustAvgDistSelec() {
	this.fltrFlags |= FLTR_HAS_ADJ_DIST_SELEC
}

func (this *Filter) FltrExpr() expression.Expression {
	return this.fltrExpr
}

func (this *Filter) OrigExpr() expression.Expression {
	return this.origExpr
}

func (this *Filter) Keyspaces() map[string]string {
	return this.keyspaces
}

func (this *Filter) OrigKeyspaces() map[string]string {
	return this.origKeyspaces
}

func (this *Filter) OptBits() int32 {
	return this.optBits
}

func (this *Filter) SetOptBits(optBits int32) {
	this.optBits = optBits
}

func (this *Filter) Selec() float64 {
	return this.selec
}

func (this *Filter) SetSelec(selec float64) {
	this.selec = selec
}

func (this *Filter) ArraySelec() float64 {
	return this.arrSelec
}

func (this *Filter) SetArraySelec(arrSelec float64) {
	this.arrSelec = arrSelec
}

func (this *Filter) AdjSelec() float64 {
	return this.adjSelec
}

func (this *Filter) SetAdjSelec(adjSelec float64) {
	this.adjSelec = adjSelec
}

func (this *Filter) IsPostjoinFilter(onclause expression.Expression, outer bool) bool {
	if onclause == nil {
		return true
	}
	if this.IsOnclause() {
		notPushable := this.NotPushable()
		// part of current ON-clause?
		if SubsetOf(onclause, this.fltrExpr) {
			return notPushable
		}
		if this.origExpr != nil {
			if SubsetOf(onclause, this.origExpr) {
				return notPushable
			}
		} else if IsDerivedExpr(this.fltrExpr) {
			var err error
			dnfExpr := onclause.Copy()
			dnf := NewDNF(dnfExpr, true, false)
			dnfExpr, err = dnf.Map(dnfExpr)
			if err != nil {
				return false
			}
			if SubsetOf(dnfExpr, this.fltrExpr) {
				return notPushable
			}
		}

		// Special handling of volatile function expression, which returns false for
		// EquivalentTo() (which is called by SubsetOf()) even for the same expr
		// (see FunctionBase.EquivalentTo()).
		// For such expression do not assume it is postjoin.
		if this.fltrExpr.HasVolatileExpr() {
			return false
		}

		// if it's not part of the current ON-clause, it must be specified
		// in an ON-clause for a later inner join (pushed), in which case
		// treat it as postjoin
		return true
	} else if this.IsJoin() {
		// join filter specified in the WHERE clause
		return true
	} else if outer {
		// selection filter on subservient side of an outer join is evaluated postjoin
		// if it is not part of the ON-clause
		return true
	}

	return false
}

// check whether the join filter is a single join filter involving any of the keyspaces provided
func (this *Filter) SingleJoinFilter(unnestKeyspaces map[string]bool) bool {
	for ks, _ := range this.keyspaces {
		if _, ok := unnestKeyspaces[ks]; !ok {
			return false
		}
	}

	return true
}

// get equivalent class filter info
func (this *Filter) EqLeft() EqElem {
	return this.eqLeft
}

func (this *Filter) EqRight() EqElem {
	return this.eqRight
}

func (this *Filter) EqfId() int32 {
	return this.eqfId
}

// setup equivalent class filter info
func (this *Filter) SetEqElems(left, right EqElem) {
	this.eqLeft = left
	this.eqRight = right
}

func (this *Filter) SetEqfId(eqfId int32) {
	this.eqfId = eqfId
}

// is this filter an equi-join filter involving the two EqElems passed in?
func (this *Filter) EquivalentEqJoin(left, right EqElem) bool {
	return this.eqLeft != nil && this.eqRight != nil &&
		((this.eqLeft == left && this.eqRight == right) ||
			(this.eqLeft == right && this.eqRight == left))
}

// Once a keyspace has been visited, join filters referring to this keyspace can remove
// this keyspace reference since it's now "available", and if there are no other
// keyspace references the join filter can be moved to filters
func MoveJoinFilters(keyspace string, baseKeyspace *BaseKeyspace) error {
	compact := false
	for i, jfl := range baseKeyspace.joinfilters {
		delete(jfl.keyspaces, keyspace)
		if len(jfl.keyspaces) == 1 {
			for ksName, _ := range jfl.keyspaces {
				if baseKeyspace.name != ksName {
					return errors.NewPlanInternalError(fmt.Sprintf("moveJoinFilters: keyspace mismatch: %s vs %s",
						baseKeyspace.name, ksName))
				}
				break
			}

			// move to filters
			baseKeyspace.filters = append(baseKeyspace.filters, jfl)
			baseKeyspace.joinfilters[i] = nil
			compact = true
		}
	}

	if compact == true {
		curlen := len(baseKeyspace.joinfilters)
		curlen = trimJoinFilters(baseKeyspace.joinfilters, curlen)
		newlen := curlen
		for i := 0; i < curlen; i++ {
			if i >= newlen {
				break
			}
			if baseKeyspace.joinfilters[i] == nil {
				if i < newlen-1 {
					baseKeyspace.joinfilters[i] = baseKeyspace.joinfilters[newlen-1]
				}
				baseKeyspace.joinfilters[newlen-1] = nil
				newlen = trimJoinFilters(baseKeyspace.joinfilters, newlen-1)
			}
		}
		baseKeyspace.joinfilters = baseKeyspace.joinfilters[:newlen]
	}

	return nil
}

// trim nil entries at the end of joinfilters slice
func trimJoinFilters(joinfilters Filters, curlen int) (newlen int) {
	newlen = curlen
	if newlen == 0 {
		return
	}

	for {
		if joinfilters[newlen-1] == nil {
			newlen--
			if newlen == 0 {
				return
			}
		} else {
			return
		}
	}
}

func (this Filters) Copy() Filters {
	filters := make(Filters, 0, len(this))
	for _, fl := range this {
		filters = append(filters, fl.Copy())
	}
	return filters
}

func (this Filters) ClearIndexFlag() {
	for _, fltr := range this {
		fltr.fltrFlags &^= FLTR_IN_INDEX_SPAN
	}
}

func (this Filters) ClearHashFlag() {
	for _, fltr := range this {
		fltr.fltrFlags &^= FLTR_IN_HASH_JOIN
	}
}

func (this Filters) ClearPlanFlags() {
	for _, fltr := range this {
		fltr.fltrFlags &^= TEMP_PLAN_FLAGS
	}
}

func (this Filters) SaveIndexFlag() {
	for _, fltr := range this {
		if (fltr.fltrFlags & FLTR_IN_INDEX_SPAN) != 0 {
			fltr.fltrFlags |= FLTR_SAV_INDEX_SPAN
		} else {
			fltr.fltrFlags &^= FLTR_SAV_INDEX_SPAN
		}
	}
}

func (this Filters) RestoreIndexFlag() {
	for _, fltr := range this {
		if (fltr.fltrFlags & FLTR_SAV_INDEX_SPAN) != 0 {
			fltr.fltrFlags |= FLTR_IN_INDEX_SPAN
			fltr.fltrFlags &^= FLTR_SAV_INDEX_SPAN
		} else {
			fltr.fltrFlags &^= FLTR_IN_INDEX_SPAN
		}
	}
}
