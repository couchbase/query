//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package planner

import (
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func Build(stmt algebra.Statement, datastore, systemstore datastore.Datastore,
	namespace string, subquery, stream bool, context *PrepareContext) (
	plan.Operator, map[string]bool, error) {

	builder := newBuilder(datastore, systemstore, namespace, subquery, context)
	if context.UseCBO() && context.Optimizer() != nil {
		builder.useCBO = true
		checkCostModel(context.FeatureControls())
	}

	o, err := stmt.Accept(builder)

	if err != nil {
		return nil, nil, err
	}

	op := o.(plan.Operator)
	_, is_prepared := o.(*plan.Prepared)
	indexKeyspaces := builder.indexKeyspaceNames

	if !subquery && !is_prepared {
		privs, er := stmt.Privileges()
		if er != nil {
			return nil, nil, er
		}

		if stream {
			op = plan.NewSequence(op, plan.NewStream(op.Cost(), op.Cardinality(), op.Size(), op.FrCost()))
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
		return plan.NewAuthorize(privs, op), indexKeyspaces, nil
	} else {
		return op, indexKeyspaces, nil
	}
}

var _MAP_KEYSPACE_CAP = 4

const (
	OPT_SELEC_NOT_AVAIL = float64(-1.0) // selectivity is not available
	OPT_COST_NOT_AVAIL  = float64(-1.0) // cost is not available
	OPT_CARD_NOT_AVAIL  = float64(-1.0) // cardinality is not available
	OPT_SIZE_NOT_AVAIL  = int64(-1)     // document size is not available
)

const (
	BUILDER_WHERE_IS_TRUE  = 1 << iota // WHERE clause is TRUE
	BUILDER_WHERE_IS_FALSE             // WHERE clause is FALSE
	BUILDER_HAS_LIMIT
	BUILDER_HAS_OFFSET
	BUILDER_HAS_GROUP
	BUILDER_HAS_ORDER
	BUILDER_HAS_WINDOW_AGGS
	BUILDER_JOIN_ENUM
	BUILDER_CHK_INDEX_ORDER
	BUILDER_PLAN_HAS_ORDER
)

type builder struct {
	indexPushDowns
	collectQueryInfo
	context            *PrepareContext
	datastore          datastore.Datastore
	systemstore        datastore.Datastore
	namespace          string
	subquery           bool
	correlated         bool
	maxParallelism     int
	delayProjection    bool                  // Used to allow ORDER BY non-projected expressions
	from               algebra.FromTerm      // Used for index selection
	where              expression.Expression // Used for index selection
	filter             expression.Expression // for Filter operator
	setOpDistinct      bool                  // Used for SETOP Distinct to apply DISTINCT on projection
	children           []plan.Operator
	subChildren        []plan.Operator
	cover              expression.HasExpressions
	node               expression.HasExpressions
	coveringScans      []plan.CoveringOperator
	coveredUnnests     map[*algebra.Unnest]bool
	countScan          plan.CoveringOperator
	skipDynamic        bool
	requirePrimaryKey  bool
	orderScan          plan.SecondaryScan
	baseKeyspaces      map[string]*base.BaseKeyspace
	keyspaceNames      map[string]string
	indexKeyspaceNames map[string]bool       // keyspace names that use indexscan (excludes non from caluse subqueries)
	pushableOnclause   expression.Expression // combined ON-clause from all inner joins
	builderFlags       uint32
	indexAdvisor       bool
	useCBO             bool
	hintIndexes        bool
	lastOp             plan.Operator // last operator built, to get cost/cardinality info
}

func (this *builder) Copy() *builder {
	rv := &builder{
		context:            this.context,
		datastore:          this.datastore,
		systemstore:        this.systemstore,
		namespace:          this.namespace,
		subquery:           this.subquery,
		correlated:         this.correlated,
		maxParallelism:     this.maxParallelism,
		delayProjection:    this.delayProjection,
		from:               this.from,
		where:              expression.Copy(this.where),
		filter:             expression.Copy(this.filter),
		setOpDistinct:      this.setOpDistinct,
		cover:              this.cover,
		node:               this.node,
		skipDynamic:        this.skipDynamic,
		requirePrimaryKey:  this.requirePrimaryKey,
		baseKeyspaces:      base.CopyBaseKeyspacesWithFilters(this.baseKeyspaces),
		keyspaceNames:      this.keyspaceNames,
		indexKeyspaceNames: this.indexKeyspaceNames,
		pushableOnclause:   expression.Copy(this.pushableOnclause),
		builderFlags:       this.builderFlags,
		indexAdvisor:       this.indexAdvisor,
		useCBO:             this.useCBO,
		hintIndexes:        this.hintIndexes,
		// the following fields are setup during planning process and thus not copied:
		// children, subChildren, coveringScan, coveredUnnests, countScan, orderScan, lastOp
	}

	this.indexPushDowns.Copy(&rv.indexPushDowns)

	// no need to copy collectQueryInfo

	return rv
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

func (this *indexPushDowns) Copy(newIndexPushdowns *indexPushDowns) {
	newIndexPushdowns.order = this.order
	newIndexPushdowns.limit = expression.Copy(this.limit)
	newIndexPushdowns.offset = expression.Copy(this.offset)
	newIndexPushdowns.oldAggregates = this.oldAggregates
	newIndexPushdowns.projection = this.projection
	newIndexPushdowns.aggs = this.aggs
	newIndexPushdowns.aggConstraint = expression.Copy(this.aggConstraint)
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

	rv.indexKeyspaceNames = make(map[string]bool, _MAP_KEYSPACE_CAP)

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

func (this *builder) joinEnum() bool {
	return (this.builderFlags & BUILDER_JOIN_ENUM) != 0
}

func (this *builder) setJoinEnum() {
	this.builderFlags |= BUILDER_JOIN_ENUM
}

func (this *builder) unsetJoinEnum() {
	this.builderFlags &^= BUILDER_JOIN_ENUM
}

func (this *builder) hasBuilderFlag(flag uint32) bool {
	return (this.builderFlags & flag) != 0
}

func (this *builder) setBuilderFlag(flag uint32) {
	this.builderFlags |= flag
}

func (this *builder) unsetBuilderFlag(flag uint32) {
	this.builderFlags &^= flag
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

func (this *builder) collectIndexKeyspaceNames(ks string) {
	this.indexKeyspaceNames[ks] = this.context.HasDeltaKeyspace(ks)
}

func (this *builder) IndexKeyspaceNames() map[string]bool {
	return this.indexKeyspaceNames
}

func (this *builder) getTermKeyspace(node *algebra.KeyspaceTerm) (datastore.Keyspace, error) {
	path := node.Path()
	if path == nil {
		return nil, nil
	}
	path.SetDefaultNamespace(this.namespace)
	ns := strings.ToLower(path.Namespace())
	keyspace, err := datastore.GetKeyspace(path.Parts()...)

	if err != nil && this.indexAdvisor && !algebra.IsSystem(ns) &&
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

func (this *builder) getDocCount(alias string) (docCount int64) {
	baseKeyspace, ok := this.baseKeyspaces[alias]
	if !ok {
		return
	}
	keyspace := baseKeyspace.Keyspace()
	if keyspace == "" {
		// not a keyspace
		return
	}

	if !baseKeyspace.HasDocCount() {
		baseKeyspace.SetDocCount(optDocCount(keyspace))
		baseKeyspace.SetHasDocCount()
	}

	docCount = baseKeyspace.DocCount()
	return
}

func (this *builder) keyspaceUseCBO(alias string) bool {
	docCount := this.getDocCount(alias)
	return docCount >= 0
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
