//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"strings"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
)

func Build(stmt algebra.Statement, datastore, systemstore datastore.Datastore,
	namespace string, subquery, stream bool, forceSQBuild bool, context *PrepareContext) (
	*plan.QueryPlan, map[string]bool, error, map[string]time.Duration) {

	builder := newBuilder(datastore, systemstore, namespace, subquery, context)
	if context.UseCBO() && context.Optimizer() != nil {
		builder.useCBO = true
		checkCostModel(context.FeatureControls())
	}

	// subquery plan is currently only for explain, explain_function and advise
	// TODO: to be expanded to all statements, plus prepareds
	// forceSQBuild argument  forces subquery plans to be built
	if stmt.Type() == "EXPLAIN" || stmt.Type() == "EXPLAIN_FUNCTION" || stmt.Type() == "ADVISE" {
		builder.setBuilderFlag(BUILDER_PLAN_SUBQUERY | BUILDER_NO_EXECUTE)
	} else if forceSQBuild {
		builder.setBuilderFlag(BUILDER_PLAN_SUBQUERY)
	}

	p, err := stmt.Accept(builder)

	if err != nil {
		return nil, nil, err, builder.subTimes
	}

	qp := p.(*plan.QueryPlan)
	op := qp.PlanOp()
	_, is_prepared := op.(*plan.Prepared)
	indexKeyspaces := builder.indexKeyspaceNames

	if !subquery && !is_prepared {
		privs, err := stmt.Privileges()
		if err != nil {
			return nil, nil, err, builder.subTimes
		}

		if stream {

			// Do not serialize Stream Op when the stmt is MERGE
			// Because in MERGE - with Serialization of Stream - whichever child DML op finishes first, notifies Stream
			// And can cause early termination
			serializable := stmt.Type() != "MERGE"

			op = plan.NewSequence(op, plan.NewStream(op.Cost(), op.Cardinality(), op.Size(), op.FrCost(), serializable))
		}

		getSeqScanPrivs(op, privs)

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
		qp.SetPlanOp(plan.NewAuthorize(privs, op))
	} else {
		privs := auth.NewPrivileges()
		getSeqScanPrivs(op, privs)
		if len(privs.List) > 0 {
			qp.SetExtraPrivs(privs)
		} else {
			qp.SetExtraPrivs(nil)
		}
	}

	return qp, indexKeyspaces, nil, builder.subTimes
}

func (this *builder) chkBldSubqueries(stmt algebra.Statement, qp *plan.QueryPlan) (err error) {
	if this.hasBuilderFlag(BUILDER_PLAN_SUBQUERY) {
		err = this.buildSubqueries(stmt, qp)
		// once plans for subqueries are built, unset builder flag
		if !this.subquery {
			this.unsetBuilderFlag(BUILDER_PLAN_SUBQUERY)
		}
	}
	return
}

func (this *builder) buildSubqueries(stmt algebra.Statement, qp *plan.QueryPlan) error {
	subqueries, er := stmt.Subqueries()
	if er != nil {
		return er
	}
	if len(subqueries) == 0 {
		return nil
	}
	var saveQInfo *saveQueryInfo
	subquery := this.subquery
	if this.indexAdvisor {
		saveQInfo = this.saveQueryInfo()
	}
	defer func() {
		this.subquery = subquery
		if this.indexAdvisor {
			this.restoreQueryInfo(saveQInfo)
		}
	}()
	this.subquery = true
	this.makeSubqueryInfos(len(subqueries))
	for _, s := range subqueries {
		subq := s.Select()
		if qp.HasSubquery(subq) {
			continue
		}

		if this.indexAdvisor {
			this.startSubqIndexAdvisor()
			this.initialIndexAdvisor(subq)
		}

		// be warned, this amends the AST for the subqueries
		p, err := subq.Accept(this)
		if err == nil {
			qplan := p.(*plan.QueryPlan)
			qp.AddSubquery(subq, qplan.PlanOp())
			for s, o := range qplan.Subqueries() {
				if !qp.HasSubquery(s) {
					qp.AddSubquery(s, o)
				}
			}
		} else if !this.indexAdvisor {
			return err
		}
		if this.indexAdvisor {
			this.endSubqIndexAdvisor(subq)
		}
	}
	return nil
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
	BUILDER_HAS_OFFSET // OFFSET clause is present
	BUILDER_HAS_GROUP
	BUILDER_HAS_ORDER
	BUILDER_HAS_WINDOW_AGGS
	BUILDER_JOIN_ENUM
	BUILDER_CHK_INDEX_ORDER
	BUILDER_PLAN_HAS_ORDER
	BUILDER_HAS_EXTRA_FLTR
	BUILDER_DO_JOIN_FILTER
	BUILDER_OR_SUBTERM
	BUILDER_HAS_EARLY_ORDER
	BUILDER_PLAN_SUBQUERY
	BUILDER_ORDER_DEPENDS_ON_LET
	BUILDER_JOIN_ON_PRIMARY
	BUILDER_NL_INNER
	BUILDER_OFFSET_PUSHDOWN // OFFSET is pushed down to index
	BUILDER_UNDER_HASH
	BUILDER_ORDER_MODIFIED
	BUILDER_SUBQTERM_UNDER_JOIN
	BUILDER_SUBQTERM_IN_JOIN_ENUM
	BUILDER_NO_EXECUTE
	BUILDER_WHERE_DEPENDS_ON_LET
)

const BUILDER_PRESERVED_FLAGS = (BUILDER_PLAN_HAS_ORDER | BUILDER_HAS_EARLY_ORDER | BUILDER_ORDER_MODIFIED)
const BUILDER_PASSTHRU_FLAGS = (BUILDER_PLAN_SUBQUERY | BUILDER_SUBQTERM_UNDER_JOIN | BUILDER_SUBQTERM_IN_JOIN_ENUM | BUILDER_NO_EXECUTE)

type builder struct {
	indexPushDowns
	collectQueryInfo
	context              *PrepareContext
	datastore            datastore.Datastore
	systemstore          datastore.Datastore
	namespace            string
	subquery             bool
	correlated           bool
	maxParallelism       int
	delayProjection      bool                  // Used to allow ORDER BY non-projected expressions
	from                 algebra.FromTerm      // Used for index selection
	where                expression.Expression // Used for index selection
	filter               expression.Expression // for Filter operator
	let                  expression.Bindings   // LET clause
	letLevel             int                   // max level of LET bindings
	setOpDistinct        bool                  // Used for SETOP Distinct to apply DISTINCT on projection
	children             []plan.Operator
	subChildren          []plan.Operator
	cover                expression.HasExpressions
	node                 expression.HasExpressions
	coveringScans        []plan.CoveringOperator
	coveredUnnests       map[*algebra.Unnest]bool
	countScan            plan.CoveringOperator
	skipDynamic          bool
	requirePrimaryKey    bool
	orderScan            plan.SecondaryScan
	baseKeyspaces        map[string]*base.BaseKeyspace
	keyspaceNames        map[string]string
	indexKeyspaceNames   map[string]bool       // keyspace names that use indexscan (excludes non from caluse subqueries)
	pushableOnclause     expression.Expression // combined ON-clause from all inner joins
	builderFlags         uint64
	indexAdvisor         bool
	useCBO               bool
	hintIndexes          bool
	lastOp               plan.Operator // last operator built, to get cost/cardinality info
	aliases              map[string]bool
	partialSortTermCount int
	stmtOrder            *algebra.Order // modified Order
	skipKeyspace         string
	mustSkipKeys         bool
	subTimes             map[string]time.Duration
	arrayId              int
	subqCoveringInfo     map[*algebra.Subselect]CoveringSubqInfo
}

func (this *builder) Copy() *builder {
	rv := &builder{
		context:              this.context,
		datastore:            this.datastore,
		systemstore:          this.systemstore,
		namespace:            this.namespace,
		subquery:             this.subquery,
		correlated:           this.correlated,
		maxParallelism:       this.maxParallelism,
		delayProjection:      this.delayProjection,
		from:                 this.from,
		where:                expression.Copy(this.where),
		filter:               expression.Copy(this.filter),
		setOpDistinct:        this.setOpDistinct,
		cover:                this.cover,
		node:                 this.node,
		skipDynamic:          this.skipDynamic,
		requirePrimaryKey:    this.requirePrimaryKey,
		baseKeyspaces:        base.CopyBaseKeyspacesWithFilters(this.baseKeyspaces),
		pushableOnclause:     expression.Copy(this.pushableOnclause),
		builderFlags:         this.builderFlags,
		indexAdvisor:         this.indexAdvisor,
		useCBO:               this.useCBO,
		hintIndexes:          this.hintIndexes,
		partialSortTermCount: this.partialSortTermCount,
		arrayId:              this.arrayId,
		// the following fields are setup during planning process and thus not copied:
		// children, subChildren, coveringScan, coveredUnnests, countScan, orderScan, lastOp
		// subqCoveringInfo
	}

	if len(this.let) > 0 {
		rv.let = this.let.Copy()
		rv.letLevel = this.letLevel
	}
	if len(this.keyspaceNames) > 0 {
		rv.keyspaceNames = make(map[string]string, len(this.keyspaceNames))
		for k, v := range this.keyspaceNames {
			rv.keyspaceNames[k] = v
		}
	}

	if len(this.aliases) > 0 {
		rv.aliases = make(map[string]bool, len(this.aliases))
		for k, v := range this.aliases {
			rv.aliases[k] = v
		}
	}

	// indexKeyspaceNames is always allocated
	rv.indexKeyspaceNames = make(map[string]bool, len(this.indexKeyspaceNames))
	for k, v := range this.indexKeyspaceNames {
		rv.indexKeyspaceNames[k] = v
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

func (this *builder) setNLInner() bool {
	nlInner := this.hasBuilderFlag(BUILDER_NL_INNER)
	if !nlInner {
		this.setBuilderFlag(BUILDER_NL_INNER)
	}
	return nlInner
}

func (this *builder) restoreNLInner(nlInner bool) {
	if !nlInner {
		this.unsetBuilderFlag(BUILDER_NL_INNER)
	}
}

func (this *builder) subqUnderJoin() bool {
	return (this.builderFlags & BUILDER_SUBQTERM_UNDER_JOIN) != 0
}

func (this *builder) setSubqUnderJoin() bool {
	subqUnderJoin := this.hasBuilderFlag(BUILDER_SUBQTERM_UNDER_JOIN)
	if !subqUnderJoin {
		this.setBuilderFlag(BUILDER_SUBQTERM_UNDER_JOIN)
	}
	return subqUnderJoin
}

func (this *builder) restoreSubqUnderJoin(subqUnderJoin bool) {
	if !subqUnderJoin {
		this.unsetBuilderFlag(BUILDER_SUBQTERM_UNDER_JOIN)
	}
}

func (this *builder) subqInJoinEnum() bool {
	return (this.builderFlags & BUILDER_SUBQTERM_IN_JOIN_ENUM) != 0
}

func (this *builder) setSubqInJoinEnum() bool {
	subqInJoinEnum := this.hasBuilderFlag(BUILDER_SUBQTERM_IN_JOIN_ENUM)
	if !subqInJoinEnum {
		this.setBuilderFlag(BUILDER_SUBQTERM_IN_JOIN_ENUM)
	}
	return subqInJoinEnum
}

func (this *builder) restoreSubqInJoinEnum(subqInJoinEnum bool) {
	if !subqInJoinEnum {
		this.unsetBuilderFlag(BUILDER_SUBQTERM_IN_JOIN_ENUM)
	}
}

func (this *builder) hasBuilderFlag(flag uint64) bool {
	return (this.builderFlags & flag) != 0
}

func (this *builder) setBuilderFlag(flag uint64) {
	this.builderFlags |= flag
}

func (this *builder) unsetBuilderFlag(flag uint64) {
	this.builderFlags &^= flag
}

func (this *builder) resetBuilderFlags(prevBuilderFlags uint64) {
	preservedFlags := (this.builderFlags & BUILDER_PRESERVED_FLAGS)
	this.builderFlags = prevBuilderFlags | preservedFlags
}

func (this *builder) passthruBuilderFlags(prevBuilderFlags uint64) {
	this.builderFlags = (prevBuilderFlags & BUILDER_PASSTHRU_FLAGS)
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
	start := util.Now()
	keyspace, err := datastore.GetKeyspace(path.Parts()...)
	this.recordSubTime("keyspace.metadata", util.Since(start))

	if err != nil && this.indexAdvisor && !algebra.IsSystemId(ns) &&
		(strings.Contains(err.TranslationKey(), "bucket_not_found") ||
			strings.Contains(err.TranslationKey(), "scope_not_found") ||
			strings.Contains(err.TranslationKey(), "keyspace_not_found")) {

		virtualKeyspace, err1 := this.getVirtualKeyspace(ns, path.Parts())
		if err1 == nil {
			return virtualKeyspace, nil
		}
	}

	if err != nil {
		parts := path.Parts()
		err2 := datastore.CheckBucketAccess(this.context.Credentials(), err, parts)

		if err2 != nil {
			return keyspace, err2
		}
	}

	if err == nil && this.indexAdvisor {
		this.setKeyspaceFound()
	} else if err != nil && (err.Code() == errors.E_CB_KEYSPACE_NOT_FOUND ||
		err.Code() == errors.E_CB_BUCKET_NOT_FOUND || err.Code() == errors.E_CB_SCOPE_NOT_FOUND) {

		err.AddErrorContext(node.ErrorContext())
	}

	return keyspace, err
}

func (this *builder) getDocCount(alias string) (docCount int64) {
	baseKeyspace, ok := this.baseKeyspaces[alias]
	if !ok || baseKeyspace.IsSystem() {
		return -1
	}
	keyspace := baseKeyspace.Keyspace()
	if keyspace == "" {
		// not a keyspace
		return -1
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

func (this *builder) recordSubTime(what string, duration time.Duration) {
	if this.subTimes == nil {
		this.subTimes = make(map[string]time.Duration)
	}
	if existing, ok := this.subTimes[what]; ok {
		duration += existing
	}
	this.subTimes[what] = duration
}

const (
	_SUBQPLAN_JOIN_ENUM = 1 << iota
	_SUBQPLAN_UNDER_JOIN
)

type subqTermPlan struct {
	op    plan.Operator
	flags int32
}

func (this *subqTermPlan) Operator() plan.Operator {
	return this.op
}

func (this *subqTermPlan) IsJoinEnum() bool {
	return (this.flags & _SUBQPLAN_JOIN_ENUM) != 0
}

func (this *subqTermPlan) IsUnderJoin() bool {
	return (this.flags & _SUBQPLAN_UNDER_JOIN) != 0
}

type coveringSubqInfo struct {
	subqTermPlans []*subqTermPlan
	coveringScans []plan.CoveringOperator
	subqueryTerms []*algebra.SubqueryTerm
}

func (this *coveringSubqInfo) SubqTermPlans() []*subqTermPlan {
	return this.subqTermPlans
}

func (this *coveringSubqInfo) AddSubqTermPlan(subqPlan *subqTermPlan) {
	this.subqTermPlans = append(this.subqTermPlans, subqPlan)
}

func (this *coveringSubqInfo) CoveringScans() []plan.CoveringOperator {
	return this.coveringScans
}

func (this *coveringSubqInfo) SubqueryTerms() []*algebra.SubqueryTerm {
	return this.subqueryTerms
}

func (this *coveringSubqInfo) addSubqueryTerm(subqTerm *algebra.SubqueryTerm) {
	this.subqueryTerms = append(this.subqueryTerms, subqTerm)
}

func (this *builder) addSubqCoveringInfo(node *algebra.Subselect, op plan.Operator) error {
	flags := int32(0)
	if this.subqUnderJoin() {
		if !this.NoExecute() {
			// no need to do cover transformation, wait till runtime
			return nil
		}
		flags |= _SUBQPLAN_UNDER_JOIN
	} else if this.subqInJoinEnum() {
		// only set JOIN_ENUM flag if UNDER_JOIN is not on (follow UNDER_JOIN first)
		flags |= _SUBQPLAN_JOIN_ENUM

		// if it is not UNDER_JOIN, then we got here when planning a SubqueryTerm during
		// join enumeration, in which case we need to mark keyspace hints (index and join hints)
		err := this.MarkKeyspaceHints()
		if err != nil {
			return err
		}
	} else {
		// no need to do cover transformation
		return nil
	}

	if this.subqCoveringInfo == nil {
		this.subqCoveringInfo = make(map[*algebra.Subselect]CoveringSubqInfo, len(this.baseKeyspaces))
	}

	// if there are SubqueryTerms, add those too such that when we do cover transformation
	// at the end of join enumeration we can find the nested subqueries
	var subqTerms []*algebra.SubqueryTerm
	for _, ks := range this.baseKeyspaces {
		if subqTerm, ok := ks.Node().(*algebra.SubqueryTerm); ok {
			subqTerms = append(subqTerms, subqTerm)
		}
	}

	subqTermPlan := &subqTermPlan{
		op:    op,
		flags: flags,
	}
	if info, ok := this.subqCoveringInfo[node]; ok {
		if len(this.coveringScans) != len(info.CoveringScans()) {
			return errors.NewPlanInternalError("addSubqCoveringInfo: incompatible coveringScans")
		} else if len(subqTerms) != len(info.SubqueryTerms()) {
			return errors.NewPlanInternalError("addSubqCoveringInfo: incompatible nested SubqueryTerms")
		}
	} else {
		this.subqCoveringInfo[node] = &coveringSubqInfo{
			coveringScans: this.coveringScans,
			subqueryTerms: subqTerms,
		}
	}
	this.subqCoveringInfo[node].AddSubqTermPlan(subqTermPlan)
	return nil
}

// Inherit the subqCoveringInfo entries that's marked as SubqUnderJoin from builderCopy (used during
// join enumeration) to builder.
func (this *builder) inheritSubqCoveringInfo(builderCopy *builder) {
	for subselect, infoCopy := range builderCopy.subqCoveringInfo {
		for _, subqTermPlan := range infoCopy.SubqTermPlans() {
			if subqTermPlan.IsUnderJoin() {
				info, ok := this.subqCoveringInfo[subselect]
				if !ok {
					if this.subqCoveringInfo == nil {
						this.subqCoveringInfo = make(map[*algebra.Subselect]CoveringSubqInfo,
							len(builderCopy.subqCoveringInfo))
					}
					info = &coveringSubqInfo{
						coveringScans: infoCopy.CoveringScans(),
						subqueryTerms: infoCopy.SubqueryTerms(),
					}
					this.subqCoveringInfo[subselect] = info
				}
				info.AddSubqTermPlan(subqTermPlan)
			}
		}
	}
}
