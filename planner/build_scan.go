//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/search"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

func (this *builder) selectScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	mutate bool) (op plan.Operator, err error) {

	keys := node.Keys()
	if keys != nil {
		this.resetPushDowns()
		switch keys.(type) {
		case *expression.ArrayConstruct, *algebra.NamedParameter, *algebra.PositionalParameter:
			this.maxParallelism = 0
		default:
			this.maxParallelism = 1
		}

		cost := OPT_COST_NOT_AVAIL
		cardinality := OPT_CARD_NOT_AVAIL
		if this.useCBO {
			cost, cardinality = getKeyScanCost(keys)
		}
		return plan.NewKeyScan(keys, mutate, cost, cardinality), nil
	}

	secondary, primary, err := this.buildScan(keyspace, node)
	if err != nil {
		return nil, err
	}

	if secondary != nil {
		return secondary, nil
	} else if primary != nil {
		return primary, nil
	} else {
		return nil, nil
	}
}

func (this *builder) buildScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm) (
	secondary plan.Operator, primary plan.Operator, err error) {

	join := node.IsAnsiJoinOp()
	hash := node.IsUnderHash()

	var hints []datastore.Index
	if len(node.Indexes()) > 0 || this.context.UseFts() {
		hints = _HINT_POOL.Get()
		defer _HINT_POOL.Put(hints)
		hints, err = allHints(keyspace, node.Indexes(), hints, this.context.IndexApiVersion(), this.context.UseFts())
		if err != nil {
			return
		}
	}

	baseKeyspace, ok := this.baseKeyspaces[node.Alias()]
	if !ok {
		return nil, nil, errors.NewPlanInternalError(fmt.Sprintf("buildScan: cannot find keyspace %s", node.Alias()))
	}

	hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())
	if hasDeltaKeyspace {
		this.resetPushDowns()
	}

	var pred, pred2 expression.Expression
	if join {
		pred = baseKeyspace.DnfPred()
	} else {
		pred = this.where
		pred2 = this.pushableOnclause

		if this.falseWhereClause() {
			return _EMPTY_PLAN, nil, nil
		}
		if this.trueWhereClause() {
			pred = nil
		}
	}

	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(node.Alias())),
		expression.NewFieldName("id", false))

	if pred != nil || pred2 != nil {
		// for ANSI JOIN, the following process is already done for ON clause filters
		if !join {
			if len(baseKeyspace.JoinFilters()) > 0 {
				// derive IS NOT NULL predicate
				err = deriveNotNullFilter(keyspace, baseKeyspace, this.useCBO,
					this.context.IndexApiVersion(), this.getIdxCandidates(), this.advisorValidate())
				if err != nil {
					return nil, nil, err
				}
			}

			// add predicates from UNNEST keyspaces
			addUnnestPreds(this.baseKeyspaces, baseKeyspace)

			// include pushed ON-clause filter
			err = CombineFilters(baseKeyspace, true, false)
			if err != nil {
				return nil, nil, err
			}

		}

		this.enableUnnest(node.Alias())
		this.collectPredicates(baseKeyspace, keyspace, node, nil, false)

		if baseKeyspace.DnfPred() != nil {
			if baseKeyspace.OrigPred() == nil {
				return nil, nil, errors.NewPlanInternalError("buildScan: NULL origPred")
			}
			return this.buildPredicateScan(keyspace, node, baseKeyspace, id, hints)
		}
	}

	if join && !hash {
		op := "join"
		if node.IsAnsiNest() {
			op = "nest"
		}
		return nil, nil, errors.NewNoAnsiJoinError(node.Alias(), op)
	} else if this.cover != nil && baseKeyspace.DnfPred() == nil {
		// Handle covering primary scan
		scan, err := this.buildCoveringPrimaryScan(keyspace, node, id, hints)
		if scan != nil || err != nil {
			return scan, nil, err
		}
	}

	primary, err = this.buildPrimaryScan(keyspace, node, hints, id, false, true, hasDeltaKeyspace)
	return nil, primary, err
}

func (this *builder) buildPredicateScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, id expression.Expression, hints []datastore.Index) (
	secondary plan.Operator, primary plan.Operator, err error) {

	// Handle constant FALSE predicate
	cpred := baseKeyspace.OrigPred().Value()
	if cpred != nil && !cpred.Truth() {
		return _EMPTY_PLAN, nil, nil
	}

	// do not consider primary index for ANSI JOIN or ANSI NEST
	var primaryKey expression.Expressions
	if !node.IsAnsiJoinOp() || node.IsUnderHash() {
		primaryKey = expression.Expressions{id}
	}

	formalizer := expression.NewSelfFormalizer(node.Alias(), nil)

	if len(hints) > 0 {
		// Set processing HINT Indexes
		this.hintIndexes = true
		secondary, primary, err = this.buildSubsetScan(
			keyspace, node, baseKeyspace, id, hints, primaryKey, formalizer, true)
		this.hintIndexes = false
		if secondary != nil || primary != nil || err != nil {
			return
		}
	}

	// collect SEARCH() functions that depends on current keyspace in the predicate
	var searchFns map[string]*search.Search
	if !node.IsUnderNL() {
		pred := baseKeyspace.DnfPred()
		if node.IsAnsiJoinOp() && baseKeyspace.OnclauseOnly() {
			pred = baseKeyspace.Onclause()
		}

		searchFns = make(map[string]*search.Search)

		if err = collectFTSSearch(node.Alias(), searchFns, pred); err != nil {
			return
		}
	}

	others := _INDEX_POOL.Get()
	defer _INDEX_POOL.Put(others)
	others, err = allIndexes(keyspace, hints, others, this.context.IndexApiVersion(), len(searchFns) > 0)
	if err != nil {
		return
	}

	if this.indexAdvisor {
		others = this.addVirtualIndexes(others)
	}

	secondary, primary, err = this.buildSubsetScan(keyspace, node,
		baseKeyspace, id, others, primaryKey, formalizer, false)

	if secondary != nil || primary != nil || err != nil {
		return
	}

	if node.IsAnsiJoinOp() {
		if node.IsPrimaryJoin() || node.IsUnderHash() {
			return nil, nil, nil
		} else {
			op := "join"
			if node.IsAnsiNest() {
				op = "nest"
			}
			return nil, nil, errors.NewNoAnsiJoinError(node.Alias(), op)
		}
	} else {
		return nil, nil, errors.NewPlanInternalError(fmt.Sprintf("buildPredicateScan: No plan generated for %s", node.Alias()))
	}
}

func (this *builder) buildSubsetScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, id expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer, force bool) (
	secondary plan.Operator, primary plan.Operator, err error) {

	join := node.IsAnsiJoinOp()
	hash := node.IsUnderHash()
	if join {
		this.resetPushDowns()
	}
	order := this.order

	// Prefer OR scan
	pred := baseKeyspace.DnfPred()
	if join && baseKeyspace.OnclauseOnly() {
		pred = baseKeyspace.Onclause()
	}
	if or, ok := pred.(*expression.Or); ok {

		scan, _, err := this.buildOrScan(node, baseKeyspace, id, or, indexes, primaryKey, formalizer)

		if scan != nil || err != nil {
			return scan, nil, err
		}
	}

	// Prefer secondary scan
	secondary, _, err = this.buildTermScan(node, baseKeyspace, id, indexes, primaryKey, formalizer)
	if secondary != nil || err != nil {
		return secondary, nil, err
	}

	if !join || hash {
		// No secondary scan, try primary scan. restore order there is predicate no need to restore others
		this.order = order
		hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())
		primary, err = this.buildPrimaryScan(keyspace, node, indexes, id, force, false, hasDeltaKeyspace)
		if err != nil {
			return nil, nil, err
		}
	} else {
		primary = nil
	}

	return nil, primary, nil
}

func (this *builder) buildTermScan(node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, id expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	secondary plan.SecondaryScan, sargLength int, err error) {

	join := node.IsAnsiJoinOp()

	var scanbuf [4]plan.SecondaryScan
	scans := scanbuf[0:1]

	if !join {
		// Consider pattern matching indexes
		err = this.PatternFor(baseKeyspace, indexes, formalizer)
		if err != nil {
			return nil, 0, err
		}
	}

	pred := baseKeyspace.DnfPred()
	if join && baseKeyspace.OnclauseOnly() {
		pred = baseKeyspace.Onclause()
	}

	// collect UNNEST bindings when HINT indexes has FTS index
	var ubs expression.Bindings
	if this.hintIndexes && this.from != nil {
		for _, idx := range indexes {
			if idx.Type() == datastore.FTS {
				ubs = make(expression.Bindings, 0, 2)
				ua := expression.Expressions{expression.NewIdentifier(node.Alias())}
				_, ubs = this.collectUnnestBindings(this.from, ua, ubs)
				break
			}
		}
	}

	sargables, all, arrays, flex, err := this.sargableIndexes(indexes, pred, pred, primaryKey,
		formalizer, ubs, node.IsUnderNL())
	if err != nil {
		return nil, 0, err
	}

	// purge any subset indexe and keep superset indexes
	minimals := this.minimalIndexes(sargables, false, pred, node)
	flex = this.minimalFTSFlexIndexes(flex, false)

	// pred has SEARCH() function get sargable FTS indexes
	var searchSargables []*indexEntry
	var searchFns map[string]*search.Search
	if !node.IsUnderNL() {
		searchFns = make(map[string]*search.Search)
		if err = collectFTSSearch(node.Alias(), searchFns, pred); err != nil {
			return nil, 0, err
		}

		searchSargables, err = this.sargableSearchIndexes(indexes, pred, searchFns, formalizer)
		if err != nil {
			return nil, 0, err
		}
	}

	indexPushDowns := this.storeIndexPushDowns()

	defer func() {
		if this.orderScan != nil {
			this.order = indexPushDowns.order
		}
	}()

	var secOffsetPushed, unnestOffsetPushed, dynamicOffsetPushed bool
	var limitPushed bool

	// Try secondary scan
	if len(minimals) > 0 || len(searchSargables) > 0 || len(flex) > 0 {
		if len(this.baseKeyspaces) > 1 {
			this.resetOffsetLimit()
			this.resetProjection()
			this.resetIndexGroupAggs()
		}

		secondary, sargLength, err = this.buildSecondaryScan(minimals, flex, node, baseKeyspace,
			id, searchSargables)
		if err != nil {
			return nil, 0, err
		}

		if secondary != nil {
			if len(this.coveringScans) > 0 || this.countScan != nil {
				return secondary, sargLength, nil
			}

			if secondary == this.orderScan {
				scans[0] = secondary
			} else {
				scans = append(scans, secondary)
			}

			secOffsetPushed = this.offset != nil
			limitPushed = limitPushed || this.limit != nil
		}
	}

	// Try UNNEST scan
	if !join && this.from != nil {
		// Try pushdowns
		this.restoreIndexPushDowns(indexPushDowns, true)
		hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())

		unnest, unnestSargLength, err := this.buildUnnestScan(node, this.from, pred, all, hasDeltaKeyspace)
		if err != nil {
			return nil, 0, err
		}

		if unnest != nil {
			if len(this.coveringScans) > 0 || this.countScan != nil {
				return unnest, unnestSargLength, err
			}

			scans = append(scans, unnest)
			if sargLength < unnestSargLength {
				sargLength = unnestSargLength
			}

			unnestOffsetPushed = this.offset != nil
			limitPushed = limitPushed || this.limit != nil
		}

		this.resetPushDowns()
	}

	// Try dynamic scan
	if !join && len(arrays) > 0 {
		// Try pushdowns
		if indexPushDowns.order == nil || this.orderScan != nil {
			this.limit = indexPushDowns.limit
			this.offset = indexPushDowns.offset
		}

		dynamicPred := baseKeyspace.OrigPred().Copy()
		dnf := base.NewDNF(dynamicPred, false, true)
		dynamicPred, err = dnf.Map(dynamicPred)
		if err != nil {
			return nil, 0, err
		}

		dynamic, dynamicSargLength, err :=
			this.buildDynamicScan(node, id, dynamicPred, arrays, primaryKey, formalizer)
		if err != nil {
			return nil, 0, err
		}

		if dynamic != nil {
			if len(this.coveringScans) > 0 || this.countScan != nil {
				return dynamic, dynamicSargLength, err
			}

			scans = append(scans, dynamic)
			if sargLength < dynamicSargLength {
				sargLength = dynamicSargLength
			}
			dynamicOffsetPushed = this.offset != nil
			limitPushed = limitPushed || this.limit != nil
		}
	}

	switch len(scans) {
	case 0:
		this.limit = indexPushDowns.limit
		this.offset = indexPushDowns.offset
		secondary = nil
	case 1:
		this.resetOffset()
		if secOffsetPushed || unnestOffsetPushed || dynamicOffsetPushed {
			this.offset = indexPushDowns.offset
		}
		secondary = scans[0]
	default:
		this.resetOffset()
		var limit expression.Expression

		if limitPushed {
			limit = offsetPlusLimit(indexPushDowns.offset, indexPushDowns.limit)
		}

		if scans[0] == nil {
			if len(scans) == 2 {
				if secOffsetPushed || unnestOffsetPushed || dynamicOffsetPushed {
					this.offset = indexPushDowns.offset
				}
				secondary = scans[1]
			} else {
				cost, cardinality := this.intersectScanCost(node, scans[1:]...)
				secondary = plan.NewIntersectScan(limit, cost, cardinality, scans[1:]...)
			}
		} else {
			if ordered, ok := scans[0].(*plan.OrderedIntersectScan); ok {
				scans = append(ordered.Scans(), scans[1:]...)
			}

			cost, cardinality := this.intersectScanCost(node, scans...)
			secondary = plan.NewOrderedIntersectScan(nil, cost, cardinality, scans...)
		}
	}

	// Return secondary scan, if any
	return secondary, sargLength, nil
}

func (this *builder) processPredicate(pred expression.Expression, isOnclause bool) (
	constant value.Value, err error) {

	pred = pred.Copy()

	for name, value := range this.context.NamedArgs() {
		nameExpr := algebra.NewNamedParameter(name)
		valueExpr := expression.NewConstant(value)
		pred, err = expression.ReplaceExpr(pred, nameExpr, valueExpr)
		if err != nil {
			return
		}
	}

	for pos, value := range this.context.PositionalArgs() {
		posExpr := algebra.NewPositionalParameter(pos + 1)
		valueExpr := expression.NewConstant(value)
		pred, err = expression.ReplaceExpr(pred, posExpr, valueExpr)
		if err != nil {
			return
		}
	}

	constant, err = ClassifyExpr(pred, this.baseKeyspaces, this.keyspaceNames, isOnclause, this.useCBO, this.advisorValidate())
	return
}

func (this *builder) processWhere(where expression.Expression) (err error) {
	var constant value.Value
	constant, err = this.processPredicate(where, false)
	if err != nil {
		return
	}
	// Handle constant TRUE/FALSE predicate
	if constant != nil {
		if constant.Truth() {
			this.setTrueWhereClause()
		} else {
			this.setFalseWhereClause()
		}
	}

	return
}

func (this *builder) intersectScanCost(node *algebra.KeyspaceTerm, scans ...plan.SecondaryScan) (float64, float64) {
	docCount, err := this.getDocCount(node)
	if err != nil {
		return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
	}

	useCBO := this.useCBO
	if !useCBO {
		return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
	}

	cost := float64(0.0)
	cardinality := float64(0.0)
	selec := float64(1.0)
	for _, scan := range scans {
		scost := scan.Cost()
		scardinality := scan.Cardinality()
		if (scost <= 0.0) || (scardinality <= 0.0) {
			useCBO = false
			break
		}

		cost += scost
		selec1 := scardinality / docCount
		if selec1 > 1.0 {
			selec1 = 1.0
		}
		selec = selec * selec1
	}

	if useCBO {
		// cost calculated in for loop above
		cardinality = selec * docCount
	} else {
		cost = OPT_COST_NOT_AVAIL
		cardinality = OPT_CARD_NOT_AVAIL
	}

	return cost, cardinality
}

// helper function check online indexes
func isValidIndex(idx datastore.Index, indexApiVersion int) bool {
	state, _, err := idx.State()
	if err != nil {
		logging.Errorp("Index selection", logging.Pair{"error", err.Error()})
		return false
	}

	if idx.Type() == datastore.FTS {
		return state == datastore.ONLINE
	}

	return (state == datastore.ONLINE) && (useIndex2API(idx, indexApiVersion) || !indexHasDesc(idx))
}

// all HINT indexes
func allHints(keyspace datastore.Keyspace, hints algebra.IndexRefs, indexes []datastore.Index, indexApiVersion int, useFts bool) (
	[]datastore.Index, error) {

	// check if HINT has FTS index refrence
	var hintFts bool

	for _, hint := range hints {
		if hint.Using() == datastore.FTS {
			hintFts = true
			break
		}
	}

	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	for _, indexer := range indexers {
		// neither FTS index reference in the HINT nor useFts set skip FTS indexer
		if !hintFts && !useFts && indexer.Name() == datastore.FTS {
			continue
		}

		idxes, err := indexer.Indexes()
		if err != nil {
			return nil, err
		}

		// all HINT indexes. If name is "", consider all indexes on the indexer
		// duplicates on the HINT will be ignored
		for _, idx := range idxes {
			/* When one or more FTS indexes is specified in the USE INDEX hint,
			   USE_FTS query parameter does not take effect. When no FTS indexes is specified in the
			   USE INDEX hint (or no hint specified), USE_FTS query parameter takes effect.
			*/
			if !hintFts && useFts && indexer.Name() == datastore.FTS && isValidIndex(idx, indexApiVersion) {
				indexes = append(indexes, idx)
				continue
			}

			for _, hint := range hints {
				using := hint.Using()
				if using == datastore.DEFAULT {
					using = datastore.GSI
				}
				if indexer.Name() == using &&
					(hint.Name() == "" || hint.Name() == idx.Name()) {
					if isValidIndex(idx, indexApiVersion) {
						indexes = append(indexes, idx)
					}
					break
				}
			}
		}
	}

	return indexes, nil
}

/*
all the indexes excluding HINT indexes.
inclFts indicates to include FTS index or not
        * true  - SEARCH() function is present
        * false - right side of some JOINs, no SERACH() function
*/

func allIndexes(keyspace datastore.Keyspace, skip, indexes []datastore.Index, indexApiVersion int, inclFts bool) (
	[]datastore.Index, error) {

	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	var skipMap map[datastore.Index]bool
	if len(skip) > 0 {
		skipMap = _SKIP_POOL.Get()
		defer _SKIP_POOL.Put(skipMap)
		for _, s := range skip {
			skipMap[s] = true
		}
	}

	for _, indexer := range indexers {
		// no FTS indexes needed and  indexer is FTS skip the indexer
		if !inclFts && indexer.Name() == datastore.FTS {
			continue
		}

		idxes, err := indexer.Indexes()
		if err != nil {
			return nil, err
		}

		for _, idx := range idxes {
			// Skip index if listed
			if len(skipMap) > 0 && skipMap[idx] {
				continue
			}

			if isValidIndex(idx, indexApiVersion) {
				indexes = append(indexes, idx)
			}

		}
	}

	return indexes, nil
}

var _INDEX_POOL = datastore.NewIndexPool(256)
var _HINT_POOL = datastore.NewIndexPool(32)
var _SKIP_POOL = datastore.NewIndexBoolPool(32)
var _EMPTY_PLAN = plan.NewValueScan(algebra.Pairs{}, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL)
