//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package inline

import (
	goerrors "errors"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

type inline struct {
}

type inlineBody struct {
	expr     expression.Expression
	varNames []string
}

func Init() {
	functions.FunctionsNewLanguage(functions.INLINE, &inline{})
}

func (this *inline) Execute(name functions.FunctionName, body functions.FunctionBody, modifiers functions.Modifier, values []value.Value, context functions.Context) (value.Value, errors.Error) {
	var parent map[string]interface{}

	funcBody, ok := body.(*inlineBody)

	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed!"), name.Name())
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
	val, err := funcBody.expr.Evaluate(value.NewValue(parent), context)
	if err != nil {
		return nil, errors.NewFunctionExecutionError("", name.Name(), err)
	} else {
		return val, nil
	}
}

func NewInlineBody(expr expression.Expression) (functions.FunctionBody, errors.Error) {
	return &inlineBody{expr: expr}, nil
}

func (this *inlineBody) SetVarNames(vars []string) errors.Error {
	var bindings expression.Bindings

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
	if vars == nil {
		args := expression.NewSimpleBinding("args", c)
		args.SetStatic(true)
		bindings = expression.Bindings{args}
	} else {
		bindings = make(expression.Bindings, len(vars))
		i := 0
		for v, _ := range vars {
			bindings[i] = expression.NewSimpleBinding(vars[v], c)
			bindings[i].SetStatic(true)
			i++
		}
	}

	f := expression.NewFormalizer("", nil)
	f.SetPermanentWiths(bindings)
	f.PushBindings(bindings, true)
	_, err := this.expr.Accept(f)
	if err != nil {
		return errors.NewInternalFunctionError(err, "")
	}
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
}

func (this *inlineBody) Indexable() value.Tristate {
	ix := this.expr.Indexable()
	if ix {
		return value.TRUE
	} else {
		return value.FALSE
	}
}

// inline only allows selects and the keyspaces are already qualified
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
