//  Copyright 2016-Present Couchbase, Inc.
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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

/*
Algorithm for exploiting array indexes with UNNEST.

Consider only INNER UNNESTs. OUTER UNNESTs cannot exploit array
indexing.

Return a combination of UNNESTs and array indexes that works.

To consider an array index, the array key must be the first key in the
array index, and is the only key exploited for UNNEST.

To find a combination of UNNESTs and array index:

Enumerate all INNER UNNESTs in the FROM clause. Identify the primary
UNNESTs, i.e. those that unnest data in the primary term of the FROM
clause.

Enumerate all array indexes on the primary term having the array key
as their first key. If the index has an index condition, i.e. a WHERE
clause, the query predicate must be a subset of the index
condition. These are the candidate array indexes.

For each primary UNNEST:

1. Find a candidate array index. The array index key must match the
UNNEST; i.e., the array index key is an ALL (DISTINCT) ARRAY
expression whose bindings match the UNNEST's expression and alias.

2. Determine if the index satisfies the current UNNEST, or if the
index should be considered for chained UNNESTs. If the index does not
have further dimensions, i.e. the ARRAY mapping IS NOT another ALL
(DISTINCT) ARRAY expression, then attempt to satisfy the query
predicate using the index. If the index has further dimensions,
i.e. the ARRAY mapping IS another ALL (DISTINCT) ARRAY expression,
then recursively attempt to chain another UNNEST for the index's next
dimension.
*/
func (this *builder) buildUnnestIndexes(node *algebra.KeyspaceTerm, from algebra.FromTerm,
	pred expression.Expression, indexes map[datastore.Index]*indexEntry) (
	unnests, primaryUnnests []*algebra.Unnest, unnestIndexes map[datastore.Index]*indexEntry) {

	if from == nil || pred == nil || node.IsAnsiJoinOp() || this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER) {
		return
	}

	// Enumerate INNER UNNESTs
	joinTerm, ok := from.(algebra.JoinTerm)
	if !ok {
		return
	}

	// Enumerate candidate array indexes
	unnestIndexes = collectUnnestIndexes(indexes)
	if len(unnestIndexes) == 0 {
		return
	}

	// Enumerate primary UNNESTs
	unnests = _UNNEST_POOL.Get()
	unnests = collectInnerUnnestsFromJoinTerm(joinTerm, unnests)
	// Enumerate primary UNNESTs
	primaryTerm := from.PrimaryTerm()
	if this.joinEnum() {
		primaryTerm = node
	}
	primaryUnnests = collectPrimaryUnnests(primaryTerm, unnests)
	return
}

// release to the pool
func releaseUnnestPools(unnests, primaryUnnests []*algebra.Unnest) {
	if unnests != nil {
		_UNNEST_POOL.Put(unnests)
	}
	if primaryUnnests != nil {
		_UNNEST_POOL.Put(primaryUnnests)
	}
}

func (this *builder) buildUnnestScan(node *algebra.KeyspaceTerm, pred, subset, origPred expression.Expression,
	unnests, primaryUnnests []*algebra.Unnest, unnestIndexes map[datastore.Index]*indexEntry,
	hasDeltaKeyspace bool) (map[datastore.Index]*indexEntry, error) {

	baseKeyspace, ok := this.baseKeyspaces[node.Alias()]
	if !ok {
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildUnnestScan: cannot find keyspace %s", node.Alias()))
	}
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(node.Alias())),
		expression.NewFieldName("id", false))
	sargables := make(map[datastore.Index]*indexEntry, len(primaryUnnests))
	for _, unnest := range primaryUnnests {
		for index, idxEntry := range unnestIndexes {
			entry, _, _, err := this.matchUnnestScan(node, pred, subset, origPred, unnest,
				idxEntry, idxEntry.arrayKey, unnests, hasDeltaKeyspace)
			if err != nil {
				return nil, err
			}
			if entry != nil {
				entry.SetPushDownProperty(_PUSHDOWN_NONE) // reset
				err = this.getIndexFilters(entry, node, baseKeyspace, id)
				if err != nil {
					return nil, err
				}
				sargables[index] = entry
			}
		}
	}

	return sargables, nil
}

func addUnnestPreds(baseKeyspaces map[string]*base.BaseKeyspace, primary *base.BaseKeyspace) error {
	unnests := primary.GetUnnests()
	if len(unnests) == 0 {
		return nil
	}

	primaries := make(map[string]bool, len(unnests)+1)
	primaries[primary.Name()] = true
	nlen := 0

	for a, _ := range unnests {
		unnestKeyspace, ok := baseKeyspaces[a]
		if !ok {
			return errors.NewPlanInternalError(
				fmt.Sprintf("addUnnestPreds: baseKeyspace not found for %s", a))
		}
		nlen += len(unnestKeyspace.Filters())
		nlen += len(unnestKeyspace.JoinFilters())
		primaries[unnestKeyspace.Name()] = true
	}

	if nlen == 0 {
		return nil
	}

	newfilters := make(base.Filters, 0, nlen)

	for a, _ := range unnests {
		unnestKeyspace, _ := baseKeyspaces[a]
		// MB-25949, includes predicates on the unnested alias
		for _, fl := range unnestKeyspace.Filters() {
			newfltr := fl.Copy()
			newfltr.SetUnnest()
			newfilters = append(newfilters, newfltr)
		}
		// MB-28720, includes join predicates that only refer to primary term
		// MB-30292, in case of multiple levels of unnest, include join predicates
		//           that only refers to aliases in the multiple levels of unnest
		for _, jfl := range unnestKeyspace.JoinFilters() {
			if jfl.SingleJoinFilter(primaries) {
				newfltr := jfl.Copy()
				newfltr.SetUnnest()
				newfilters = append(newfilters, newfltr)
			}
		}
	}

	primary.AddFilters(newfilters)
	return nil
}

/*
Enumerate INNER UNNEST terms.
*/
func collectInnerUnnests(from algebra.FromTerm, buf []*algebra.Unnest) []*algebra.Unnest {
	joinTerm, ok := from.(algebra.JoinTerm)
	if !ok {
		return buf
	}
	return collectInnerUnnestsFromJoinTerm(joinTerm, buf)
}

func collectInnerUnnestsFromJoinTerm(joinTerm algebra.JoinTerm, buf []*algebra.Unnest) []*algebra.Unnest {
	buf = collectInnerUnnests(joinTerm.Left(), buf)

	unnest, ok := joinTerm.(*algebra.Unnest)
	if ok && !unnest.Outer() {
		buf = append(buf, unnest)
	}

	return buf
}

/*
Enumerate primary UNNESTs.
False positives are ok.
*/
func collectPrimaryUnnests(term algebra.SimpleFromTerm, unnests []*algebra.Unnest) []*algebra.Unnest {
	var buf []*algebra.Unnest
	primaryAlias := expression.NewIdentifier(term.Alias())
	for _, u := range unnests {
		// This test allows false positives, but that's ok
		if u.Expression().DependsOn(primaryAlias) {
			if nil == buf {
				buf = _UNNEST_POOL.Get()
			}
			buf = append(buf, u)
		}
	}

	return buf
}

/*
Enumerate array indexes for UNNEST.
*/
func collectUnnestIndexes(indexes map[datastore.Index]*indexEntry) map[datastore.Index]*indexEntry {

	unnestIndexes := make(map[datastore.Index]*indexEntry, len(indexes))
	for index, entry := range indexes {
		if len(entry.keys) != 0 && entry.arrayKeyPos == 0 {
			unnestIndexes[index] = entry
		}
	}

	return unnestIndexes
}

func (this *builder) matchUnnest(node *algebra.KeyspaceTerm, pred, subset expression.Expression,
	unnest *algebra.Unnest, entry *indexEntry, arrayKey *expression.All,
	unnests []*algebra.Unnest, hasDeltaKeyspace bool) (
	*indexEntry, *algebra.Unnest, *expression.All, error) {

	var sargKey, origSargKey expression.Expression
	var err error
	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())
	advisorValidate := this.advisorValidate()
	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]

	newArrayKey := arrayKey
	array, ok := arrayKey.Array().(*expression.Array)
	if ok {
		if len(array.Bindings()) != 1 {
			return nil, nil, nil, nil
		}

		binding := array.Bindings()[0]
		if !unnest.Expression().EquivalentTo(binding.Expression()) {
			return nil, nil, nil, nil
		}

		var origBinding *expression.Binding
		when := array.When()
		arrayMapping := array.ValueMapping()
		alias := expression.NewIdentifier(unnest.Alias())
		alias.SetUnnestAlias(true)

		if unnest.Alias() != binding.Variable() {
			nakey, naok := arrayMapping.(*expression.All)
			for naok {
				if a, aok := nakey.Array().(*expression.Array); aok {
					// disallow if unnest alias is nested binding variable in the array index
					if len(a.Bindings()) != 1 ||
						unnest.Alias() == a.Bindings()[0].Variable() {
						return nil, nil, nil, nil
					}
					nakey, naok = a.ValueMapping().(*expression.All)
				} else {
					naok = false
				}
			}

			origBinding = binding
			binding = expression.NewSimpleBinding(unnest.Alias(), unnest.Expression())
			renamer := expression.NewRenamer(array.Bindings(), expression.Bindings{binding})
			if when != nil {
				when, err = renamer.Map(when.Copy())
				if err != nil {
					return nil, nil, nil, nil
				}
			}
			arrayMapping, err = renamer.Map(arrayMapping.Copy())
			if err != nil {
				return nil, nil, nil, nil
			}
		}

		if when != nil && !base.SubsetOf(subset, when) {
			return nil, nil, nil, nil
		}

		nestedArrayKey, ok := arrayMapping.(*expression.All)
		if ok {
			for _, u := range unnests {
				if u == unnest ||
					!u.Expression().DependsOn(alias) {
					continue
				}

				nEntry, un, nArrayKey, err := this.matchUnnest(node, pred, subset, u, entry,
					nestedArrayKey, unnests, hasDeltaKeyspace)
				if err != nil {
					return nil, nil, nil, err
				}

				if nEntry != nil {
					newArrayKey = expression.NewAll(expression.NewArray(nArrayKey,
						expression.Bindings{binding}, when), arrayKey.Distinct())
					return nEntry, un, newArrayKey, err
				}
			}

			return nil, nil, nil, nil
		}

		sargKey = arrayMapping
		if origBinding != nil {
			if unnest.Alias() != origBinding.Variable() {
				// remember the original mapping before binding variable replacement
				origSargKey = array.ValueMapping()
			}

			newArrayKey = expression.NewAll(expression.NewArray(arrayMapping,
				expression.Bindings{binding}, when), arrayKey.Distinct())
		}
	} else if unnest.Alias() == "" || !unnest.Expression().EquivalentTo(arrayKey.Array()) {
		return nil, nil, nil, nil
	} else {
		unnestIdent := expression.NewIdentifier(unnest.Alias())
		unnestIdent.SetUnnestAlias(true)
		sargKey = unnestIdent
	}

	keys, isArrays := getUnnestIndexSargKeys(entry.idxKeys, sargKey)
	var origKeys expression.Expressions
	if origSargKey != nil {
		origKeys = getUnnestSargKeys(entry.keys, origSargKey)
	} else {
		origKeys = getUnnestSargKeys(entry.keys, sargKey)
	}

	skip := useSkipIndexKeys(entry.index, this.context.IndexApiVersion())
	missing := entry.HasFlag(IE_LEADINGMISSING)
	min, max, sum, skeys := SargableFor(pred, keys, missing, skip, isArrays, this.context, this.aliases)

	n := min
	if skip && (n > 0 || missing) {
		n = max
	}

	if n == 0 && missing {
		n = 1
	}

	if n == 0 {
		return nil, nil, nil, nil
	}

	spans, exactSpans, err := SargFor(pred, entry, keys, missing, isArrays, n, false, useCBO,
		baseKeyspace, this.keyspaceNames, advisorValidate, this.aliases, this.context)
	if err != nil {
		return nil, nil, nil, err
	}

	// ArrayKey has Descend(WITHIN) false positives possible
	if exactSpans && newArrayKey != nil && newArrayKey.HasDescend() {
		exactSpans = false
	}
	if min == 0 {
		exactSpans = false
	}
	cardinality, selectivity, cost, frCost, size :=
		OPT_CARD_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, OPT_COST_NOT_AVAIL,
		OPT_COST_NOT_AVAIL, OPT_SIZE_NOT_AVAIL
	if useCBO {
		cost, selectivity, cardinality, size, frCost, _ =
			indexScanCost(entry.index, origKeys, this.context.RequestId(),
				spans, node.Alias(), this.advisorValidate(), this.context)
		baseKeyspace.AddUnnestIndex(entry.index, unnest.Alias())
	}

	entry = newIndexEntry(entry.index, keys, n, entry.partitionKeys, min, n, sum,
		entry.cond, entry.origCond, spans, exactSpans, skeys)
	entry.setArrayKey(newArrayKey, 0)
	entry.cardinality, entry.selectivity, entry.cost, entry.frCost, entry.size =
		cardinality, selectivity, cost, frCost, size
	return entry, unnest, newArrayKey, nil
}

func (this *builder) matchUnnestScan(node *algebra.KeyspaceTerm, pred, subset, origPred expression.Expression,
	unnest *algebra.Unnest, entry *indexEntry, arrayKey *expression.All, unnests []*algebra.Unnest,
	hasDeltaKeyspace bool) (
	*indexEntry, *algebra.Unnest, *expression.All, error) {

	var err error
	arrayKey, _ = arrayKey.Copy().(*expression.All)
	entry, unnest, arrayKey, err = this.matchUnnest(node, pred, subset, unnest, entry,
		arrayKey, unnests, hasDeltaKeyspace)
	if err != nil || entry == nil {
		return entry, unnest, arrayKey, err
	}
	entry.setArrayKey(arrayKey, entry.arrayKeyPos)
	entry.unnestAliases = getUnnestAliases(entry.arrayKey, unnest)

	unnestFilters, _, _, _, err := this.coveringExpressions(node, entry, unnest,
		unnests, false)
	if err != nil {
		return entry, unnest, arrayKey, err
	}

	coverAliases := getUnnestAliases(entry.arrayKey, unnest)
	entry.pushDownProperty = this.indexPushDownProperty(entry, entry.sargKeys,
		unnestFilters, pred, origPred, node.Alias(), coverAliases, true, false,
		(len(this.baseKeyspaces) == len(entry.unnestAliases)+1), false)

	return entry, unnest, arrayKey, err
}

func getUnnestIndexSargKeys(keys datastore.IndexKeys, sargKey expression.Expression) (
	rv datastore.IndexKeys, isArrays []bool) {

	// replace the array index key with the "unnested" sargKey.
	// (assumes the array index key is the first index key)
	rv = make(datastore.IndexKeys, 0, len(keys))
	if fks, ok := sargKey.(*expression.FlattenKeys); ok {
		for i, op := range fks.Operands() {
			attr := getFlattenKeyAttributes(fks, i)
			rv = append(rv, &datastore.IndexKey{op, attr})
		}
	} else {
		rv = append(rv, &datastore.IndexKey{sargKey, datastore.IK_NONE})
	}
	isArrays = make([]bool, len(rv))
	for i := 0; i < len(rv); i++ {
		isArrays[i] = true
	}

	if len(rv) < len(keys) {
		rv = append(rv, keys[len(rv):]...)
	}

	return
}

func getUnnestSargKeys(keys expression.Expressions, sargKey expression.Expression) (rv expression.Expressions) {

	// replace the array index key with the "unnested" sargKey.
	// (assumes the array index key is the first index key)
	rv = make(expression.Expressions, 0, len(keys))
	if fks, ok := sargKey.(*expression.FlattenKeys); ok {
		rv = append(rv, fks.Operands()...)
	} else {
		rv = append(rv, sargKey)
	}

	if len(rv) < len(keys) {
		rv = append(rv, keys[len(rv):]...)
	}

	return
}

func getUnnestFilters(aliases []string) expression.Expressions {
	unnestFilters := make(expression.Expressions, 0, len(aliases))
	for _, s := range aliases {
		if s != "" {
			unnestIdent := expression.NewIdentifier(s)
			unnestIdent.SetUnnestAlias(true)
			unnestFilters = append(unnestFilters, expression.NewIsNotMissing(unnestIdent))
		}
	}
	return unnestFilters
}

/*
 Array varaibles replaced with and unnest variables.
 Collect the varaibles from the leaf (if no binding varaible replace with leaf Unnest alias)
*/

func getUnnestAliases(expr expression.Expression, leafUnnest *algebra.Unnest) (
	unnestAliases []string) {
	for all, ok := expr.(*expression.All); ok; all, ok = expr.(*expression.All) {
		if array, ok := all.Array().(*expression.Array); ok {
			expr = array.ValueMapping()
			unnestAliases = append(unnestAliases, array.Bindings()[0].Variable())
		} else {
			unnestAliases = append(unnestAliases, leafUnnest.Alias())
			break
		}
	}
	// reverse the aliases
	for i, j := 0, len(unnestAliases)-1; i < j; i, j = i+1, j-1 {
		unnestAliases[i], unnestAliases[j] = unnestAliases[j], unnestAliases[i]
	}
	return unnestAliases
}

/*
 * collect Unnest Bindings that depends on expression
 *     recursively go through dependent expression
 *     When detects OUTER JOIN it stops
 */

func (this *builder) collectUnnestBindings(from algebra.FromTerm, ua expression.Expressions,
	ub expression.Bindings) (expression.Expressions, expression.Bindings) {

	if joinTerm, ok := from.(algebra.JoinTerm); ok {
		ua, ub = this.collectUnnestBindings(joinTerm.Left(), ua, ub)
		if unnest, ok := joinTerm.(*algebra.Unnest); ok && !unnest.Outer() {
			for _, a := range ua {
				if unnest.Expression().DependsOn(a) {
					ua = append(ua, expression.NewIdentifier(unnest.Alias()))
					ub = append(ub, expression.NewSimpleBinding(unnest.Alias(),
						unnest.Expression()))
					return ua, ub
				}
			}
		}
	}

	return ua, ub
}

func chkOpUnnestIndexes(op plan.Operator, unnestIndexes map[datastore.Index]*base.UnnestIndexInfo,
	unnestIdxMap map[datastore.Index]bool) (sel float64, found bool) {
	sel = OPT_SELEC_NOT_AVAIL
	found = false
	switch op := op.(type) {
	case *plan.IndexScan3:
		index := op.Index()
		if unnestIdxMap != nil {
			if idxInfo, ok := unnestIndexes[index]; ok {
				unnestIdxMap[index] = true
				sel = idxInfo.GetSelec()
				found = true
			}
		} else {
			for idx, idxInfo := range unnestIndexes {
				if idx == index {
					sel = idxInfo.GetSelec()
					found = true
				} else {
					delete(unnestIndexes, idx)
				}
			}
		}
		return
	case *plan.DistinctScan:
		return chkOpUnnestIndexes(op.Scan(), unnestIndexes, unnestIdxMap)
	case *plan.IntersectScan:
		return chkMultiOpUnnestIndexes(op.Scans(), unnestIndexes, unnestIdxMap, false)
	case *plan.OrderedIntersectScan:
		return chkMultiOpUnnestIndexes(op.Scans(), unnestIndexes, unnestIdxMap, false)
	case *plan.UnionScan:
		return chkMultiOpUnnestIndexes(op.Scans(), unnestIndexes, unnestIdxMap, true)
	}
	return
}

func chkMultiOpUnnestIndexes(scans []plan.SecondaryScan,
	unnestIndexes map[datastore.Index]*base.UnnestIndexInfo, unnestIdxMap map[datastore.Index]bool,
	union bool) (sel float64, found bool) {

	top := false
	if unnestIdxMap == nil {
		top = true
		unnestIdxMap = make(map[datastore.Index]bool, len(scans))
	}

	sel = OPT_SELEC_NOT_AVAIL
	first := true
	for _, op := range scans {
		sc, fc := chkOpUnnestIndexes(op, unnestIndexes, unnestIdxMap)
		if !fc {
			continue
		}
		found = true
		if first {
			sel = sc
			first = false
		} else if sel > 0.0 && sc > 0.0 {
			if union {
				sel = sel + sc - (sel * sc)
			} else {
				sel = sel * sc
			}
		} else {
			sel = OPT_SELEC_NOT_AVAIL
		}
	}

	if top {
		for idx, idxInfo := range unnestIndexes {
			if _, ok := unnestIdxMap[idx]; ok {
				idxInfo.SetSelec(sel)
			} else {
				delete(unnestIndexes, idx)
			}
		}
	}

	return
}

var _UNNEST_POOL = algebra.NewUnnestPool(8)
