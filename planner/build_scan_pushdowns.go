//  Copyright (c) 2017 Couchbase, Inc.
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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func (this *builder) indexPushDownProperty(entry *indexEntry, indexKeys, unnestFiletrs expression.Expressions,
	pred expression.Expression, alias string, unnest, covering bool) (pushDownProperty PushDownProperties) {

	// Check all predicates are part of spans, exact and no false positives possible
	exact := this.checkExactSpans(entry, pred, alias, unnestFiletrs)
	if exact {
		pushDownProperty |= _PUSHDOWN_EXACTSPANS
	}

	// Covering index check for other pushdowns
	if covering && exact {
		pushDownProperty |= this.indexCoveringPushDownProperty(entry, indexKeys, alias, unnest, pushDownProperty)
	}

	// Check Query Order By matches with index key order.
	exactLimitOffset := exact
	if this.order != nil {
		ok, _ := this.useIndexOrder(entry, entry.keys)
		if ok && (this.group == nil || isPushDownProperty(pushDownProperty, _PUSHDOWN_FULLGROUPAGGS)) {
			pushDownProperty |= _PUSHDOWN_ORDER
		} else {
			exactLimitOffset = false
		}
	} else if this.group != nil && !isPushDownProperty(pushDownProperty, _PUSHDOWN_FULLGROUPAGGS) {
		exactLimitOffset = false
	}

	// Check all predicates are part of spans, exact and no false positives possible
	if isPushDownProperty(pushDownProperty, _PUSHDOWN_EXACTSPANS) {

		// LIMIT Pushdown is possible when
		//        *  Query Order By not present
		//        *  Query Order By matches with Index key order
		//        *  LIMIT is hint to indexer

		if this.limit != nil && exactLimitOffset {
			pushDownProperty |= _PUSHDOWN_LIMIT
		}

		// OFFSET Pushdown is possible when
		//        *  Index API2
		//        *  Offset can be pushed based on spans becasue OFFSET needs to be exact NOT hint to indexer
		//        *  Query Order By not present
		//        *  Query Order By matches with Index key order

		if this.offset != nil && exactLimitOffset && useIndex2API(entry.index, this.context.IndexApiVersion()) &&
			entry.spans.CanPushDownOffset(entry.index, pred.MayOverlapSpans(),
				!unnest && indexHasArrayIndexKey(entry.index)) {
			pushDownProperty |= _PUSHDOWN_OFFSET
		}
	}
	if this.indexAdvisor && covering {
		this.collectPushdownProperty(entry.index, alias, pushDownProperty)
	}
	return pushDownProperty
}

func (this *builder) indexCoveringPushDownProperty(entry *indexEntry, indexKeys expression.Expressions,
	alias string, unnest bool, pushDownProperty PushDownProperties) PushDownProperties {

	// spans needs to be exact
	if !isPushDownProperty(pushDownProperty, _PUSHDOWN_EXACTSPANS) {
		return pushDownProperty
	}

	// Check aggregate pushdowns using API3
	aggProperty, tryApi2 := this.indexAggPushDownProperty(entry, indexKeys, alias, unnest, pushDownProperty)
	pushDownProperty |= aggProperty

	// Exploiting IndexScan for aggregates using API2/API1.
	//          * COUNT(), COUNT(DISTINCT op ), MIN(), MAX(),
	//          * Requires single aggregate in projection
	//          * NO Group By

	if !isPushDownProperty(pushDownProperty, _PUSHDOWN_GROUPAGGS) && this.oldAggregates && tryApi2 &&
		len(this.aggs) == 1 && len(this.group.By()) == 0 && !indexHasArrayIndexKey(entry.index) {
		for _, ag := range this.aggs {
			switch agg := ag.(type) {

			case *algebra.Count:
				if this.canPushDownCount(entry, agg.Operands()[0], indexKeys, agg.Distinct()) {
					pushDownProperty |= _PUSHDOWN_GROUPAGGS
				}

			case *algebra.Min:
				if this.canPushDownMinMax(entry, agg.Operands()[0], indexKeys, false) {
					pushDownProperty |= _PUSHDOWN_GROUPAGGS
				}

			case *algebra.Max:
				if this.canPushDownMinMax(entry, agg.Operands()[0], indexKeys, true) {
					pushDownProperty |= _PUSHDOWN_GROUPAGGS
				}
			}
		}
	}

	// Check Projection Distinct can be pushdown
	if !isPushDownProperty(pushDownProperty, _PUSHDOWN_DISTINCT) &&
		(this.group == nil || isPushDownProperty(pushDownProperty, _PUSHDOWN_GROUPAGGS)) &&
		this.canPushDownProjectionDistinct(entry, this.projection, entry.keys) {
		pushDownProperty |= _PUSHDOWN_DISTINCT
	}

	return pushDownProperty
}

func (this *builder) indexAggPushDownProperty(entry *indexEntry, indexKeys expression.Expressions,
	alias string, unnest bool, pushDownProperty PushDownProperties) (PushDownProperties, bool) {

	if this.group == nil || !useIndex3API(entry.index, this.context.IndexApiVersion()) ||
		!isPushDownProperty(pushDownProperty, _PUSHDOWN_EXACTSPANS) ||
		isPushDownProperty(pushDownProperty, _PUSHDOWN_GROUPAGGS) ||
		!util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_GROUPAGG_PUSHDOWN) {
		return pushDownProperty, true
	}

	// Group keys needs to be covered by index keys (including document key)
	for _, gexpr := range this.group.By() {
		if !expression.IsCovered(gexpr, alias, indexKeys) {
			return pushDownProperty, false
		}
	}

	groupMatch, maxKeyPos := this.indexGroupLeadingIndexKeysMatch(entry, indexKeys)

	// For array Index to use for  non-MIN,non-MAX aggregates
	//    For non Unnest Scan
	//       * Equality predicate on array Index required
	//       * DISTINCT on the array Index key required
	//       * ALL is ok if it is leading key. Indexer will consider one row for META().id
	// For Unnest Scan ALL index or DISTINCT index with Distinct aggregates can use it

	arrayIndexIsOK := true
	for i, sk := range entry.index.RangeKey() {
		if isArray, distinct := sk.IsArrayIndexKey(); isArray {
			if unnest {
				arrayIndexIsOK = !distinct
			} else {
				eq, _ := entry.spans.EquivalenceRangeAt(i)
				arrayIndexIsOK = eq && (distinct || i == 0)
			}
			break
		}
	}

nextagg:
	for _, agg := range this.aggs {
		op := agg.Operands()[0]

		constOp := (op == nil || op.Value() != nil)

		// aggregate expression needs to be covered by index keys (including document key)
		if !constOp && !expression.IsCovered(op, alias, indexKeys) {
			return pushDownProperty, false
		}

		switch agg.(type) {
		case *algebra.Min, *algebra.Max:
			continue nextagg
		default:
			// Distinct aggregates argument can be any key in the matched leading keys + 0|1
			// 0 for partition index and 1 for non partition index
			if agg.Distinct() {
				if groupMatch {
					if constOp {
						continue nextagg
					}

					for i, key := range indexKeys {
						if i <= maxKeyPos && op.EquivalentTo(key) {
							continue nextagg
						}
					}
				}

				// Distinct key not matched leading keys + 0+1
				return pushDownProperty, false
			} else {
				// non Unnest IndexScan can only use DISTINCT array index with eqaulity preidicate
				// Unnest IndexScan can only use DISTINCT array with Distinct aggregates

				if !arrayIndexIsOK {
					return pushDownProperty, false
				}
			}
		}
	}

	pushDownProperty |= _PUSHDOWN_GROUPAGGS
	if groupMatch {
		pushDownProperty |= _PUSHDOWN_FULLGROUPAGGS
	}
	return pushDownProperty, false
}

func (this *builder) indexGroupLeadingIndexKeysMatch(entry *indexEntry, indexKeys expression.Expressions) (bool, int) {

	// generate unique group keys.
	groupkeys := make(map[string]bool, len(this.group.By())+1)
	for _, gexpr := range this.group.By() {
		// ignore constants
		if gexpr.Value() != nil {
			continue
		}
		groupkeys[gexpr.String()] = true
	}

	// For Partition index the partition keys needs to be in group keys to use DIstinct aggregates
	for _, pexpr := range entry.partitionKeys {
		if _, ok := groupkeys[pexpr.String()]; !ok {
			return false, 0
		}
	}

	// Check group keys matching leading keys. If equality predicate that index key can be skipped in group
	nMatched := 0
	nGroupMatched := 0
	for nMatched < len(indexKeys) {
		if _, ok := groupkeys[indexKeys[nMatched].String()]; ok {
			// index key matched with group key, check duplicate index keys
			duplicate := false
			for k := 0; !duplicate && k <= nMatched-1; k++ {
				if indexKeys[nMatched].EquivalentTo(indexKeys[k]) {
					duplicate = true
				}
			}
			nMatched++
			if !duplicate {
				nGroupMatched++
			}
		} else if eq, _ := entry.spans.EquivalenceRangeAt(nMatched); eq {
			// index key is equality predicate, skip it
			nMatched++
		} else {
			// not matched break it
			break
		}
	}

	// Check all group keys matched with leading index keys
	return (nGroupMatched == len(groupkeys)), nMatched
}

func (this *builder) checkExactSpans(entry *indexEntry, pred expression.Expression, alias string,
	unnestFiletrs expression.Expressions) bool {
	// spans are not exact
	if !entry.exactSpans {
		return false
	}

	// check for non sargable key is in predicate
	exprs, _, err := indexCoverExpressions(entry, entry.sargKeys, pred, nil, alias)
	if err != nil {
		return false
	}

	if this.aggConstraint != nil {
		exprs = append(exprs, this.aggConstraint)
	}

	if len(unnestFiletrs) > 0 {
		exprs = append(exprs, unnestFiletrs...)
	}

	if !expression.IsCovered(pred, alias, exprs) {
		return false
	}

	// all predicates are part of spans, exact and no false positives possible
	return true
}

func (this *builder) canPushDownCount(entry *indexEntry, op expression.Expression,
	keys expression.Expressions, distinct bool) bool {

	// COUNT( DISTINCT op) is supported in API2
	if distinct && !useIndex2API(entry.index, this.context.IndexApiVersion()) {
		return false
	}

	// no operand can be push down
	if op == nil {
		return !distinct
	}

	// constant non-MISSING, non-NULL operand can be pushdown
	val := op.Value()
	if val != nil {
		return val.Type() > value.NULL
	}

	// operand needs to be leading key
	if len(keys) == 0 || !op.EquivalentTo(keys[0]) {
		return false
	}

	// Scan should not include NULL or MISSING
	return entry.spans.SkipsLeadingNulls()
}

func (this *builder) canPushDownMinMax(entry *indexEntry, op expression.Expression, keys expression.Expressions,
	max bool) bool {
	// aggregate operand is constant
	if op.Value() != nil {
		return true
	}

	// MAX() pushdown is supported in API2 only
	if max && !useIndex2API(entry.index, this.context.IndexApiVersion()) {
		return false
	}

	// aggregate operand needs to be leading key
	if len(keys) == 0 || !op.EquivalentTo(keys[0]) {
		return false
	}

	// get the index collation of the leading key
	descCollation := indexKeyIsDescCollation(0, getIndexKeys(entry))
	if max {
		// MAX() can be pushdown when leading index key is DESC collation
		return descCollation && entry.spans.CanUseIndexOrder(false)
	} else {
		// MIN() can be pushdown when leading index key is ASC collation and NULLS are not included
		return !descCollation && entry.spans.CanUseIndexOrder(false) && entry.spans.SkipsLeadingNulls()
	}
}

func (this *builder) canPushDownProjectionDistinct(entry *indexEntry, projection *algebra.Projection,
	indexKeys expression.Expressions) bool {

	// Only supported in API2
	if projection == nil || !useIndex2API(entry.index, this.context.IndexApiVersion()) || !projection.Distinct() {
		return false
	}

	// Disable distinct pushdown for HASH partition. API3
	if useIndex3API(entry.index, this.context.IndexApiVersion()) {
		partition, err := entry.index.(datastore.Index3).PartitionKeys()
		if err != nil || (partition != nil && partition.Strategy != datastore.NO_PARTITION) {
			return false
		}
	}

	hash := _STRING_BOOL_POOL.Get()
	defer _STRING_BOOL_POOL.Put(hash)

	for _, key := range indexKeys {
		hash[key.String()] = true
	}

	// all projections needs to be part of the index keys
	for _, expr := range projection.Expressions() {
		if expr.Value() == nil {
			if _, ok := hash[expr.String()]; !ok {
				return false
			}
		}
	}

	return true
}

func (this *builder) useIndexOrder(entry *indexEntry, keys expression.Expressions) (bool, plan.IndexKeyOrders) {

	// Force the use of sorts on indexes that we know not to be ordered
	// (for now system indexes)
	// for now - if they are of a non descript type, then they aren't sorted
	// when GSI starts implementing other types of indexes (eg bitmap)
	// we will revisit this approach

	if entry.index.Type() == datastore.SYSTEM ||
		!entry.spans.CanUseIndexOrder(useIndex3API(entry.index, this.context.IndexApiVersion())) {
		return false, nil
	}

	var filters map[string]value.Value
	if entry.cond != nil {
		filters = _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(filters)
		filters = entry.cond.FilterCovers(filters)
		filters = entry.origCond.FilterCovers(filters)
	}

	var hashProj map[string]expression.Expression

	if this.projection != nil {
		hashProj = make(map[string]expression.Expression, len(this.projection.Terms()))
		for _, term := range this.projection.Terms() {
			hashProj[term.Alias()] = term.Expression()
		}
	}

	indexKeys := getIndexKeys(entry)
	i := 0
	indexOrder := make(plan.IndexKeyOrders, 0, len(keys))
outer:
	for _, orderTerm := range this.order.Terms() {
		// orderTerm is constant
		if orderTerm.Expression().Static() != nil {
			continue
		}

		// non-constant orderTerms are more than index keys
		if i >= len(keys) {
			// match with condition EQ terms
			if equalConditionFilter(filters, orderTerm.Expression().String()) {
				continue outer
			}
			return false, nil
		}

		for {
			projexpr, projalias := hashProj[orderTerm.Expression().Alias()]
			if indexKeyIsDescCollation(i, indexKeys) == orderTerm.Descending() &&
				(!orderTerm.NullsPos() || !entry.spans.CanProduceUnknowns(i)) &&
				(orderTerm.Expression().EquivalentTo(keys[i]) ||
					(projalias && expression.Equivalent(projexpr, keys[i]))) {
				// orderTerm matched with index key
				indexOrder = append(indexOrder, plan.NewIndexKeyOrders(i, orderTerm.Descending()))
				i++
				continue outer
			} else if equalConditionFilter(filters, orderTerm.Expression().String()) {
				// orderTerm matched with Condition EQ
				continue outer
			} else if eq, _ := entry.spans.EquivalenceRangeAt(i); eq {
				// orderTerm not yet matched, but can skip equivalence range key
				indexOrder = append(indexOrder,
					plan.NewIndexKeyOrders(i, indexKeyIsDescCollation(i, indexKeys)))
				i++
				if i >= len(keys) {
					return false, nil
				}
			} else {
				return false, nil
			}
		}
	}

	return true, indexOrder
}

func equalConditionFilter(filters map[string]value.Value, str string) bool {
	if filters == nil {
		return false
	}

	v, ok := filters[str]
	return ok && v != nil
}

func indexKeyIsDescCollation(keypos int, indexKeys datastore.IndexKeys) bool {
	return len(indexKeys) > 0 && keypos < len(indexKeys) && indexKeys[keypos].HasAttribute(datastore.IK_DESC)
}
