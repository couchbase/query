//  Copyright 2014-Present Couchbase, Inc.
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
	"github.com/couchbase/query/expression/search"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

func (this *builder) buildSecondaryScan(indexes, arrayIndexes, flex map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace, subset, id expression.Expression,
	searchSargables []*indexEntry) (scan plan.SecondaryScan, sargLength int, err error) {

	pred := baseKeyspace.DnfPred()
	unnests, primaryUnnests, unnestIndexes := this.buildUnnestIndexes(node, this.from,
		pred, arrayIndexes)
	defer releaseUnnestPools(unnests, primaryUnnests)

	scan, sargLength, err = this.buildCovering(indexes, unnestIndexes, flex, node,
		baseKeyspace, subset, id, searchSargables, unnests)
	if scan != nil || err != nil {
		return
	}

	hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())

	if this.group != nil || hasDeltaKeyspace {
		this.resetPushDowns()
	}

	if !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) {
		err = this.sargIndexes(baseKeyspace, node.IsUnderHash(), indexes)
		if err != nil {
			return nil, 0, err
		}
	}

	for _, entry := range indexes {
		entry.pushDownProperty = this.indexPushDownProperty(entry, entry.keys, nil,
			pred, node.Alias(), nil, false, false, (len(this.baseKeyspaces) == 1),
			implicitAnyCover(entry, true, this.context.FeatureControls()))
	}

	if len(primaryUnnests) > 0 && len(unnests) > 0 && len(unnestIndexes) > 0 {
		var unnestSargables map[datastore.Index]*indexEntry
		unnestSargables, err = this.buildUnnestScan(node, pred, subset, unnests,
			primaryUnnests, unnestIndexes, hasDeltaKeyspace)
		if err != nil {
			return nil, 0, err
		}
		// add to regular Secondary IndexScan list
		for index, entry := range unnestSargables {
			indexes[index] = entry
		}
	}

	return this.buildCreateSecondaryScan(indexes, flex, node, baseKeyspace,
		pred, id, searchSargables, hasDeltaKeyspace)
}

func (this *builder) buildCreateSecondaryScan(indexes, flex map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace, pred, id expression.Expression,
	searchSargables []*indexEntry, hasDeltaKeyspace bool) (scan plan.SecondaryScan, sargLength int, err error) {

	indexes = this.minimalIndexes(indexes, true, pred, node)
	// Already done. need only for one index
	// flex = this.minimalFTSFlexIndexes(flex, true)
	searchSargables = this.minimalSearchIndexes(flex, searchSargables)

	orderEntry, limit, searchOrders := this.buildSecondaryScanPushdowns(indexes, flex, searchSargables, pred, node)

	// Ordering scan, if any, will go into scans[0]
	var scanBuf [16]plan.SecondaryScan
	var scans []plan.SecondaryScan
	var indexProjection *plan.IndexProjection
	sargLength = 0
	cap := len(indexes) + len(flex) + len(searchSargables)

	if cap <= len(scanBuf) {
		scans = scanBuf[0:1]
	} else {
		scans = make([]plan.SecondaryScan, 1, cap)
	}

	if len(indexes) == 1 {
		for _, entry := range indexes {
			indexProjection = this.buildIndexProjection(entry, nil, nil, true, nil)
			if cap == 1 && this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET) {
				this.limit = offsetPlusLimit(this.offset, this.limit)
				this.resetOffset()
			}
			break
		}
	} else {
		indexProjection = this.buildIndexProjection(nil, nil, nil, true, nil)
	}

	for index, entry := range indexes {
		// skip primary index with no sargable keys. Able to do PrimaryScan
		if index.IsPrimary() && entry.minKeys == 0 {
			continue
		}
		// If this is a join with primary key (meta().id), then it's
		// possible to get right hand documdents directly without
		// accessing through an index (similar to "regular" join).
		// In such cases do not consider secondary indexes that does
		// not include meta().id as a sargable index key. In addition,
		// the index must have either a WHERE clause or at least
		// one other sargable key.
		if node.IsPrimaryJoin() && id != nil {
			metaFound := false
			for _, key := range entry.sargKeys {
				if key.EquivalentTo(id) {
					metaFound = true
					break
				}
			}

			if !metaFound || (len(entry.sargKeys) <= 1 && index.Condition() == nil) {
				continue
			}
		}

		var indexKeyOrders plan.IndexKeyOrders
		if orderEntry != nil && index == orderEntry.index {
			_, indexKeyOrders = this.useIndexOrder(entry, entry.keys)
		}

		if entry.index.Type() != datastore.SYSTEM {
			this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		}
		scan = entry.spans.CreateScan(index, node, this.context.IndexApiVersion(), false, false,
			overlapSpans(pred), false, this.offset, this.limit, indexProjection,
			indexKeyOrders, nil, nil, nil, nil, entry.cost, entry.cardinality,
			entry.size, entry.frCost, hasDeltaKeyspace)

		if orderEntry != nil && index == orderEntry.index {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}

		if len(entry.sargKeys) > sargLength {
			sargLength = len(entry.sargKeys)
		}
		for _, a := range entry.unnestAliases {
			baseKeyspace.AddUnnestIndex(index, a)
		}
	}

	this.resetProjection()

	// Search() access path for flex indexes
	for index, entry := range flex {
		sfn := entry.sargKeys[0].(*search.Search)
		sOrders := searchOrders
		if entry != orderEntry {
			sOrders = nil
		}

		this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		scan := this.CreateFTSSearch(index, node, sfn, sOrders, nil, nil, hasDeltaKeyspace)
		if entry == orderEntry {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}

		if entry.maxKeys > sargLength {
			sargLength = entry.maxKeys
		}
	}

	for _, entry := range searchSargables {
		sfn := entry.sargKeys[0].(*search.Search)
		sOrders := searchOrders
		if entry != orderEntry {
			sOrders = nil
		}

		this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		scan := this.CreateFTSSearch(entry.index, node, sfn, sOrders, nil, nil, hasDeltaKeyspace)
		if entry == orderEntry {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}

		if entry.maxKeys > sargLength {
			sargLength = entry.maxKeys
		}
	}

	if len(scans) == 1 {
		this.orderScan = scans[0]
		return scans[0], sargLength, nil
	} else if scans[0] == nil && len(scans) == 2 {
		return scans[1], sargLength, nil
	} else if scans[0] == nil {
		cost, cardinality, size, frCost := this.intersectScanCost(node, scans[1:]...)
		return plan.NewIntersectScan(limit, cost, cardinality, size, frCost, scans[1:]...), sargLength, nil
	} else {
		cost, cardinality, size, frCost := this.intersectScanCost(node, scans...)
		scan = plan.NewOrderedIntersectScan(nil, cost, cardinality, size, frCost, scans...)
		this.orderScan = scan
		return scan, sargLength, nil
	}
}

func (this *builder) buildSecondaryScanPushdowns(indexes, flex map[datastore.Index]*indexEntry,
	searchSargables []*indexEntry, pred expression.Expression, node *algebra.KeyspaceTerm) (
	orderEntry *indexEntry, limit expression.Expression, searchOrders []string) {

	// get ordered Index. If any index doesn't have Limit/Offset pushdown those must turned off
	pushDown := this.hasOffsetOrLimit()
	for _, entry := range indexes {
		if this.order != nil && orderEntry == nil && entry.IsPushDownProperty(_PUSHDOWN_ORDER) {
			orderEntry = entry
		}

		if pushDown && ((this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET)) ||
			(this.limit != nil && !entry.IsPushDownProperty(_PUSHDOWN_LIMIT))) {
			pushDown = false
		}
	}

	for _, entry := range flex {
		if this.order != nil && orderEntry == nil && entry.IsPushDownProperty(_PUSHDOWN_ORDER) {
			orderEntry = entry
			searchOrders = entry.searchOrders
		}

		if pushDown && ((this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET)) ||
			(this.limit != nil && !entry.IsPushDownProperty(_PUSHDOWN_LIMIT))) {
			pushDown = false
		}
	}

	if len(searchSargables) > 0 {
		if !pushDown {
			this.resetOffsetLimit()
		}

		var searchOrderEntry *indexEntry

		searchOrderEntry, searchOrders, _ = this.searchPagination(searchSargables, pred, node.Alias())
		pushDown = this.hasOffsetOrLimit()
		if orderEntry == nil {
			orderEntry = searchOrderEntry
		}
	}

	// ordered index found turn off parallelism. If not turn off pushdowns
	if orderEntry != nil {
		this.maxParallelism = 1
	} else if this.order != nil {
		this.resetOrderOffsetLimit()
		return nil, nil, nil
	}

	// if not single index turn off pushdowns and apply on IntersectScan
	if pushDown && (len(indexes)+len(flex)+len(searchSargables)) > 1 {
		limit = offsetPlusLimit(this.offset, this.limit)
		this.resetOffsetLimit()
	} else if !pushDown {
		this.resetOffsetLimit()
	}
	return
}

func (this *builder) sargableIndexes(indexes []datastore.Index, pred, subset expression.Expression,
	primaryKey expression.Expressions, formalizer *expression.Formalizer,
	ubs expression.Bindings, join bool) (
	sargables, arrays, flex map[datastore.Index]*indexEntry, err error) {

	flexPred := pred
	if len(this.context.NamedArgs()) > 0 || len(this.context.PositionalArgs()) > 0 {
		flexPred, err = base.ReplaceParameters(flexPred, this.context.NamedArgs(), this.context.PositionalArgs())
		if err != nil {
			return
		}
	}

	sargables = make(map[datastore.Index]*indexEntry, len(indexes))
	arrays = make(map[datastore.Index]*indexEntry, len(indexes))

	var keys expression.Expressions
	var entry *indexEntry
	var flexRequest *datastore.FTSFlexRequest

	for _, index := range indexes {
		if index.Type() == datastore.FTS {
			if this.hintIndexes {
				// FTS Flex index sargability
				if flexRequest == nil {
					flex = make(map[datastore.Index]*indexEntry, len(indexes))
					flexRequest = this.buildFTSFlexRequest(formalizer.Keyspace(), flexPred, ubs)
				}

				entry, err = this.sargableFlexSearchIndex(index, flexRequest, join)
				if err != nil {
					return
				}

				if entry != nil {
					flex[index] = entry
				}
			}
			continue
		}

		var cond, origCond expression.Expression
		var allKey *expression.All
		var apos int

		if index.IsPrimary() {
			if primaryKey != nil {
				keys = primaryKey
			} else {
				continue
			}
		} else {
			cond = index.Condition()
			if cond != nil {
				if subset == nil {
					continue
				}

				if cond, origCond, err = formalizeExpr(formalizer, cond, true); err != nil {
					return
				}

				if !base.SubsetOf(subset, cond) && (origCond == nil || !base.SubsetOf(subset, origCond)) {
					continue
				}
			}

			rangeKeys := index.RangeKey()
			for i, key := range rangeKeys {
				if a, ok := key.(*expression.All); ok {
					allKey = a
					apos = i
				}
			}

			keys = expression.CopyExpressions(expression.GetFlattenKeys(rangeKeys))

			for i, key := range keys {
				if key, _, err = formalizeExpr(formalizer, key, false); err != nil {
					return
				}
				keys[i] = key
			}
		}

		var partitionKeys expression.Expressions
		partitionKeys, err = indexPartitionKeys(index, formalizer)
		if err != nil {
			return
		}

		skip := useSkipIndexKeys(index, this.context.IndexApiVersion())
		missing := indexHasLeadingKeyMissingValues(index, this.context.FeatureControls())
		min, max, sum, skeys := SargableFor(pred, keys, missing, skip, this.context, this.aliases)
		exact := min == 0 && pred == nil

		n := min
		if skip && (n > 0 || missing) {
			n = max
		}

		if n == 0 && missing && primaryKey != nil {
			n = 1
		}

		if n > 0 || allKey != nil {
			entry := newIndexEntry(index, keys, keys[0:n], partitionKeys, min, n, sum, cond, origCond, nil, exact, skeys)
			if missing {
				entry.SetFlags(IE_LEADINGMISSING, true)
			}
			if allKey != nil {
				arrays[index] = entry
				var ak expression.Expression
				if ak, _, err = formalizeExpr(formalizer, allKey, false); err != nil {
					return
				}
				allKey, _ := ak.(*expression.All)
				entry.setArrayKey(allKey, apos)
			}

			if n > 0 {
				sargables[index] = entry
			}
		}
	}

	return sargables, arrays, flex, nil
}

func formalizeExpr(formalizer *expression.Formalizer, expr expression.Expression, orig bool) (
	newExpr, origExpr expression.Expression, err error) {
	if expr != nil {
		expr = expr.Copy()
		formalizer.SetIndexScope()
		expr, err = formalizer.Map(expr)
		formalizer.ClearIndexScope()
		if err == nil {
			if orig {
				origExpr = expr.Copy()
			}

			dnf := base.NewDNF(expr, true, true)
			expr, err = dnf.Map(expr)
		}
	}
	return expr, origExpr, err
}

func indexPartitionKeys(index datastore.Index,
	formalizer *expression.Formalizer) (partitionKeys expression.Expressions, err error) {

	index3, ok := index.(datastore.Index3)
	if !ok {
		return
	}

	partitionInfo, _ := index3.PartitionKeys()
	if partitionInfo == nil || partitionInfo.Strategy == datastore.NO_PARTITION {
		return
	}

	partitionKeys = partitionInfo.Exprs
	if formalizer == nil {
		return partitionKeys, err
	}

	partitionKeys = partitionKeys.Copy()
	for i, key := range partitionKeys {
		key = key.Copy()

		formalizer.SetIndexScope()
		partitionKeys[i], err = formalizer.Map(key)
		formalizer.ClearIndexScope()
		if err != nil {
			return nil, err
		}
	}
	return partitionKeys, err
}

func (this *builder) minimalIndexes(sargables map[datastore.Index]*indexEntry, shortest bool,
	pred expression.Expression, node *algebra.KeyspaceTerm) map[datastore.Index]*indexEntry {

	alias := node.Alias()
	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())

	var predFc map[string]value.Value
	if pred != nil {
		predFc = _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(predFc)
		predFc = pred.FilterCovers(predFc)
	}

	if useCBO {
		advisorValidate := this.advisorValidate()
		for _, se := range sargables {
			if se.cost <= 0.0 {
				cost, selec, card, size, frCost, e := indexScanCost(se.index, se.sargKeys,
					this.context.RequestId(), se.spans, alias, advisorValidate,
					this.context)
				if e != nil || (cost <= 0.0 || card <= 0.0 || size <= 0 || frCost <= 0.0) {
					useCBO = false
				} else {
					se.cardinality, se.selectivity, se.cost, se.frCost, se.size = card, selec, cost, frCost, size
				}
			}
		}
	}

	for s, se := range sargables {
		for t, te := range sargables {
			if t == s {
				continue
			}

			if useCBO && shortest {
				if matchedLeadingKeys(se, te, predFc) && se.cost < te.cost {
					delete(sargables, t)
				}
			} else {
				if narrowerOrEquivalent(se, te, shortest, predFc) {
					delete(sargables, t)
				}
			}
		}
	}

	if useCBO && shortest && len(sargables) > 1 {
		sargables = this.chooseIntersectScan(sargables, node)
	}

	return sargables
}

/*
Is se narrower or equivalent to te.
  true : purge te
  false: keep both

*/
func narrowerOrEquivalent(se, te *indexEntry, shortest bool, predFc map[string]value.Value) bool {

	snk, snc := matchedKeysConditions(se, te, shortest, predFc)

	be := bestIndexBySargableKeys(se, te, se.nEqCond, te.nEqCond)
	if be == te { // te is better index
		return false
	}

	if te.nSargKeys > 0 && te.nSargKeys == snk+snc && se.nSargKeys > snk && be == se {
		// all te sargable keys are se sargable keys and part of se condition;
		//  se have more different sragable keys

		return true
	}

	// non shortest case keep both indexes so that covering can be done on best index
	if !shortest {
		return false
	}

	if be == se {
		return true
	}

	// if te and se has same sargKeys (or equivalent condition), and there exists
	// a non-sarged array key, prefer the one without the array key
	if te.nSargKeys == snk+snc &&
		se.PushDownProperty() == te.PushDownProperty() {
		if se.HasFlag(IE_ARRAYINDEXKEY) != te.HasFlag(IE_ARRAYINDEXKEY) {
			if !se.HasFlag(IE_ARRAYINDEXKEY_SARGABLE) && !te.HasFlag(IE_ARRAYINDEXKEY_SARGABLE) {
				if te.HasFlag(IE_ARRAYINDEXKEY) && !se.HasFlag(IE_ARRAYINDEXKEY) {
					return true
				} else if se.HasFlag(IE_ARRAYINDEXKEY) && !te.HasFlag(IE_ARRAYINDEXKEY) {
					return false
				}
			}

		}
	}

	if te.cond != nil && (se.cond == nil || !base.SubsetOf(se.cond, te.cond)) {
		return false
	}

	if te.nSargKeys > 0 {
		if te.nSargKeys > (snk + snc) {
			return false
		} else if te.nSargKeys == (snk+snc) &&
			se.PushDownProperty() == te.PushDownProperty() {
			if se.minKeys != te.minKeys {
				// for two indexes with the same sargKeys, favor the one
				// with more consecutive leading sargKeys
				// e.g (c1, c4) vs (c1, c2, c4) with predicates on c1 and c4
				return se.minKeys > te.minKeys
			} else if se.maxKeys != te.maxKeys {
				// favor the one with shorter sargKeys
				return se.maxKeys < te.maxKeys
			}
		}
	}

	if se.sumKeys+se.nEqCond != te.sumKeys+te.nEqCond {
		return se.sumKeys+se.nEqCond > te.sumKeys+te.nEqCond
	}

	if se.PushDownProperty() != te.PushDownProperty() {
		return se.PushDownProperty() > te.PushDownProperty()
	}

	return se.cond != nil ||
		len(se.keys) < len(te.keys) ||
		!(te.nSargKeys == 0 && se.nSargKeys == 0 && te.index.IsPrimary())
}

// Calculates how many keys te sargable keys matched with se sargable keys and se condition
func matchedKeysConditions(se, te *indexEntry, shortest bool, predFc map[string]value.Value) (nk, nc int) {

outer:
	for ti, tk := range te.sargKeys {
		if !te.skeys[ti] {
			continue
		}

		for si, sk := range se.sargKeys {
			if se.skeys[si] {
				if matchedIndexKey(sk, tk) {
					nk++
					continue outer
				}
			}
		}

		if matchedIndexCondition(se, tk, predFc, shortest) {
			nc++
		}
	}

	return
}

// for CBO, prune indexes that has similar leading index keys
func matchedLeadingKeys(se, te *indexEntry, predFc map[string]value.Value) bool {
	nkeys := 0
	ncond := 0
	for i, tk := range te.sargKeys {
		if !te.skeys[i] {
			continue
		}

		if i >= len(se.sargKeys) {
			break
		}

		if !se.skeys[i] {
			continue
		}

		sk := se.sargKeys[i]
		if matchedIndexKey(sk, tk) {
			nkeys++
		} else if matchedIndexCondition(se, tk, predFc, true) {
			ncond++
		} else {
			break
		}
	}

	return (nkeys + ncond) > 0
}

func matchedIndexKey(sk, tk expression.Expression) bool {
	return base.SubsetOf(sk, tk) || sk.DependsOn(tk)
}

func matchedIndexCondition(se *indexEntry, tk expression.Expression, predFc map[string]value.Value,
	shortest bool) bool {

	/* Count number of matches
	 * Indexkey is part of other index condition as equality predicate
	 * If trying to determine shortest index(For: IntersectScan)
	 *     indexkey is not equality predicate and indexkey is part of other index condition
	 *     (In case of equality predicate keeping IntersectScan might be better)
	 */
	_, condEq := se.condFc[tk.String()]
	_, predEq := predFc[tk.String()]
	if condEq || (shortest && !predEq && se.cond != nil && se.cond.DependsOn(tk)) {
		return true
	}

	return false
}

func (this *builder) sargIndexes(baseKeyspace *base.BaseKeyspace, underHash bool,
	sargables map[datastore.Index]*indexEntry) error {

	pred := baseKeyspace.DnfPred()
	isOrPred := false
	orIsJoin := false
	if !underHash {
		if _, ok := pred.(*expression.Or); ok {
			isOrPred = true
			for _, fl := range baseKeyspace.Filters() {
				if fl.IsJoin() {
					orIsJoin = true
					break
				}
			}
		}
	}

	useCBO := this.useCBO && (baseKeyspace.DocCount() >= 0)
	advisorValidate := this.advisorValidate()
	for _, se := range sargables {
		var spans SargSpans
		var exactSpans bool
		var err error

		if se.maxKeys == 0 {
			se.spans = _WHOLE_SPANS.Copy()
			if pred != nil {
				se.exactSpans = false
			}
			continue
		}

		useFilters := true
		if isOrPred {
			useFilters = false
		} else {
			for _, key := range se.keys {
				if _, ok := key.(*expression.And); ok {
					useFilters = false
					break
				}
			}
		}
		isMissings := getMissings(se)

		if useFilters {
			spans, exactSpans, err = SargForFilters(baseKeyspace.Filters(), se.keys, isMissings,
				se.maxKeys, underHash, useCBO, baseKeyspace, this.keyspaceNames,
				advisorValidate, this.aliases, this.context)
		} else {
			spans, exactSpans, err = SargFor(baseKeyspace.DnfPred(), se, se.keys, isMissings,
				se.maxKeys, orIsJoin, useCBO, baseKeyspace, this.keyspaceNames,
				advisorValidate, this.aliases, this.context)
		}
		if err != nil || spans.Size() == 0 {
			logging.Errora(func() string {
				return fmt.Sprintf("Sargable index not sarged: pred:<ud>%v</ud> sarg_keys:<ud>%v</ud> error:%v",
					pred,
					se.sargKeys,
					err,
				)
			})

			return errors.NewPlanError(nil,
				fmt.Sprintf("Sargable index not sarged; pred=%v, sarg_keys=%v, error=%v",
					pred.String(), se.sargKeys.String(), err))
		}

		se.spans = spans
		if exactSpans && !useIndex2API(se.index, this.context.IndexApiVersion()) {
			exactSpans = spans.ExactSpan1(len(se.keys))
		}
		se.exactSpans = exactSpans
	}

	return nil
}

func indexHasArrayIndexKey(index datastore.Index) bool {
	return hasArrayIndexKey(index.RangeKey())
}

func hasArrayIndexKey(keys expression.Expressions) bool {
	for _, sk := range keys {
		if isArray, _, _ := sk.IsArrayIndexKey(); isArray {
			return true
		}
	}
	return false
}

func (this *builder) chooseIntersectScan(sargables map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm) map[datastore.Index]*indexEntry {

	keyspace, err := this.getTermKeyspace(node)
	if err != nil {
		return sargables
	}

	nTerms := 0
	if this.order != nil {
		nTerms = len(this.order.Terms())
	}

	return optChooseIntersectScan(keyspace, sargables, nTerms, node.Alias(),
		this.advisorValidate(), this.context)
}

func bestIndexBySargableKeys(se, te *indexEntry, snc, tnc int) *indexEntry {
	si := 0
	ti := 0
	for si < len(se.skeys) && ti < len(te.skeys) {
		if se.skeys[si] != te.skeys[ti] {
			if se.skeys[si] {
				if tnc == 0 {
					return se
				}
				tnc--
				si++
			} else if te.skeys[ti] {
				if snc == 0 {
					return te
				}
				snc--
				ti++
			}
		} else {
			si++
			ti++
		}
	}

	for ; si < len(se.skeys); si++ {
		if se.skeys[si] {
			if tnc == 0 {
				return se
			}
			tnc--
		}
	}

	for ; ti < len(te.skeys); ti++ {
		if te.skeys[ti] {
			if snc == 0 {
				return te
			}
			snc--
		}
	}

	if tnc > snc {
		return te
	} else if snc > tnc {
		return se
	}

	return nil
}

func overlapSpans(expr expression.Expression) bool {
	return expr != nil && expr.MayOverlapSpans()
}

func getMissings(entry *indexEntry) (isMissings []bool) {
	// MB-38287. isMissing true non-array indexkeys only.
	isMissings = make([]bool, len(entry.keys))
	size := 1
	if entry.arrayKey != nil && entry.arrayKey.Flatten() {
		size = entry.arrayKey.FlattenSize()
	}
	for i, _ := range entry.keys {
		if entry.arrayKey == nil || i < entry.arrayKeyPos || i >= entry.arrayKeyPos+size {
			isMissings[i] = true
		}
	}
	return
}
