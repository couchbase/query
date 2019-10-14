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
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func Build(stmt algebra.Statement, datastore, systemstore datastore.Datastore,
	namespace string, subquery, stream bool, namedArgs map[string]value.Value,
	positionalArgs value.Values, indexApiVersion int, featureControls uint64) (
	plan.Operator, error) {

	// request id in planner is separate from request id in execution context
	requestId, err := util.UUIDV3()
	if err != nil {
		return nil, err
	}
	builder := newBuilder(datastore, systemstore, namespace, subquery, namedArgs, positionalArgs,
		indexApiVersion, featureControls, requestId)
	if distributed.RemoteAccess().Enabled(distributed.NEW_OPTIMIZER) && util.IsFeatureEnabled(featureControls, util.N1QL_CBO) {
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
			op = plan.NewSequence(op, plan.NewStream())
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
	datastore         datastore.Datastore
	systemstore       datastore.Datastore
	namespace         string
	indexApiVersion   int
	featureControls   uint64
	requestId         string
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
	namedArgs         map[string]value.Value
	positionalArgs    value.Values
	baseKeyspaces     map[string]*base.BaseKeyspace
	pushableOnclause  expression.Expression // combined ON-clause from all inner joins
	builderFlags      uint32
	indexAdvisor      bool
	useCBO            bool
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
	namedArgs map[string]value.Value, positionalArgs value.Values, indexApiVersion int,
	featureControls uint64, requestId string) *builder {
	rv := &builder{
		datastore:       datastore,
		systemstore:     systemstore,
		namespace:       namespace,
		subquery:        subquery,
		delayProjection: false,
		namedArgs:       namedArgs,
		positionalArgs:  positionalArgs,
		indexApiVersion: indexApiVersion,
		featureControls: featureControls,
		requestId:       requestId,
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

func (this *builder) getTermKeyspace(node *algebra.KeyspaceTerm) (datastore.Keyspace, error) {
	path := node.Path()
	path.SetDefaultNamespace(this.namespace)
	ns := path.Namespace()
	ds := this.datastore
	if strings.ToLower(ns) == "#system" {
		ds = this.systemstore
	}
	namespace, err := ds.NamespaceByName(ns)
	if err != nil {
		return nil, err
	}
	keyspace, err := getKeyspace(namespace, path)

	if err != nil && this.indexAdvisor && strings.ToLower(ns) != "#system" &&
		(strings.Contains(err.TranslationKey(), "bucket_not_found") ||
			strings.Contains(err.TranslationKey(), "scope_not_found") ||
			strings.Contains(err.TranslationKey(), "keyspace_not_found")) {
		if v, ok := namespace.(datastore.VirtualNamespace); ok {
			return v.VirtualKeyspaceByName(path.Keyspace())
		}
	}

	return keyspace, err
}

func getKeyspace(namespace datastore.Namespace, path *algebra.Path) (datastore.Keyspace, errors.Error) {
	if path.IsCollection() {
		bucket, err := namespace.BucketByName(path.Bucket())
		if err != nil {
			return nil, err
		}
		scope, err := bucket.ScopeByName(path.Scope())
		if err != nil {
			return nil, err
		}
		return scope.KeyspaceByName(path.Keyspace())
	}
	return namespace.KeyspaceByName(path.Keyspace())
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
