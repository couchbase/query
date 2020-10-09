//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

/*
  For Window aggregates builds the Order and WindowAggregate operators
  Goals:
       Minimize number of sorts required by combining PARTITION BY + ORDER BY, collation, nulls position as much as possible.
       Keep the length PARTITION BY + ORDER BY expressions in DESC order, so that we can enhance later do partial sort
       With in the same sort
             Keep length PARTITION BY expressions in DESC order, so that we can hold less number of rows
*/

func (this *builder) visitWindowAggregates(windowAggs algebra.Aggregates) {

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	if this.useCBO && this.lastOp != nil {
		cost = this.lastOp.Cost()
		cardinality = this.lastOp.Cardinality()
	}

	// build the Window groups as described above
	for _, wOrderGroup := range this.buildWindowGroups(windowAggs) {

		// For each Sort required build Order operator
		order := wOrderGroup.sortGroups.buildOrder()
		if order != nil {
			if this.useCBO && cost > 0.0 && cardinality > 0.0 {
				scost, scardinality := getSortCost(this.baseKeyspaces,
					len(order.Terms()), cardinality, 0, 0)
				if scost > 0.0 && scardinality > 0.0 {
					cost += scost
					cardinality = scardinality
				} else {
					cost = OPT_COST_NOT_AVAIL
					cardinality = OPT_CARD_NOT_AVAIL
				}
			}
			this.addSubChildren(plan.NewOrder(order, nil, nil, cost, cardinality))
		}

		for i := len(wOrderGroup.pbys) - 1; i >= 0; i-- {
			if wOrderGroup.pbys[i] != nil && len(wOrderGroup.pbys[i].aggs) > 0 {
				if this.useCBO && cost > 0.0 && cardinality > 0.0 {
					cost, cardinality = getWindowAggCost(this.baseKeyspaces,
						wOrderGroup.pbys[i].aggs, cost, cardinality)
				}
				this.addSubChildren(plan.NewWindowAggregate(wOrderGroup.pbys[i].aggs, cost, cardinality))
			}
		}
	}

	// make all Order/WindowAggregate operators as Sequence
	this.addChildren(plan.NewSequence(this.subChildren...))
	this.subChildren = make([]plan.Operator, 0, 8)
}

/*
Aggregates in the partition group
*/
type WindowPartitionGroup struct {
	aggs algebra.Aggregates
}

type WindowPartitionGroups []*WindowPartitionGroup

/*
 Order Groups
*/
type WindowOrderGroup struct {
	sortGroups SortGroups
	pbys       WindowPartitionGroups
}

type WindowOrderGroups []*WindowOrderGroup

type SortGroup struct {
	groups     SortGroups
	expr       expression.Expression
	descending bool
	nullsPos   bool
	flags      uint32
	exprStr    string
	pos        int
	minPos     int
	maxPos     int
}

type SortGroups []*SortGroup

const (
	_SORT_POS_FXIED = 1 << iota
	_SORT_NULLS_FIXED
	_SORT_COLLATION_FIXED
	_SORT_TERM_GROUP
)

func hasEveryFlag(sflags, tflags uint32) bool {
	return (sflags != 0) && (sflags&tflags) == tflags
}

func hasAnyFlag(sflags, tflags uint32) bool {
	return (sflags & tflags) != 0
}

func (this *builder) buildWindowGroups(windowAggs algebra.Aggregates) WindowOrderGroups {
	windowOrderGroups := make(WindowOrderGroups, 0, len(windowAggs))
	var noOrderAggs algebra.Aggregates
	var wOrderGroup *WindowOrderGroup

outer:
	for _, agg := range windowAggs {
		wTerm := agg.WindowTerm()
		if wTerm.PartitionBy() != nil || wTerm.OrderBy() != nil {
			var pl, ol int
			pl = len(wTerm.PartitionBy())
			if wTerm.OrderBy() != nil {
				ol = len(wTerm.OrderBy().Terms())
			}

			// make SortGroups
			sortGroups := make(SortGroups, pl+ol)

			if wTerm.PartitionBy() != nil {
				// add all PBY as another SortGroups
				psortGroups := make(SortGroups, 0, pl)
				for _, expr := range wTerm.PartitionBy() {
					sg := &SortGroup{expr: expr, exprStr: expr.String(), minPos: 1, maxPos: pl}
					psortGroups = append(psortGroups, sg)
				}

				// len PBY == 1 then inline and fix position else make SortGroups as sub group
				if pl == 1 {
					psortGroups[0].flags = _SORT_POS_FXIED
					psortGroups[0].pos = pl
					sortGroups[0] = psortGroups[0]
				} else {
					sortGroups[0] = &SortGroup{groups: psortGroups}
				}
			}

			if wTerm.OrderBy() != nil {
				// OBY as SortGroups and Fix the position, collation and nulls
				flags := uint32(_SORT_POS_FXIED | _SORT_NULLS_FIXED | _SORT_COLLATION_FIXED)
				for i, term := range wTerm.OrderBy().Terms() {
					sg := &SortGroup{expr: term.Expression(), exprStr: term.Expression().String(),
						descending: term.Descending(), nullsPos: term.NullsPos(), flags: flags}
					sg.pos = pl + i + 1
					sg.minPos = pl + i + 1
					sg.maxPos = pl + i + 1
					sortGroups[pl+i] = sg
				}
			}

			// If sortGroups can be added any existing Order Group add it else make new one
			for _, wOrderGroup = range windowOrderGroups {
				if this.addToAnaylticalOrderGroup(wOrderGroup, sortGroups, agg) {
					this.addToWindowPartitionGroup(wOrderGroup, agg)
					continue outer
				}
			}

			wOrderGroup = &WindowOrderGroup{sortGroups: sortGroups}
			this.addToWindowPartitionGroup(wOrderGroup, agg)
			windowOrderGroups = append(windowOrderGroups, wOrderGroup)
		} else {
			// collect no PBY/ no OBY aggregates
			noOrderAggs = append(noOrderAggs, agg)
		}
	}

	if len(noOrderAggs) > 0 {
		// add no PBY/ no OBY aggregates at the end because those need whole result set.
		if len(windowOrderGroups) == 0 {
			windowOrderGroups = append(windowOrderGroups, &WindowOrderGroup{})
		}
		wOrderGroup = windowOrderGroups[len(windowOrderGroups)-1]

		for _, agg := range noOrderAggs {
			this.addToWindowPartitionGroup(wOrderGroup, agg)
		}
	}

	return windowOrderGroups
}

/*
 Check if given sortGroups can be added to wOrderGroup
 If needed wOrderGroup.sortGroups are modified.
 If not possible wOrderGroup.sortGroups are retained at original state
*/

func (this *builder) addToAnaylticalOrderGroup(wOrderGroup *WindowOrderGroup, sortGroups SortGroups,
	agg algebra.Aggregate) bool {
	if len(wOrderGroup.sortGroups) >= len(sortGroups) {
		// make copy of wOrderGroup.sortGroups
		aSortGroups := wOrderGroup.sortGroups.Copy()

		// sortGroups is subset of wOrderGroup.sortGroups (if required it modifies)
		if sortGroups.subSetOf(aSortGroups) {

			// restore original wOrderGroup.sortGroups
			wOrderGroup.sortGroups = aSortGroups
			return true
		}
	}
	return false
}

/*
  Add aggregate to given order group into right Partition group
*/
func (this *builder) addToWindowPartitionGroup(wOrderGroup *WindowOrderGroup, agg algebra.Aggregate) {
	wTerm := agg.WindowTerm()
	pos := len(wTerm.PartitionBy())
	if pos >= len(wOrderGroup.pbys) {
		// more partition groups are needed
		pbys := make(WindowPartitionGroups, pos+1)
		copy(pbys, wOrderGroup.pbys)
		wOrderGroup.pbys = pbys
	}

	if wOrderGroup.pbys[pos] == nil {
		// new partition group
		wOrderGroup.pbys[pos] = &WindowPartitionGroup{}
	}

	wOrderGroup.pbys[pos].aggs = append(wOrderGroup.pbys[pos].aggs, agg)
}

func (this SortGroups) subSetOf(other SortGroups) bool {
	// boundary conditions
	if len(this) > len(other) || len(other) <= 0 || len(this) <= 0 {
		return false
	}

	var ok bool
	var os *SortGroup
	var osPbyMap map[string]*SortGroup
	nPby := 0
	tonlyPby := false

	if other[0] != nil && len(other[0].groups) > 0 {
		// others PBY expression map
		nPby = len(other[0].groups)
		osPbyMap = other[0].groups.GetSortGroupStringMap(nPby)
	} else {
		osPbyMap = make(map[string]*SortGroup)
	}

	// check this PBY + OBY expressions
	for i, ts := range this {
		if ts == nil {
			continue
		}

		if len(ts.groups) > 0 {
			// this PBY expressions
			for _, tgs := range ts.groups {
				// This PBY expression must be others PBY expression or OBY until len(this PBY)
				_, ok = tgs.exprInSortGroups(other, nPby, len(ts.groups), osPbyMap)
				if !ok {
					return false
				}
			}
			tonlyPby = true
		} else {
			// this OBY expressions. get expression other exact position
			tonlyPby = false
			os = other[i]
			if os != nil && len(os.groups) > 0 {
				osPbyMap = os.groups.GetSortGroupStringMap(len(os.groups))
				os, _ = ts.exprInSortGroups(other, 0, 0, osPbyMap)
			}

			// none or expresion is not same
			if os == nil || ts.exprStr != os.exprStr {
				return false
			}

			// check positions
			checkFalg := (ts.pos == os.pos)

			// position and others Collation is fixed then this and other collation needs to be matched
			if checkFalg && hasAnyFlag(os.flags, _SORT_COLLATION_FIXED) {
				if ts.descending != os.descending {
					return false
				}
			} else {
				// transfer this collation to others and mark it fixed
				os.descending = ts.descending
				os.flags |= _SORT_COLLATION_FIXED
			}

			// position and others nulls is fixed then this and other nulls needs to be matched
			if checkFalg && hasAnyFlag(os.flags, _SORT_NULLS_FIXED) {
				if ts.nullsPos != os.nullsPos {
					return false
				}
			} else {
				// transfer this nulls to others and mark it fixed
				os.nullsPos = ts.nullsPos
				os.flags |= _SORT_NULLS_FIXED
			}

			// if positions are not fixed, fix position of this expr in others if needed split others PBY
			if !checkFalg && !other.splitAndFixPos(this, os, ts.pos) {
				return false
			}
		}
	}

	// this PBY only slpit others PBY with this PBY
	if tonlyPby && !other.splitAndFixPos(this, nil, len(this[0].groups)) {
		return false
	}

	return true
}

/*
 Split this SortGroups around orgTs and fix the pos
*/

func (this SortGroups) splitAndFixPos(other SortGroups, orgTs *SortGroup, pos int) bool {
	for i, ts := range this {
		if i > pos {
			// crossed pos
			return false
		} else if ts == nil || i+len(ts.groups) < pos {
			// not reached pos we needed
			continue
		}

		// remove the entry from the group
		tsPbyMap := ts.groups.GetSortGroupStringMap(len(ts.groups))
		if orgTs != nil {
			delete(tsPbyMap, orgTs.exprStr)
		}
		ltsGroup := make(SortGroups, 0, len(ts.groups))
		rtsGroup := make(SortGroups, 0, len(ts.groups))

		// add the entries that present other PBY group on left side
		for _, sts := range other[0].groups {
			if t, ok := tsPbyMap[sts.exprStr]; ok {
				t.maxPos = pos - 1
				ltsGroup = append(ltsGroup, t)
				delete(tsPbyMap, t.exprStr)
			}
		}

		if len(ltsGroup) == 1 {
			this[i] = ltsGroup[0]
			this[i].pos = this[i].minPos
		} else {
			ts.groups = ltsGroup
		}

		// add the current entry at pos
		if orgTs != nil {
			orgTs.pos = pos
			orgTs.minPos = pos
			orgTs.maxPos = pos
			this[pos-1] = orgTs
		}

		// add remaining entires into right group
		for _, t := range tsPbyMap {
			t.minPos = pos + 1
			t.maxPos = t.minPos + len(tsPbyMap)
			rtsGroup = append(rtsGroup, t)
		}

		if len(rtsGroup) == 1 {
			rtsGroup[0].pos = pos + 1
			this[pos] = rtsGroup[0]
		} else if len(rtsGroup) > 1 {
			this[pos] = &SortGroup{groups: rtsGroup}
		}
		return true
	}
	return false
}

/*
 Check SortGroup is in others PBY or others OBY until given position
*/

func (this *SortGroup) exprInSortGroups(other SortGroups, start, end int, osMap map[string]*SortGroup) (os *SortGroup, ok bool) {
	// If In the map (PBY) remove it so that it will not find it again
	if os, ok = osMap[this.exprStr]; ok {
		delete(osMap, this.exprStr)
		return
	}

	if start < end && end < len(other) {
		for _, os = range other[start:end] {
			if os != nil && this.exprStr == os.exprStr {
				return os, true
			}
		}
	}
	return nil, false
}

/*
 Give SortGroups make map of SortGroup until nth position
 PBY terms
*/

func (this SortGroups) GetSortGroupStringMap(nItems int) map[string]*SortGroup {
	sMap := make(map[string]*SortGroup, len(this))
	for i, s := range this {
		if i <= nItems && s != nil {
			if len(s.groups) > 0 {
				for _, gs := range s.groups {
					if gs != nil {
						sMap[gs.exprStr] = gs
					}
				}

			} else {
				sMap[s.exprStr] = s
			}
		}
	}
	return sMap
}

/*
  make OrderTerm from SortGroups
*/
func (this SortGroups) buildOrder() (order *algebra.Order) {
	if len(this) > 0 {
		terms := make(algebra.SortTerms, 0, len(this))
		for _, sg := range this {
			if sg == nil {
				continue
			} else if len(sg.groups) > 0 {
				for _, s := range sg.groups {
					if s != nil {
						term := algebra.NewSortTerm(s.expr, s.descending, s.nullsPos)
						terms = append(terms, term)
					}
				}
			} else {
				term := algebra.NewSortTerm(sg.expr, sg.descending, sg.nullsPos)
				terms = append(terms, term)
			}
		}
		order = algebra.NewOrder(terms)
	}

	return
}

/*
 Copy SortGroups
*/
func (this SortGroups) Copy() SortGroups {

	sortGroups := make(SortGroups, len(this))
	everyFlags := uint32(_SORT_POS_FXIED | _SORT_NULLS_FIXED | _SORT_COLLATION_FIXED)
	for i, s := range this {
		// If all flags are set no need to make copy else we need to make copy becuase it can be changed
		if s != nil && !hasEveryFlag(s.flags, everyFlags) {
			var groups SortGroups
			if len(s.groups) > 0 {
				groups = s.groups.Copy()
			}
			sortGroups[i] = &SortGroup{expr: s.expr, exprStr: s.exprStr,
				descending: s.descending, nullsPos: s.nullsPos, flags: s.flags, groups: groups, pos: s.pos,
				minPos: s.minPos, maxPos: s.maxPos}
		} else {
			sortGroups[i] = s
		}
	}

	return sortGroups
}

/* DESC Sort Window aggregates based on len(pby)+len(oby)
 * When equal use len(pby) which has flexibility
 * As number of aggregates are limited use simple bubule sort
 */
func sortWindowAggregates(aggs map[string]algebra.Aggregate) algebra.Aggregates {
	n := len(aggs)

	aggn := make(algebra.Aggregates, 0, len(aggs))
	for _, ag := range aggs {
		aggn = append(aggn, ag)
	}

	if n > 1 {
		for i := 0; i < n; i++ {
			var ipl, iol, itl int

			ipl = len(aggn[i].WindowTerm().PartitionBy())
			if aggn[i].WindowTerm().OrderBy() != nil {
				iol = len(aggn[i].WindowTerm().OrderBy().Terms())
			}
			itl = ipl + iol

			for j := i + 1; j < n; j++ {
				var jpl, jol, jtl int

				jpl = len(aggn[j].WindowTerm().PartitionBy())
				if aggn[j].WindowTerm().OrderBy() != nil {
					jol = len(aggn[j].WindowTerm().OrderBy().Terms())
				}
				jtl = jpl + jol

				if (jtl > itl) || (jtl == itl && jpl > ipl) {
					ag := aggn[i]
					aggn[i] = aggn[j]
					aggn[j] = ag
					ipl = jpl
					iol = jol
					itl = jtl
				}
			}
		}
	}

	return aggn
}
