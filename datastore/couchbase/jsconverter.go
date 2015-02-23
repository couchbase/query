//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package couchbase

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"github.com/couchbase/query/expression"
	stack "github.com/couchbase/query/util"
	"io"
	"strings"
)

type JSConverter struct {
	stack stack.Stack
}

type funcExpr struct {
	name     string
	operands *list.List
}

func writeOperands(operands *list.List) string {
	var buf bytes.Buffer
	for e := operands.Front(); e != nil; e = e.Next() {
		jsc := NewJSConverter()
		buf.WriteString(jsc.Visit(e.Value.(expression.Expression)))
		if e.Next() != nil {
			buf.WriteString(",")
		}
	}
	buf.WriteString(")")
	return buf.String()

}

func NewJSConverter() *JSConverter {
	return &JSConverter{stack: stack.Stack{}}
}

func (this *JSConverter) Visit(expr expression.Expression) string {
	var buf bytes.Buffer
	s, err := expr.Accept(this)
	if err != nil {
		panic(fmt.Sprintf("Unexpected error in JSConverter: %v", err))
	}

	switch s := s.(type) {
	case []byte:
		buf.WriteString(string(s))
		for this.stack.Size() != 0 {
			funcExpr := this.stack.Pop().(*funcExpr)
			buf.WriteString(funcExpr.name)
			if funcExpr.operands.Front() != nil {
				buf.WriteString(writeOperands(funcExpr.operands))
			}
		}

	default:
		buf.WriteString(s.(string))
	}

	// if the stack is not empty, pop the function
	/*
		for this.stack.Size() != 0 {
			funcExpr := this.stack.Pop().(*funcExpr)
			buf.WriteString(funcExpr.name)
			if funcExpr.operands.Front() != nil {
				buf.WriteString(writeOperands(funcExpr.operands))
			}
		}
	*/

	return buf.String()
}

// Arithmetic

func (this *JSConverter) VisitAdd(expr *expression.Add) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	operands := expr.Operands()
	for i, op := range operands {
		if i > 0 {
			buf.WriteString(" + ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitDiv(expr *expression.Div) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" / ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitMod(expr *expression.Mod) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" % ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitMult(expr *expression.Mult) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	operands := expr.Operands()
	for i, op := range operands {
		if i > 0 {
			buf.WriteString(" * ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitNeg(expr *expression.Neg) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(-")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitSub(expr *expression.Sub) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" - ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

// Case

func (this *JSConverter) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return nil, fmt.Errorf("Expression not implemented")
}

func (this *JSConverter) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {

	return nil, fmt.Errorf("Expression not implemented")
}

// Collection

func (this *JSConverter) VisitAny(expr *expression.Any) (interface{}, error) {
	return nil, fmt.Errorf("Expression not implemented")
}

func (this *JSConverter) VisitArray(expr *expression.Array) (interface{}, error) {
	return nil, fmt.Errorf("Expression not implemented")
}

func (this *JSConverter) VisitEvery(expr *expression.Every) (interface{}, error) {
	return nil, fmt.Errorf("Expression not implemented")
}

func (this *JSConverter) VisitExists(expr *expression.Exists) (interface{}, error) {
	return nil, fmt.Errorf("Expression not implemented")
}

func (this *JSConverter) VisitFirst(expr *expression.First) (interface{}, error) {
	return nil, fmt.Errorf("Expression not implemented")
}

func (this *JSConverter) VisitIn(expr *expression.In) (interface{}, error) {
	return nil, fmt.Errorf("Expression not implemented")
}

func (this *JSConverter) VisitWithin(expr *expression.Within) (interface{}, error) {
	return nil, fmt.Errorf("Expression not implemented")
}

// Comparison

func (this *JSConverter) VisitBetween(expr *expression.Between) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" > ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(" && ")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" < ")
	buf.WriteString(this.Visit(expr.Third()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitEq(expr *expression.Eq) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" == ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitLE(expr *expression.LE) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" <= ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitLike(expr *expression.Like) (interface{}, error) {
	return nil, fmt.Errorf("Expression not implemented")
}

func (this *JSConverter) VisitLT(expr *expression.LT) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" < ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitIsMissing(expr *expression.IsMissing) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("typeof(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(") == 'undefined')")
	return buf.String(), nil
}

func (this *JSConverter) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("typeof(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(") != 'undefined')")
	return buf.String(), nil
}

func (this *JSConverter) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" != null)")
	return buf.String(), nil
}

func (this *JSConverter) VisitIsNotValued(expr *expression.IsNotValued) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("!(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" != null)")
	return buf.String(), nil
}

func (this *JSConverter) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" == null)")
	return buf.String(), nil
}

func (this *JSConverter) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" != null)")
	return buf.String(), nil
}

// Concat
func (this *JSConverter) VisitConcat(expr *expression.Concat) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	for i, op := range expr.Operands() {
		if i > 0 {
			buf.WriteString(" + ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

// Constant
func (this *JSConverter) VisitConstant(expr *expression.Constant) (interface{}, error) {
	return json.Marshal(expr.Value())
}

// Identifier
func (this *JSConverter) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {

	var buf bytes.Buffer

	buf.WriteString("`")
	buf.WriteString(expr.Alias())
	buf.WriteString("`")
	return buf.String(), nil
}

// Construction

func (this *JSConverter) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("[")

	for i, op := range expr.Operands() {
		if i > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString("]")
	return buf.String(), nil
}

func (this *JSConverter) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("{")

	i := 0
	for k, v := range expr.Children() {
		if i > 0 {
			buf.WriteString(", ")
		}

		kb, _ := json.Marshal(k)
		buf.Write(kb)
		buf.WriteString(": ")
		buf.WriteString(this.Visit(v))
		i++
	}

	buf.WriteString("}")
	return buf.String(), nil
}

// Logic

func (this *JSConverter) VisitAnd(expr *expression.And) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	for i, op := range expr.Operands() {
		if i > 0 {
			buf.WriteString(" && ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitNot(expr *expression.Not) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(! ")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *JSConverter) VisitOr(expr *expression.Or) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	for i, op := range expr.Operands() {
		if i > 0 {
			buf.WriteString(" || ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

// Navigation

func (this *JSConverter) VisitElement(expr *expression.Element) (interface{}, error) {
	var buf bytes.Buffer
	//buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString("[")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString("]")
	//buf.WriteString(")")

	// if the stack is not empty, pop the function
	for this.stack.Size() != 0 {
		funcExpr := this.stack.Pop().(*funcExpr)
		if funcExpr.operands.Front() != nil {
			buf.WriteString(writeOperands(funcExpr.operands))
		} else {
			buf.WriteString(funcExpr.name)
		}
	}

	return buf.String(), nil
}

func (this *JSConverter) VisitField(expr *expression.Field) (interface{}, error) {
	var buf bytes.Buffer
	// parenthesis causing problems with certain expressions
	// lack of thereof could still present a problem with other
	// types of expressions. FIXME MAYBE
	//buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(".")
	buf.WriteString(this.Visit(expr.Second()))
	//buf.WriteString(")")
	// if the stack is not empty, pop the function
	for this.stack.Size() != 0 {
		funcExpr := this.stack.Pop().(*funcExpr)
		if funcExpr.operands.Front() != nil {
			buf.WriteString(writeOperands(funcExpr.operands))
		} else {
			buf.WriteString(funcExpr.name)
		}
	}

	return buf.String(), nil
}

func (this *JSConverter) VisitFieldName(expr *expression.FieldName) (interface{}, error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(expr.Alias())+2))
	buf.WriteString("`")
	buf.WriteString(expr.Alias())
	buf.WriteString("`")

	return buf.String(), nil
}

func (this *JSConverter) VisitSlice(expr *expression.Slice) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operands()[0]))
	buf.WriteString("[")
	buf.WriteString(this.Visit(expr.Operands()[1]))
	buf.WriteString(" : ")

	if len(expr.Operands()) > 2 {
		buf.WriteString(this.Visit(expr.Operands()[2]))
	}

	buf.WriteString("])")
	return buf.String(), nil
}

// Function
func (this *JSConverter) VisitFunction(expr expression.Function) (interface{}, error) {
	var buf bytes.Buffer

	functionExpr := &funcExpr{operands: list.New()}

	buf.WriteString("(")
	var nopush bool
	var nobracket bool
	var pushOperands bool

	switch strings.ToLower(expr.Name()) {
	case "lower":
		functionExpr.name = ".toLowerCase()"
		this.stack.Push(functionExpr)
	case "upper":
		functionExpr.name = ".toUpperCase()"
		this.stack.Push(functionExpr)
	case "length":
		functionExpr.name = ".length"
		this.stack.Push(functionExpr)
	case "str_to_millis":
		fallthrough
	case "millis":
		nopush = true
		buf.WriteString("Date.parse(")
	case "contains":
		functionExpr.name = ".indexOf("
		this.stack.Push(functionExpr)
		pushOperands = true
	case "substr":
		functionExpr.name = ".substring("
		this.stack.Push(functionExpr)
		pushOperands = true
	case "now_millis":
		buf.WriteString("Date.now().toString()")
	case "meta":
		buf.WriteString("meta")
		buf.WriteString(".")
		nobracket = true
	default:
		nopush = true
		buf.WriteString(expr.Name())
		buf.WriteString("(")
	}

	var firstOp expression.Expression

	for i, op := range expr.Operands() {
		if pushOperands == true {

			if i == 0 {
				firstOp = op
				continue
			} else {
				functionExpr.operands.PushBack(op)
			}

		} else {
			if i > 0 {
				buf.WriteString(", ")
			}

			if op == nil {
				buf.WriteString("*") // for count(*)
			} else {
				buf.WriteString(this.Visit(op))
			}
		}
	}

	if firstOp != nil {
		buf.WriteString(this.Visit(firstOp))
	}

	if nopush == true && nobracket == false {
		buf.WriteString("))")
	} else {
		buf.WriteString(")")
	}

	return buf.String(), nil
}

// Subqueries
func (this *JSConverter) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	return nil, fmt.Errorf("Subqueries cannot be index expressions")
}

// Named parameters
func (this *JSConverter) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return nil, fmt.Errorf("Parameters cannot be index expressions")
}

// Positional parameters
func (this *JSConverter) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return nil, fmt.Errorf("Parameters cannot be index expressions")
}

// Bindings

func (this *JSConverter) visitBindings(bindings expression.Bindings, w io.Writer, in, within string) {
	for i, b := range bindings {
		if i > 0 {
			io.WriteString(w, ", ")
		}

		io.WriteString(w, "`")
		io.WriteString(w, b.Variable())
		io.WriteString(w, "`")

		if b.Descend() {
			io.WriteString(w, within)
		} else {
			io.WriteString(w, in)
		}

		io.WriteString(w, this.Visit(b.Expression()))
	}
}
