//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package inline

import (
	goerrors "errors"
	"strings"
	"sync"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/functions"
	functionsBridge "github.com/couchbase/query/functions/bridge"
	"github.com/couchbase/query/value"
)

type inline struct {
}

type inlineBody struct {
	expr          expression.Expression
	varNames      []string
	text          string
	subqueryPlans *algebra.SubqueryPlans // subquery plans
	mutex         sync.RWMutex           // mutex
}

func Init() {
	functions.FunctionsNewLanguage(functions.INLINE, &inline{})
}

func (this *inline) CheckAuthorize(name string, fcontext functions.Context) bool {
	if context, contextOk := fcontext.(functionsBridge.InlineUdfContext); contextOk {
		if _, _, ok := context.GetInlineUdf(name); ok {
			return false
		}
	}
	return true
}

func (this *inline) Execute(name functions.FunctionName, body functions.FunctionBody, modifiers functions.Modifier,
	values []value.Value, context functions.Context) (value.Value, errors.Error) {
	funcBody, ok := body.(*inlineBody)
	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed!"), name.Name())
	}

	// all subquery plans setup appropriate place
	var val value.Value
	expr, varNames, err := funcBody.GetPlans(name.Key(), context)
	if err == nil {
		var parent map[string]interface{}

		if varNames == nil {
			args := make([]value.Value, len(values))
			for i, _ := range values {
				args[i] = value.NewValue(values[i])
			}
			parent = map[string]interface{}{"args": args}
		} else {
			if len(values) != len(varNames) {
				return nil, errors.NewArgumentsMismatchError(name.Name())
			}
			parent = make(map[string]interface{}, len(values))
			for i, _ := range values {
				parent[varNames[i]] = values[i]
			}
		}

		val, err = expr.Evaluate(value.NewValue(parent), context)
	}
	if err != nil {
		return nil, errors.NewFunctionExecutionError("", name.Name(), err)
	}
	return val, nil
}

func NewInlineBody(expr expression.Expression, text string) (functions.FunctionBody, errors.Error) {
	return &inlineBody{expr: expr, text: strings.TrimSuffix(text, ";")}, nil
}

func (this *inlineBody) SetVarNames(vars []string) errors.Error {
	this.varNames = vars
	return setVarNames(this.expr, vars)
}

func setVarNames(expr expression.Expression, vars []string) errors.Error {
	var bindings expression.Bindings

	/* We do not have parameter values at this stage, so the binding is
	   done only to identify variables as variables and not formalize them
	   as fields. We use a dummy expression for that.
	   We also have to mark the variable as with aliases, ie predefined
	   values (which is what they are), and have the value descend to
	   children formalizers, so that subqueries is not mistakenly marked
	   as correlated
	*/
	c := expression.NewConstant("")
	if len(vars) == 0 {
		args := expression.NewSimpleBinding("args", c)
		args.SetStatic(true)
		bindings = expression.Bindings{args}
	} else {
		bindings = make(expression.Bindings, len(vars))
		i := 0
		for v, _ := range vars {
			bindings[i] = expression.NewSimpleBinding(vars[v], c)
			bindings[i].SetStatic(true)
			bindings[i].SetFuncVariable(true)
			i++
		}
	}

	f := expression.NewFunctionFormalizer("", nil)
	f.SetFuncVariable(bindings)
	_, err := expr.Accept(f)
	if err != nil {
		return errors.NewInternalFunctionError(err, "")
	}
	return nil
}

func (this *inlineBody) SetStorage(context functions.Context, path []string) errors.Error {
	return nil
}

func (this *inlineBody) Lang() functions.Language {
	return functions.INLINE
}

func (this *inlineBody) Body(object map[string]interface{}) {
	object["#language"] = "inline"
	object["expression"] = this.expr.String()
	if this.varNames != nil {
		vars := make([]value.Value, len(this.varNames))
		for v, _ := range this.varNames {
			vars[v] = value.NewValue(this.varNames[v])
		}
		object["parameters"] = vars
	}
	object["text"] = this.text
}

func (this *inlineBody) Indexable() value.Tristate {
	ix := this.expr.Indexable()
	if ix {
		return value.TRUE
	} else {
		return value.FALSE
	}
}

// inline only allows selects and all objects are already qualified
// both at the algebra and plan level:
// the subquery plan cache will never have conflicts for the same subquery across two Query Contexts,
// so no need to switch
func (this *inlineBody) SwitchContext() value.Tristate {
	return value.FALSE
}

func (this *inlineBody) IsExternal() bool {
	return false
}

func (this *inlineBody) Privileges() (*auth.Privileges, errors.Error) {
	subqueries, err := expression.ListSubqueries(expression.Expressions{this.expr}, false)
	if err != nil {
		return nil, errors.NewError(err, "")
	}

	privileges := auth.NewPrivileges()
	for _, s := range subqueries {
		sub := s.(*algebra.Subquery)
		sp, e := sub.Select().Privileges()
		if e != nil {
			return nil, e
		}

		privileges.AddAll(sp)
	}

	return privileges, nil
}

// subquery plans of expression (even no subquery, palns will be empty).

func (this *inlineBody) GetSubqueryPlans(lock bool) (expression.Expression, []string, *algebra.SubqueryPlans) {
	if lock {
		this.mutex.RLock()
		defer this.mutex.RUnlock()
	}
	return this.expr, this.varNames, this.subqueryPlans
}

// Generate or Setup Plans UDF when expression changed. Copy into local context by refrence once per statement execution.

func (this *inlineBody) GetPlans(udfName string, fcontext functions.Context) (expression.Expression, []string, error) {
	context, contextOk := fcontext.(functionsBridge.InlineUdfContext)
	if contextOk {
		// If UDF name already part of the context use that expression/Plans.
		// Avoids pick new udf body middle any change udf will not be picked up.
		if expr, varNames, ok := context.GetInlineUdf(udfName); ok {
			return expr, varNames, nil
		}
	} else {
		return nil, nil, errors.NewInternalFunctionError(goerrors.New("Inlineudf Context"), udfName)
	}

	var good, trans bool
	var err error
	var expr expression.Expression

	// Get UDF subquery plans
	udfExpr, varNames, subqueryPlans := this.GetSubqueryPlans(true)
	if subqueryPlans != nil {
		expr = subqueryPlans.GetExpression(true)
		// If already present verify it
		good, trans = context.VerifySubqueryPlans(expr, subqueryPlans, true)
	}
	if subqueryPlans == nil || !good {
		// If no plans or not valid
		this.mutex.Lock()
		if this.subqueryPlans == nil || subqueryPlans == this.subqueryPlans {
			expr, err = parser.Parse(udfExpr.String())
			if err == nil {
				err = setVarNames(expr, varNames)
			}
			if err != nil {
				this.mutex.Unlock()
				return nil, nil, err
			}
			//  Store new plan in UDF
			subqueryPlans = algebra.NewSubqueryPlans()
			err = context.SetupSubqueryPlans(expr, subqueryPlans, true, true, false)
			if err != nil {
				this.mutex.Unlock()
				return nil, nil, err
			}
			// reverify so that we can handle in transaction context
			// verify in lock so that metadata update will not be done in parallel
			_, trans = context.VerifySubqueryPlans(expr, subqueryPlans, true)
			this.subqueryPlans = subqueryPlans
			this.mutex.Unlock()
		} else {
			subqueryPlans = this.subqueryPlans
			expr = subqueryPlans.GetExpression(true)
			this.mutex.Unlock()
			// reverify so that we can handle in transaction context
			_, trans = context.VerifySubqueryPlans(expr, subqueryPlans, true)
		}
	}
	if trans {
		// In transaction context regenerate algebra tree
		expr, err = parser.Parse(udfExpr.String())
		if err == nil {
			err = setVarNames(expr, varNames)
		}
	}
	if err == nil {
		// Setup subquery plans for transaction or copy from UDF to context
		err = context.SetupSubqueryPlans(expr, subqueryPlans, true, false, trans)
	}
	if err == nil {
		// Store UDF information in the context
		context.SetInlineUdf(udfName, expr, varNames)
	}
	return expr, varNames, err
}
