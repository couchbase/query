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
	plan *plan.ExplainFunction

	// Map of plan information
	// Key: query statement
	plans map[string]*planEntry

	// line numbers of Dynamic N1QL queries inside a JS UDF
	dynamicLineNos []uint
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

		lang, stmts, err := functions.FunctionStatements(this.plan.FuncName(), context.Credentials(), &this.operatorCtx)

		if err != nil {
			context.Error(err)
			return
		}

		if lang == functions.INLINE {
			// Inline function subquery plans already part the Context use them
			subqPlans := context.GetSubqueryPlans(false)
			if subqPlans != nil {
				this.plans = make(map[string]*planEntry)
				verifyF := func(key *algebra.Select, options uint32, splan, isk interface{}) (bool, bool) {
					if qp, ok := splan.(*plan.QueryPlan); ok {
						this.plans[key.String()] = &planEntry{qPlan: qp, optimHints: key.OptimHints(), uses: 1}
					}
					return true, false
				}
				subqPlans.ForEach(nil, uint32(0), true, verifyF)
			}
		} else if stmts != nil {
			if lang == functions.JAVASCRIPT {
				qs, _ := stmts.(map[string]interface{})
				stmtStrings, ok := qs["embedded"].([]string)

				if ok {
					this.plans, err = createStmtPlans(stmtStrings, context)

					if err != nil {
						context.Error(err)
						return
					}
				}

				if dl, ok := qs["dynamic"].([]uint); ok {
					this.dynamicLineNos = dl
				}
			} else {
				context.Error(errors.NewExplainFunctionError(nil, "Not supported for this function."))
				return
			}
		}

		bytes, errM := this.marshalPlans()
		if errM != nil {
			context.Error(errors.NewExplainFunctionError(err, "Error marshaling JSON plans."))
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
	uses       int    // number of times the statement was seen in the function
	extraInfo  string // optional extra info to return in the marshalled output
}

func (this *ExplainFunction) marshalPlans() ([]byte, error) {
	r := make(map[string]interface{}, 3)
	r["function"] = this.plan.FuncName().Key()

	if len(this.dynamicLineNos) > 0 {
		r["line_numbers"] = this.dynamicLineNos
	}

	if len(this.plans) > 0 {
		marshalledPlans := make([]map[string]interface{}, 0, len(this.plans))

		for stmt, pe := range this.plans {

			query := map[string]interface{}{
				"statement": stmt,
				"uses":      pe.uses,
			}

			if pe.qPlan != nil {
				op := pe.qPlan.PlanOp()
				query["plan"] = op

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
							"subquery":   k.String(),
							"plan":       v,
							"correlated": k.IsCorrelated(),
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

				if pe.extraInfo != "" {
					query["plan"] = pe.extraInfo
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
func createSQPlans(stmts []*algebra.Subquery, context *Context) (map[string]*planEntry, errors.Error) {
	var prepContext planner.PrepareContext
	plans := make(map[string]*planEntry, len(stmts))

	planner.NewPrepareContext(&prepContext, context.requestId, context.queryContext, context.namedArgs,
		context.positionalArgs, context.indexApiVersion, context.featureControls, context.useFts, context.useCBO,
		context.optimizer, context.deltaKeyspaces, context, false)

	for _, s := range stmts {

		statement := s.Select().String()
		entry, ok := plans[statement]

		if !ok {

			// Build the query plan
			// Set ForceSQBuild = true - so that subquery plans are built as well
			qp, _, err, _ := planner.Build(s.Select(), context.datastore, context.systemstore, context.namespace, true, false, true, &prepContext)

			if err != nil {
				return nil, errors.NewExplainFunctionError(err, fmt.Sprintf("Error building query plan for statement- %s", statement))
			}

			pe := planEntry{qPlan: qp, optimHints: s.Select().OptimHints(), uses: 1}
			plans[statement] = &pe
		} else {
			// if a query is in the function > 1 times - do not regenerate its plan information
			// just increment usage data
			entry.uses++
		}
	}

	return plans, nil
}

// Returns a list of plans given list of queries as strings
func createStmtPlans(stmts []string, context *Context) (map[string]*planEntry, errors.Error) {
	ds := datastore.GetDatastore()
	creds := context.Credentials()
	plans := make(map[string]*planEntry, len(stmts))

	for _, s := range stmts {
		entry, ok := plans[s]

		if !ok {
			// the values of the statement's named or positional arguments are not passed
			// since js-evaluator cannot return said values when query requests for all statements inside a UDF
			canExplain, ast, qp, err := context.ExplainStatement(s, nil, nil, false)

			if err != nil {
				return nil, errors.NewExplainFunctionError(err, fmt.Sprintf("Error building query plan for statement- %s", s))
			}

			// If the statement cannot be Explained
			// But no error was generated
			// Create an entry to be marshalled - instead of ignoring the statement
			if !canExplain {
				pe := planEntry{uses: 1}

				if ast != nil {
					// Explain is disabled on - ADVISE, EXPLAIN, EXECUTE queries
					pe.extraInfo = fmt.Sprintf("EXPLAIN is not supported on queries of type %s", ast.Type())
				}

				plans[s] = &pe
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

			pe := planEntry{qPlan: qp, optimHints: ast.OptimHints(), uses: 1}
			plans[s] = &pe

		} else {
			// if a query is in the function > 1 times - do not regenerate its plan information
			// just increment usage data
			entry.uses++
		}
	}

	return plans, nil
}
