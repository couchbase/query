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
	hasSubqueries bool // if the inline function body contains subqueries
}

func Init() {
	functions.FunctionsNewLanguage(functions.INLINE, &inline{})
}

// Returns all the queries in the inline UDF body
// If the function body: Is a subquery - return it
// If not: return all subqueries in the function body
func (this *inline) FunctionStatements(name functions.FunctionName, body functions.FunctionBody, context functions.Context) (interface{}, errors.Error) {
	funcBody, ok := body.(*inlineBody)

	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed!"), name.Name())
	}

	// Get the function body to safely use
	expr, err := funcBody.getBodyExpr()
	if err != nil {
		return nil, errors.NewInternalFunctionError(err, name.Name())
	}

	var subqueries [](*algebra.Subquery)

	if sq, ok := expr.(*algebra.Subquery); ok {
		subqueries = append(subqueries, sq)
	} else {
		sqs, err := expression.ListSubqueries(expression.Expressions{expr}, false)

		if err != nil {
			return nil, errors.NewInternalFunctionError(err, name.Name())
		}

		for _, s := range sqs {
			sub, ok := s.(*algebra.Subquery)

			if ok {
				subqueries = append(subqueries, sub)
			}
		}
	}

	return subqueries, nil
}

func (this *inline) Execute(name functions.FunctionName, body functions.FunctionBody, modifiers functions.Modifier, values []value.Value, context functions.Context) (value.Value, errors.Error) {
	var parent map[string]interface{}

	funcBody, ok := body.(*inlineBody)

	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed"), name.Name())
	}

	if funcBody.varNames == nil {
		args := make([]value.Value, len(values))
		for i, _ := range values {
			args[i] = value.NewValue(values[i])
		}
		parent = map[string]interface{}{"args": args}
	} else {
		if len(values) != len(funcBody.varNames) {
			return nil, errors.NewArgumentsMismatchError(name.Name())
		}
		parent = make(map[string]interface{}, len(values))
		for i, _ := range values {
			parent[funcBody.varNames[i]] = values[i]
		}
	}

	expr := funcBody.expr
	if c, ok := context.(functionsBridge.InlineUdfContext); ok {
		var err error

		// Get the function body to safely use
		// Generate and use a new AST expression object when the funcBody.expr contains subqueries
		expr, err = c.GetAndSetInlineUdfExprs(name.Key(), funcBody.expr, funcBody.hasSubqueries)
		if err != nil {
			return nil, errors.NewInternalFunctionError(err, name.Name())
		}
	}

	val, err := expr.Evaluate(value.NewValue(parent), context)

	if err != nil {
		return nil, errors.NewFunctionExecutionError("", name.Name(), err)
	} else {
		return val, nil
	}
}

func NewInlineBody(expr expression.Expression, text string) (functions.FunctionBody, errors.Error) {
	return &inlineBody{expr: expr, text: strings.TrimSuffix(text, ";"), hasSubqueries: expression.ContainsSubquery(expr)}, nil
}

func (this *inlineBody) SetVarNames(vars []string) errors.Error {
	var bindings expression.Bindings
	var f *expression.Formalizer

	this.varNames = vars

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
		f = expression.NewFormalizer("", nil)
	} else {
		bindings = make(expression.Bindings, len(vars))
		i := 0
		for v, _ := range vars {
			bindings[i] = expression.NewSimpleBinding(vars[v], c)
			bindings[i].SetStatic(true)
			i++
		}
		f = expression.NewFunctionFormalizer("", nil)
	}

	f.SetPermanentWiths(bindings)
	f.PushBindings(bindings, true)
	_, err := this.expr.Accept(f)
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
		return nil, errors.NewInternalFunctionError(err, "")
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

func (this *inlineBody) Test(name functions.FunctionName) errors.Error {
	return nil
}

func (this *inlineBody) Load(name functions.FunctionName) errors.Error {
	return nil
}

func (this *inlineBody) Unload(name functions.FunctionName) {
}

func (this *inlineBody) Expressions() expression.Expressions {
	return expression.Expressions{this.expr}
}

// MB-58479: Since the Inline UDF's body is shared
// Reparse the function body only when it contains subqueries
// to prevent race conditions and concurrent reads/writes on the shared Expression object
// If the function body has no subqueries it is safe to use the original shared Expression object
func (this *inlineBody) getBodyExpr() (expression.Expression, error) {
	if this.hasSubqueries {
		return parser.Parse(this.expr.String())
	}

	return this.expr, nil
}
