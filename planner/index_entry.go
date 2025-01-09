//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

type PushDownProperties uint32

const (
	_PUSHDOWN_NONE     PushDownProperties = iota
	_PUSHDOWN_DISTINCT PushDownProperties = 1 << iota
	_PUSHDOWN_EXACTSPANS
	_PUSHDOWN_LIMIT
	_PUSHDOWN_OFFSET
	_PUSHDOWN_COVERED_UNNEST
	_PUSHDOWN_PARTIAL_ORDER
	_PUSHDOWN_ORDER
	_PUSHDOWN_GROUPAGGS
	_PUSHDOWN_FULLGROUPAGGS
)

const (
	IE_NONE           = 0
	IE_LEADINGMISSING = 1 << iota
	IE_ARRAYINDEXKEY
	IE_ARRAYINDEXKEY_SARGABLE
	IE_HAS_FILTER
	IE_HAS_JOIN_FILTER
	IE_HAS_EARLY_ORDER
	IE_OR_USE_FILTERS
	IE_OR_NON_SARG_EXPR
	IE_LIMIT_OFFSET_COST
	IE_SEARCH_KNN
	IE_VECTOR_KEY_SARGABLE
	IE_VECTOR_KEY_SKIP_ORDER
	IE_NONEQ_COND
	IE_VECTOR_RERANK
)

type indexEntry struct {
	index                datastore.Index
	idxKeys              datastore.IndexKeys
	idxSargKeys          datastore.IndexKeys
	keys                 expression.Expressions
	sargKeys             expression.Expressions
	partitionKeys        expression.Expressions
	includes             expression.Expressions
	arrayKey             *expression.All
	arrayKeyPos          int
	minKeys              int
	maxKeys              int
	sumKeys              int
	includeKeys          int
	nSargKeys            int
	skeys                []bool
	cond                 expression.Expression
	origCond             expression.Expression
	spans                SargSpans
	exactSpans           bool
	pushDownProperty     PushDownProperties
	cost                 float64
	cardinality          float64
	selectivity          float64
	size                 int64
	frCost               float64
	fetchCost            float64
	searchOrders         []string
	condFc               map[string]value.Value
	nEqCond              int
	nCondKeys            int
	numIndexedKeys       uint32
	flags                uint32
	unnestAliases        []string
	exactFilters         map[*base.Filter]bool
	indexFilters         expression.Expressions
	orderExprs           expression.Expressions
	partialSortTermCount int
}

func newIndexEntry(index datastore.Index, idxKeys datastore.IndexKeys, includes expression.Expressions,
	sargLength int, partitionKeys expression.Expressions, minKeys, maxKeys, sumKeys, includeKeys int,
	cond, origCond expression.Expression, spans SargSpans, exactSpans bool, skeys []bool) *indexEntry {
	rv := &indexEntry{
		index:            index,
		idxKeys:          idxKeys,
		includes:         includes,
		partitionKeys:    partitionKeys,
		minKeys:          minKeys,
		maxKeys:          maxKeys,
		sumKeys:          sumKeys,
		includeKeys:      includeKeys,
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
		fetchCost:        OPT_COST_NOT_AVAIL,
		flags:            IE_NONE,
	}

	rv.keys = make(expression.Expressions, 0, len(idxKeys))
	for _, key := range idxKeys {
		rv.keys = append(rv.keys, key.Expr)
	}

	if sargLength > len(idxKeys) || sargLength < 0 {
		sargLength = len(idxKeys)
	}
	rv.idxSargKeys = rv.idxKeys[0:sargLength]
	rv.sargKeys = rv.keys[0:sargLength]

	rv.arrayKeyPos = -1
	for _, b := range skeys {
		if b {
			rv.nSargKeys++
		}
	}

	if rv.cond != nil {
		var other int
		fc := make(map[string]value.Value, 4)
		cond := rv.cond
		if rv.origCond != nil {
			cond = rv.origCond
		}
		rv.condFc = cond.FilterCovers(fc)
		rv.nEqCond, rv.nCondKeys, other = countEqCond(cond, rv.sargKeys, rv.skeys)
		if other > 0 {
			rv.flags |= IE_NONEQ_COND
		}
	}
	return rv
}

func (this *indexEntry) Copy() *indexEntry {
	rv := &indexEntry{
		index:            this.index,
		idxKeys:          this.idxKeys.Copy(),
		keys:             expression.CopyExpressions(this.keys),
		includes:         expression.CopyExpressions(this.includes),
		partitionKeys:    expression.CopyExpressions(this.partitionKeys),
		arrayKeyPos:      this.arrayKeyPos,
		minKeys:          this.minKeys,
		maxKeys:          this.maxKeys,
		sumKeys:          this.sumKeys,
		includeKeys:      this.includeKeys,
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
		fetchCost:        this.fetchCost,
		condFc:           this.condFc,
		nEqCond:          this.nEqCond,
		nCondKeys:        this.nCondKeys,
		flags:            this.flags,
	}
	rv.idxSargKeys = rv.idxKeys[0:len(this.idxSargKeys)]
	rv.sargKeys = rv.keys[0:len(this.sargKeys)]
	if this.arrayKey != nil {
		rv.arrayKey, _ = expression.Copy(this.arrayKey).(*expression.All)
	}
	rv.searchOrders = make([]string, len(this.searchOrders))
	copy(rv.searchOrders, this.searchOrders)
	rv.unnestAliases = make([]string, len(this.unnestAliases))
	copy(rv.unnestAliases, this.unnestAliases)
	if len(this.skeys) > 0 {
		rv.skeys = make([]bool, len(this.skeys))
		copy(rv.skeys, this.skeys)
	}
	if len(this.exactFilters) > 0 {
		rv.exactFilters = make(map[*base.Filter]bool, len(this.exactFilters))
		for k, v := range this.exactFilters {
			rv.exactFilters[k] = v
		}
	}

	return rv
}

func (this *indexEntry) Flags() uint32 {
	return this.flags
}

func (this *indexEntry) SetFlags(flags uint32, add bool) {
	if add {
		this.flags |= flags
	} else {
		this.flags = flags
	}
}

func (this *indexEntry) UnsetFlags(flags uint32) {
	this.flags &^= flags
}

func (this *indexEntry) HasFlag(flag uint32) bool {
	return (this.flags & flag) != 0
}

// return flags relevant for index key values (join filter, early order)
func (this *indexEntry) IndexKeyFlags() uint32 {
	return (this.flags & (IE_HAS_JOIN_FILTER | IE_HAS_EARLY_ORDER))
}

func (this *indexEntry) PushDownProperty() PushDownProperties {
	return this.pushDownProperty
}

func (this *indexEntry) SetPushDownProperty(property PushDownProperties) {
	this.pushDownProperty = property
}

func (this *indexEntry) IsPushDownProperty(property PushDownProperties) bool {
	return isPushDownProperty(this.pushDownProperty, property)
}

func (this *indexEntry) setSearchOrders(so []string) {
	this.searchOrders = so
}

func (this *indexEntry) setArrayKey(key *expression.All, pos int) {
	this.arrayKey = key
	this.arrayKeyPos = pos
	if key != nil {
		this.flags |= IE_ARRAYINDEXKEY
		if pos >= 0 {
			size := 1
			if key.Flatten() {
				size = key.FlattenSize()
			}
			for i := pos; i < pos+size; i++ {
				if i < len(this.skeys) && this.skeys[i] {
					this.flags |= IE_ARRAYINDEXKEY_SARGABLE
				}
			}
		}
	}
}

// for comparing indexes, use both index scan cost and fetch cost
func (this *indexEntry) scanCost() float64 {
	if this.cost > 0.0 && this.fetchCost > 0.0 {
		return this.cost + this.fetchCost
	}
	return this.cost
}

func isPushDownProperty(pushDownProperty, property PushDownProperties) bool {
	if property == _PUSHDOWN_NONE {
		return (pushDownProperty == property)
	}
	return (pushDownProperty & property) != 0
}

func comparePushDownProperties(prop1, prop2 PushDownProperties) (better, similar bool) {
	ga1 := prop1&(_PUSHDOWN_FULLGROUPAGGS|_PUSHDOWN_GROUPAGGS) != 0
	ga2 := prop2&(_PUSHDOWN_FULLGROUPAGGS|_PUSHDOWN_GROUPAGGS) != 0
	if ga1 != ga2 {
		return ga1, false
	}

	prop1 &^= (_PUSHDOWN_FULLGROUPAGGS | _PUSHDOWN_GROUPAGGS)
	prop2 &^= (_PUSHDOWN_FULLGROUPAGGS | _PUSHDOWN_GROUPAGGS)

	o1 := prop1&(_PUSHDOWN_ORDER|_PUSHDOWN_PARTIAL_ORDER) != 0
	o2 := prop2&(_PUSHDOWN_ORDER|_PUSHDOWN_PARTIAL_ORDER) != 0
	if o1 != o2 {
		return o1, false
	}

	prop1 &^= (_PUSHDOWN_ORDER | _PUSHDOWN_PARTIAL_ORDER)
	prop2 &^= (_PUSHDOWN_ORDER | _PUSHDOWN_PARTIAL_ORDER)

	return prop1 > prop2, prop1 == prop2
}

type EqExpr struct {
	expression.MapperBase
	sargKyes expression.Expressions
	skeys    []bool
	nCond    int
	nKeys    int
	nOther   int
}

// Number Equality predicate in index Condition that not part of index keys

func countEqCond(cond expression.Expression, sargKyes expression.Expressions, skeys []bool) (int, int, int) {
	rv := &EqExpr{sargKyes: sargKyes, skeys: skeys}
	rv.SetMapFunc(func(expr expression.Expression) (expression.Expression, error) {
		switch e := expr.(type) {
		case *expression.Eq:
			for i, sk := range rv.sargKyes {
				if rv.skeys[i] &&
					(expression.Equivalent(sk, e.First()) ||
						expression.Equivalent(sk, e.Second())) {
					rv.nKeys++
					return expr, nil
				}
			}
			rv.nCond++
			return expr, nil
		case *expression.In:
			for i, sk := range rv.sargKyes {
				if rv.skeys[i] && expression.Equivalent(sk, e.First()) {
					rv.nKeys++
					return expr, nil
				}
			}
			return expr, nil
		case *expression.And:
			return expr, expr.MapChildren(rv)
		default:
			rv.nOther++
			return expr, nil
		}
	})

	rv.SetMapper(rv)

	if _, err := rv.Map(cond); err != nil {
		return 0, 0, 0
	}
	return rv.nCond, rv.nKeys, rv.nOther
}
