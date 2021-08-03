//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type PushDownProperties uint32

const (
	_PUSHDOWN_NONE     PushDownProperties = iota
	_PUSHDOWN_DISTINCT PushDownProperties = 1 << iota
	_PUSHDOWN_EXACTSPANS
	_PUSHDOWN_LIMIT
	_PUSHDOWN_OFFSET
	_PUSHDOWN_ORDER
	_PUSHDOWN_GROUPAGGS
	_PUSHDOWN_FULLGROUPAGGS
)

type indexEntry struct {
	index            datastore.Index
	keys             expression.Expressions
	sargKeys         expression.Expressions
	partitionKeys    expression.Expressions
	minKeys          int
	maxKeys          int
	sumKeys          int
	nSargKeys        int
	skeys            []bool
	cond             expression.Expression
	origCond         expression.Expression
	spans            SargSpans
	exactSpans       bool
	pushDownProperty PushDownProperties
	cost             float64
	cardinality      float64
	selectivity      float64
	size             int64
	frCost           float64
	searchOrders     []string
	condFc           map[string]value.Value
	nEqCond          int
	numIndexedKeys   uint32
}

func newIndexEntry(index datastore.Index, keys, sargKeys, partitionKeys expression.Expressions,
	minKeys, maxKeys, sumKeys int, cond, origCond expression.Expression, spans SargSpans,
	exactSpans bool, skeys []bool) *indexEntry {
	rv := &indexEntry{
		index:            index,
		keys:             keys,
		sargKeys:         sargKeys,
		partitionKeys:    partitionKeys,
		minKeys:          minKeys,
		maxKeys:          maxKeys,
		sumKeys:          sumKeys,
		skeys:            skeys,
		cond:             cond,
		origCond:         origCond,
		spans:            spans,
		exactSpans:       exactSpans,
		pushDownProperty: _PUSHDOWN_NONE,
		cost:             OPT_COST_NOT_AVAIL,
		cardinality:      OPT_CARD_NOT_AVAIL,
		selectivity:      OPT_SELEC_NOT_AVAIL,
		size:             OPT_SIZE_NOT_AVAIL,
		frCost:           OPT_COST_NOT_AVAIL,
	}

	for _, b := range skeys {
		if b {
			rv.nSargKeys++
		}
	}

	if rv.cond != nil {
		fc := make(map[string]value.Value, 4)
		rv.condFc = rv.cond.FilterCovers(fc)
		rv.nEqCond = countEqCond(rv.cond, rv.sargKeys, rv.skeys)
	}
	return rv
}

func (this *indexEntry) Copy() *indexEntry {
	rv := &indexEntry{
		index:            this.index,
		keys:             expression.CopyExpressions(this.keys),
		sargKeys:         expression.CopyExpressions(this.sargKeys),
		partitionKeys:    expression.CopyExpressions(this.partitionKeys),
		minKeys:          this.minKeys,
		maxKeys:          this.maxKeys,
		sumKeys:          this.sumKeys,
		nSargKeys:        this.nSargKeys,
		cond:             expression.Copy(this.cond),
		origCond:         expression.Copy(this.origCond),
		spans:            CopySpans(this.spans),
		exactSpans:       this.exactSpans,
		pushDownProperty: this.pushDownProperty,
		cost:             this.cost,
		cardinality:      this.cardinality,
		selectivity:      this.selectivity,
		size:             this.size,
		frCost:           this.frCost,
		condFc:           this.condFc,
		nEqCond:          this.nEqCond,
	}

	copy(rv.searchOrders, this.searchOrders)
	if len(this.skeys) > 0 {
		copy(rv.skeys, this.skeys)
	}

	return rv
}

func (this *indexEntry) PushDownProperty() PushDownProperties {
	return this.pushDownProperty
}

func (this *indexEntry) IsPushDownProperty(property PushDownProperties) bool {
	return isPushDownProperty(this.pushDownProperty, property)
}

func (this *indexEntry) setSearchOrders(so []string) {
	this.searchOrders = so
}

func isPushDownProperty(pushDownProperty, property PushDownProperties) bool {
	if property == _PUSHDOWN_NONE {
		return (pushDownProperty == property)
	}
	return (pushDownProperty & property) != 0
}

type EqExpr struct {
	expression.MapperBase
	sargKyes expression.Expressions
	skeys    []bool
	ncount   int
}

// Number Equality predicate in index Condition that not part of index keys

func countEqCond(cond expression.Expression, sargKyes expression.Expressions, skeys []bool) int {
	rv := &EqExpr{sargKyes: sargKyes, skeys: skeys}
	rv.SetMapFunc(func(expr expression.Expression) (expression.Expression, error) {
		if e, ok := expr.(*expression.Eq); ok {
			for i, sk := range rv.sargKyes {
				if rv.skeys[i] &&
					(expression.Equivalent(sk, e.First()) ||
						expression.Equivalent(sk, e.Second())) {
					return expr, nil
				}
			}
			rv.ncount++
			return expr, nil
		}
		return expr, expr.MapChildren(rv)
	})

	rv.SetMapper(rv)

	if _, err := rv.Map(cond); err != nil {
		return 0
	}
	return rv.ncount
}
