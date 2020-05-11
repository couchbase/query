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
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
)

func Build(stmt algebra.Statement, datastore, systemstore datastore.Datastore,
	namespace string, subquery, stream bool, context *PrepareContext) (
	plan.Operator, error) {

	builder := newBuilder(datastore, systemstore, namespace, subquery, context)
	if util.IsFeatureEnabled(context.FeatureControls(), util.N1QL_CBO) && context.Optimizer() != nil {
		builder.useCBO = true
	}

	o, err := stmt.Accept(builder)

	if err != nil {
		return nil, err
	}

	op := o.(plan.Operator)
	_, is_prepared := o.(*plan.Prepared)

	if !subquery && !is_prepared {
		privs, er := stmt.Privileges()
		if er != nil {
			return nil, er
		}

		if stream {
			op = plan.NewSequence(op, plan.NewStream(op.Cost(), op.Cardinality()))
		}

		// Always insert an Authorize operator, even if no privileges need to
		// be verified.
		//
		// We do this because the list of authenticated users is generated as
		// part of authentication, and this list may be needed in the query
		// (see the function CURRENT_USERS()).
		//
		// This should not impose a burden in production because every real
		// query is against secured tables anyway, and would therefore
		// have privileges that need verification, meaning the Authorize
		// operator would have been present in any case.
		return plan.NewAuthorize(privs, op), nil
	} else {
		return op, nil
	}
}

var _MAP_KEYSPACE_CAP = 4

const (
	OPT_SELEC_NOT_AVAIL = -1.0 // selectivity is not available
	OPT_COST_NOT_AVAIL  = -1.0 // cost is not available
	OPT_CARD_NOT_AVAIL  = -1.0 // cardinality is not available
)

const (
	BUILDER_WHERE_IS_TRUE  = 1 << iota // WHERE clause is TRUE
	BUILDER_WHERE_IS_FALSE             // WHERE clause is FALSE
)

type builder struct {
	indexPushDowns
	collectQueryInfo
	context           *PrepareContext
	datastore         datastore.Datastore
	systemstore       datastore.Datastore
	namespace         string
	subquery          bool
	correlated        bool
	maxParallelism    int
	delayProjection   bool                  // Used to allow ORDER BY non-projected expressions
	from              algebra.FromTerm      // Used for index selection
	where             expression.Expression // Used for index selection
	setOpDistinct     bool                  // Used for SETOP Distinct to apply DISTINCT on projection
	children          []plan.Operator
	subChildren       []plan.Operator
	cover             expression.HasExpressions
	node              expression.HasExpressions
	coveringScans     []plan.CoveringOperator
	coveredUnnests    map[*algebra.Unnest]bool
	countScan         plan.CoveringOperator
	skipDynamic       bool
	requirePrimaryKey bool
	orderScan         plan.SecondaryScan
	baseKeyspaces     map[string]*base.BaseKeyspace
	keyspaceNames     map[string]string
	pushableOnclause  expression.Expression // combined ON-clause from all inner joins
	builderFlags      uint32
	indexAdvisor      bool
	useCBO            bool
	hintIndexes       bool
	lastOp            plan.Operator // last operator built, to get cost/cardinality info
}

type indexPushDowns struct {
	order         *algebra.Order        // Used to collect aggregates from ORDER BY, and for ORDER pushdown
	limit         expression.Expression // Used for LIMIT pushdown
	offset        expression.Expression // Used for OFFSET pushdown
	oldAggregates bool                  // Used for pre-API3 Projection aggregate
	projection    *algebra.Projection   // Used for ORDER/projection Distinct pushdown to IndexScan2
	group         *algebra.Group        // Group BY
	aggs          algebra.Aggregates    // all aggregates in query
	aggConstraint expression.Expression // aggregate Constraint
}

func (this *builder) storeIndexPushDowns() *indexPushDowns {
	idxPushDowns := &indexPushDowns{}
	idxPushDowns.order = this.order
	idxPushDowns.limit = this.limit
	idxPushDowns.offset = this.offset
	idxPushDowns.oldAggregates = this.oldAggregates
	idxPushDowns.projection = this.projection
	idxPushDowns.group = this.group
	idxPushDowns.aggs = this.aggs
	idxPushDowns.aggConstraint = this.aggConstraint

	return idxPushDowns
}

func (this *builder) restoreIndexPushDowns(idxPushDowns *indexPushDowns, pagination bool) {
	if pagination {
		this.order = idxPushDowns.order
		this.limit = idxPushDowns.limit
		this.offset = idxPushDowns.offset
	}
	this.oldAggregates = idxPushDowns.oldAggregates
	this.projection = idxPushDowns.projection
	this.group = idxPushDowns.group
	this.aggs = idxPushDowns.aggs
	this.aggConstraint = idxPushDowns.aggConstraint
}

func newBuilder(datastore, systemstore datastore.Datastore, namespace string, subquery bool,
	context *PrepareContext) *builder {
	rv := &builder{
		datastore:       datastore,
		systemstore:     systemstore,
		namespace:       namespace,
		subquery:        subquery,
		delayProjection: false,
		context:         context,
	}

	return rv
}

func (this *builder) trueWhereClause() bool {
	return (this.builderFlags & BUILDER_WHERE_IS_TRUE) != 0
}

func (this *builder) setTrueWhereClause() {
	this.builderFlags |= BUILDER_WHERE_IS_TRUE
}

func (this *builder) unsetTrueWhereClause() {
	this.builderFlags &^= BUILDER_WHERE_IS_TRUE
}

func (this *builder) falseWhereClause() bool {
	return (this.builderFlags & BUILDER_WHERE_IS_FALSE) != 0
}

func (this *builder) setFalseWhereClause() {
	this.builderFlags |= BUILDER_WHERE_IS_FALSE
}

func (this *builder) collectKeyspaceNames() {
	if len(this.keyspaceNames) > 0 || len(this.baseKeyspaces) == 0 {
		return
	}

	this.keyspaceNames = make(map[string]string, len(this.baseKeyspaces))
	for _, ks := range this.baseKeyspaces {
		this.keyspaceNames[ks.Name()] = ks.Keyspace()
	}

	return
}

func (this *builder) getTermKeyspace(node *algebra.KeyspaceTerm) (datastore.Keyspace, error) {
	path := node.Path()
	path.SetDefaultNamespace(this.namespace)
	ns := strings.ToLower(path.Namespace())
	keyspace, err := datastore.GetKeyspace(path.Parts()...)

	if err != nil && this.indexAdvisor && ns != "#system" &&
		(strings.Contains(err.TranslationKey(), "bucket_not_found") ||
			strings.Contains(err.TranslationKey(), "scope_not_found") ||
			strings.Contains(err.TranslationKey(), "keyspace_not_found")) {

		virtualKeyspace, err1 := this.getVirtualKeyspace(ns, path.Parts())
		if err1 == nil {
			return virtualKeyspace, nil
		}
	}

	if err == nil && this.indexAdvisor {
		this.setKeyspaceFound()
	}

	return keyspace, err
}

func (this *builder) getDocCount(node *algebra.KeyspaceTerm) (float64, error) {
	keyspace, err := this.getTermKeyspace(node)
	if err != nil {
		return 0.0, err
	}

	docCount, err := keyspace.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil {
		return 0.0, err
	}

	return float64(docCount), nil
}

func (this *builder) addSubChildren(ops ...plan.Operator) {
	if len(ops) > 0 {
		this.lastOp = ops[len(ops)-1]
		this.subChildren = append(this.subChildren, ops...)
	}
}

func (this *builder) addChildren(ops ...plan.Operator) {
	if len(ops) > 0 {
		this.lastOp = ops[len(ops)-1]
		this.children = append(this.children, ops...)
	}
}

func (this *builder) addParallel(subChildren ...plan.Operator) *plan.Parallel {
	return plan.NewParallel(plan.NewSequence(subChildren...), this.maxParallelism)
}

func (this *builder) addSubchildrenParallel() *plan.Parallel {
	parallel := plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism)
	this.subChildren = make([]plan.Operator, 0, 16)
	return parallel
}
