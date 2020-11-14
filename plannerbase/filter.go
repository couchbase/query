//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plannerbase

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

const (
	FLTR_IS_JOIN       = 1 << iota // is this originally a join filter
	FLTR_IS_ONCLAUSE               // is this an ON-clause filter for ANSI JOIN
	FLTR_IS_DERIVED                // is this a derived filter
	FLTR_IS_UNNEST                 // is this ann unnest filter (inherited)
	FLTR_SELEC_DONE                // calculation of selectivity is done
	FLTR_HAS_DEF_SELEC             // has default selectivity
	FLTR_IN_INDEX_SPAN             // used in index span
	FLTR_IN_HASH_JOIN              // used as join filter for hash join
	FLTR_HAS_SUBQ                  // has subquery
	FLTR_HAS_ADJ_SELEC             // has adjusted selectivity
	FLTR_PRIMARY_JOIN              // join on meta id
)

const TEMP_PLAN_FLAGS = (FLTR_IN_INDEX_SPAN | FLTR_IN_HASH_JOIN)

type Filter struct {
	fltrExpr      expression.Expression // filter expression
	origExpr      expression.Expression // original filter expression
	keyspaces     map[string]string     // keyspace references
	origKeyspaces map[string]string     // original keyspace references
	fltrFlags     uint32                // filter flags
	selec         float64               // filter selectivity
	arrSelec      float64               // filter selectivity for array index
	adjSelec      float64               // selectivity adjustment
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
		fltrExpr:  this.fltrExpr.Copy(),
		fltrFlags: this.fltrFlags,
		selec:     this.selec,
		arrSelec:  this.arrSelec,
	}

	if this.origExpr != nil {
		rv.origExpr = this.origExpr.Copy()
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

func (this *Filter) SetAdjustedSelec() {
	this.fltrFlags |= FLTR_HAS_ADJ_SELEC
}

func (this *Filter) HasAdjustedSelec() bool {
	return (this.fltrFlags & FLTR_HAS_ADJ_SELEC) != 0
}

func (this *Filter) SetPrimaryJoin() {
	this.fltrFlags |= FLTR_PRIMARY_JOIN
}

func (this *Filter) IsPrimaryJoin() bool {
	return (this.fltrFlags & FLTR_PRIMARY_JOIN) != 0
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
		// part of current ON-clause?
		if SubsetOf(onclause, this.fltrExpr) {
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
					return errors.NewPlanInternalError(fmt.Sprintf("moveJoinFilters: keyspace mismatch: %s vs %s", baseKeyspace.name, ksName))
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
