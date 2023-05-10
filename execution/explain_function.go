//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/value"
)

type ExplainFunction struct {
	base
	plan  *plan.ExplainFunction
	plans map[string]planEntry
}

func NewExplainFunction(plan *plan.ExplainFunction, context *Context) *ExplainFunction {
	rv := &ExplainFunction{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *ExplainFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExplainFunction(this)
}

func (this *ExplainFunction) Copy() Operator {
	rv := &ExplainFunction{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *ExplainFunction) PlanOp() plan.Operator {
	return this.plan
}

func (this *ExplainFunction) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped
		if !active {
			return
		}

		lang, stmts, err := functions.FunctionStatements(this.plan.FuncName(), context.Credentials())

		if err != nil {
			context.Fatal(err)
			return
		}

		if stmts != nil {
			if lang == functions.INLINE {
				this.plans, err = createSQPlans(stmts, context)

				if err != nil {
					context.Fatal(err)
					return
				}
			} else if lang == functions.JAVASCRIPT {
				this.plans, err = createStmtPlans(stmts, context)

				if err != nil {
					context.Fatal(err)
					return
				}
			}
		}

		bytes, errM := this.marshalPlans()
		if errM != nil {
			context.Fatal(errors.NewExplainFunctionError(err, "EXPLAIN FUNCTION: Error marshaling JSON plans."))
			return
		}

		av := value.NewAnnotatedValue(bytes)
		if context.UseRequestQuota() {
			err := context.TrackValueSize(av.Size())
			if err != nil {
				context.Error(err)
				av.Recycle()
				return
			}
		}
		this.sendItem(av)

	})
}

func (this *ExplainFunction) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *ExplainFunction) Done() {
	this.baseDone()
	this.plan = nil
	this.plans = nil
}

type planEntry struct {
	qPlan      *plan.QueryPlan
	optimHints *algebra.OptimHints
}

func (this *ExplainFunction) marshalPlans() ([]byte, error) {
	r := make(map[string]interface{}, 2)
	r["function"] = this.plan.FuncName().Key()

	if len(this.plans) > 0 {
		marshalledPlans := make([]map[string]interface{}, 0, len(this.plans))

		for stmt, pe := range this.plans {

			if pe.qPlan != nil {
				op := pe.qPlan.PlanOp()

				query := map[string]interface{}{
					"statement": stmt,
					"plan":      op,
				}

				if op != nil {
					if op.Cost() > 0.0 {
						query["cost"] = op.Cost()
					}
					if op.Cardinality() > 0.0 {
						query["cardinality"] = op.Cardinality()
					}
				}

				if pe.optimHints != nil {
					query["optimizer_hints"] = pe.optimHints
				}

				subqueries := pe.qPlan.Subqueries()

				if len(subqueries) > 0 {

					marshalledSubqueries := make([]map[string]interface{}, 0, len(subqueries))
					for k, v := range subqueries {

						subquery := map[string]interface{}{
							"subquery": k.String(),
							"plan":     v,
						}

						optimHints := k.OptimHints()
						if optimHints != nil {
							subquery["optimizer_hints"] = optimHints
						}

						marshalledSubqueries = append(marshalledSubqueries, subquery)
					}

					query["~subqueries"] = marshalledSubqueries
				}

				marshalledPlans = append(marshalledPlans, query)

			} else {
				query := map[string]interface{}{
					"statement": stmt,
					"plan":      "EXPLAIN is not supported for statements of type ADVISE, EXPLAIN, EXECUTE",
				}

				marshalledPlans = append(marshalledPlans, query)
			}
		}

		r["plans"] = marshalledPlans
	}

	return json.Marshal(r)
}

// Returns a list of subquery plans
// Since Authorization check for Inline functions' inner subqueries is already done in FunctionStatements()
// We do not perform another Authorize() check
func createSQPlans(stmts []interface{}, context *Context) (map[string]planEntry, errors.Error) {
	var prepContext planner.PrepareContext
	plans := make(map[string]planEntry, len(stmts))

	planner.NewPrepareContext(&prepContext, context.requestId, context.queryContext, context.namedArgs,
		context.positionalArgs, context.indexApiVersion, context.featureControls, context.useFts, context.useCBO,
		context.optimizer, context.deltaKeyspaces, context, false)

	for _, stmt := range stmts {
		if s, ok := stmt.(*algebra.Subquery); ok {

			statement := s.Select().String()
			if _, ok := plans[statement]; !ok {

				// Build the query plan
				// Set ForceSQBuild = true - so that subquery plans are built as well
				qp, _, err, _ := planner.Build(s.Select(), context.datastore, context.systemstore, context.namespace, true, false, true, &prepContext)

				if err != nil {
					return nil, errors.NewExplainFunctionError(err, fmt.Sprintf("EXPLAIN FUNCTION: Error building query plan for statement %s", statement))
				}

				pe := planEntry{qPlan: qp, optimHints: s.Select().OptimHints()}
				plans[statement] = pe
			}
		}
	}

	return plans, nil
}

// Returns a list of plans given list of queries as strings
func createStmtPlans(stmts []interface{}, context *Context) (map[string]planEntry, errors.Error) {

	ds := datastore.GetDatastore()
	creds := context.Credentials()
	plans := make(map[string]planEntry, len(stmts))

	for _, stmt := range stmts {
		if s, ok := stmt.(string); ok {

			if _, ok := plans[s]; !ok {
				canExplain, ast, qp, err := context.ExplainStatement(s, context.namedArgs, context.positionalArgs, false)

				if err != nil {
					return nil, errors.NewExplainFunctionError(err, fmt.Sprintf("EXPLAIN FUNCTION: Error building query plan for statement %s", s))
				}

				// If Explain is disabled on the statement - ADVISE, EXPLAIN, EXECUTE
				if !canExplain {
					pe := planEntry{}
					plans[s] = pe
					continue
				}

				privs, errP := ast.Privileges()
				if errP != nil {
					return nil, errP
				}

				// Verify the privileges needed for this individual query
				errA := ds.Authorize(privs, creds)
				if errA != nil {
					errA = datastore.HandleDsAuthError(errA, privs, creds)
					return nil, errA
				}

				pe := planEntry{qPlan: qp, optimHints: ast.OptimHints()}
				plans[s] = pe
			}
		}
	}

	return plans, nil
}
