//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const _INIT_BUF = 128

type Stringer struct {
	buf strings.Builder
}

func (this *Stringer) WriteString(s string) {
	this.buf.WriteString(s)
}

func (this *Stringer) String() string {
	s := this.buf.String()
	this.buf.Reset()
	return s
}

func NewStringer() *Stringer {
	s := &Stringer{}
	s.buf.Grow(_INIT_BUF)
	return s
}

func (this *Stringer) Visit(expr Expression) string {
	_, err := expr.Accept(this)
	if err != nil {
		panic(fmt.Sprintf("Unexpected error in Stringer. expr: %v, error: %v", expr, err))
	}
	return this.String()
}

// this supresses returning the buffer so more can be written to it afterwards
func (this *Stringer) VisitShared(expr Expression) {
	_, err := expr.Accept(this)
	if err != nil {
		panic(fmt.Sprintf("Unexpected error in Stringer. expr: %v, error: %v", expr, err))
	}
}

// Arithmetic

func (this *Stringer) VisitAdd(expr *Add) (interface{}, error) {
	this.WriteString("(")
	for i, op := range expr.operands {
		if i > 0 {
			this.WriteString(" + ")
		}
		this.VisitShared(op)
	}
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitDiv(expr *Div) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" / ")
	this.VisitShared(expr.Second())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitMod(expr *Mod) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" % ")
	this.VisitShared(expr.Second())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitMult(expr *Mult) (interface{}, error) {
	this.WriteString("(")
	for i, op := range expr.operands {
		if i > 0 {
			this.WriteString(" * ")
		}
		this.VisitShared(op)
	}
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitNeg(expr *Neg) (interface{}, error) {
	this.WriteString("(-")
	this.VisitShared(expr.Operand())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitSub(expr *Sub) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" - ")
	this.VisitShared(expr.Second())
	this.WriteString(")")
	return nil, nil
}

// Case

func (this *Stringer) VisitSearchedCase(expr *SearchedCase) (interface{}, error) {
	this.WriteString("case")
	for _, when := range expr.whenTerms {
		this.WriteString(" when ")
		this.VisitShared(when.When)
		this.WriteString(" then ")
		this.VisitShared(when.Then)
	}
	if expr.elseTerm != nil {
		this.WriteString(" else ")
		this.VisitShared(expr.elseTerm)
	}
	this.WriteString(" end")
	return nil, nil
}

func (this *Stringer) VisitSimpleCase(expr *SimpleCase) (interface{}, error) {
	this.WriteString("case ")
	this.VisitShared(expr.searchTerm)
	for _, when := range expr.whenTerms {
		this.WriteString(" when ")
		this.VisitShared(when.When)
		this.WriteString(" then ")
		this.VisitShared(when.Then)
	}
	if expr.elseTerm != nil {
		this.WriteString(" else ")
		this.VisitShared(expr.elseTerm)
	}
	this.WriteString(" end")
	return nil, nil
}

// Collection

func (this *Stringer) VisitAny(expr *Any) (interface{}, error) {
	this.WriteString("any ")
	this.visitBindings(expr.bindings, " in ", " within ")
	this.WriteString(" satisfies ")
	this.VisitShared(expr.satisfies)
	this.WriteString(" end")
	return nil, nil
}

func (this *Stringer) VisitEvery(expr *Every) (interface{}, error) {
	this.WriteString("every ")
	this.visitBindings(expr.bindings, " in ", " within ")
	this.WriteString(" satisfies ")
	this.VisitShared(expr.satisfies)
	this.WriteString(" end")
	return nil, nil
}

func (this *Stringer) VisitAnyEvery(expr *AnyEvery) (interface{}, error) {
	this.WriteString("any and every ")
	this.visitBindings(expr.bindings, " in ", " within ")
	this.WriteString(" satisfies ")
	this.VisitShared(expr.satisfies)
	this.WriteString(" end")
	return nil, nil
}

func (this *Stringer) VisitArray(expr *Array) (interface{}, error) {
	this.WriteString("array ")
	this.VisitShared(expr.valueMapping)
	this.WriteString(" for ")
	this.visitBindings(expr.bindings, " in ", " within ")
	if expr.when != nil {
		this.WriteString(" when ")
		this.VisitShared(expr.when)
	}
	this.WriteString(" end")
	return nil, nil
}

func (this *Stringer) VisitFirst(expr *First) (interface{}, error) {
	this.WriteString("first ")
	this.VisitShared(expr.valueMapping)
	this.WriteString(" for ")
	this.visitBindings(expr.bindings, " in ", " within ")
	if expr.when != nil {
		this.WriteString(" when ")
		this.VisitShared(expr.when)
	}
	this.WriteString(" end")
	return nil, nil
}

func (this *Stringer) VisitObject(expr *Object) (interface{}, error) {
	this.WriteString("object ")
	this.VisitShared(expr.nameMapping)
	this.WriteString(" : ")
	this.VisitShared(expr.valueMapping)
	this.WriteString(" for ")
	this.visitBindings(expr.bindings, " in ", " within ")
	if expr.when != nil {
		this.WriteString(" when ")
		this.VisitShared(expr.when)
	}
	this.WriteString(" end")
	return nil, nil
}

func (this *Stringer) VisitExists(expr *Exists) (interface{}, error) {
	this.WriteString("(exists ")
	this.VisitShared(expr.Operand())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitIn(expr *In) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" in ")
	this.VisitShared(expr.Second())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitWithin(expr *Within) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" within ")
	this.VisitShared(expr.Second())
	this.WriteString(")")
	return nil, nil
}

// Comparison

func (this *Stringer) VisitBetween(expr *Between) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" between ")
	this.VisitShared(expr.Second())
	this.WriteString(" and ")
	this.VisitShared(expr.Third())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitEq(expr *Eq) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" = ")
	this.VisitShared(expr.Second())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitLE(expr *LE) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" <= ")
	this.VisitShared(expr.Second())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitLike(expr *Like) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" like ")
	this.VisitShared(expr.Second())
	if !expr.IsDefaultEscape() {
		this.WriteString(" escape ")
		this.VisitShared(expr.Escape())
	}
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitLT(expr *LT) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(" < ")
	this.VisitShared(expr.Second())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitIsMissing(expr *IsMissing) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.Operand())
	this.WriteString(" is missing)")
	return nil, nil
}

func (this *Stringer) VisitIsNotMissing(expr *IsNotMissing) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.Operand())
	this.WriteString(" is not missing)")
	return nil, nil
}

func (this *Stringer) VisitIsNotNull(expr *IsNotNull) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.Operand())
	this.WriteString(" is not null)")
	return nil, nil
}

func (this *Stringer) VisitIsNotValued(expr *IsNotValued) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.Operand())
	this.WriteString(" is not valued)")
	return nil, nil
}

func (this *Stringer) VisitIsNull(expr *IsNull) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.Operand())
	this.WriteString(" is null)")
	return nil, nil
}

func (this *Stringer) VisitIsValued(expr *IsValued) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.Operand())
	this.WriteString(" is valued)")
	return nil, nil
}

// Concat
func (this *Stringer) VisitConcat(expr *Concat) (interface{}, error) {
	this.WriteString("(")
	for i, op := range expr.operands {
		if i > 0 {
			this.WriteString(" || ")
		}
		this.VisitShared(op)
	}
	this.WriteString(")")
	return nil, nil
}

// Constant
func (this *Stringer) VisitConstant(expr *Constant) (interface{}, error) {
	if expr.value.Type() == value.MISSING {
		this.WriteString(expr.value.String())
	} else {
		b, _ := expr.value.MarshalJSON()
		this.WriteString(string(b))
	}
	return nil, nil
}

// Identifier
func (this *Stringer) VisitIdentifier(expr *Identifier) (interface{}, error) {
	identifier := expr.identifier
	this.WriteString("`")
	this.WriteString(identifier)
	this.WriteString("`")
	if expr.CaseInsensitive() {
		this.WriteString("i")
	}
	return nil, nil
}

// Construction

func (this *Stringer) VisitArrayConstruct(expr *ArrayConstruct) (interface{}, error) {
	this.WriteString("[")
	for i, op := range expr.operands {
		if i > 0 {
			this.WriteString(", ")
		}
		this.VisitShared(op)
	}
	this.WriteString("]")
	return nil, nil
}

func (this *Stringer) VisitObjectConstruct(expr *ObjectConstruct) (interface{}, error) {
	this.WriteString("{")
	// Sort names
	var nameBuf [_NAME_CAP]string
	var names []string
	if len(expr.bindings) <= len(nameBuf) {
		names = nameBuf[0:0]
	} else {
		names = _NAME_POOL.GetCapped(len(expr.bindings))
		defer _NAME_POOL.Put(names)
	}
	for name, _ := range expr.bindings {
		names = append(names, name)
	}
	sort.Strings(names)

	i := 0
	for _, n := range names {
		if i > 0 {
			this.WriteString(", ")
		}
		// MB-21231 value.stringvalue.String() marshals strings already,
		// so string values have quotes prepepended.
		// We must avoid re-marshalling or we'll enter quoting hell.
		this.WriteString(n)
		this.WriteString(": ")
		v := expr.bindings[n]
		this.VisitShared(v)
		i++
	}
	this.WriteString("}")
	return nil, nil
}

// Logic

func (this *Stringer) VisitAnd(expr *And) (interface{}, error) {
	this.WriteString("(")
	for i, op := range expr.operands {
		if i > 0 {
			this.WriteString(" and ")
		}
		this.VisitShared(op)
	}
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitNot(expr *Not) (interface{}, error) {
	this.WriteString("(not ")
	this.VisitShared(expr.Operand())
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitOr(expr *Or) (interface{}, error) {
	this.WriteString("(")
	for i, op := range expr.operands {
		if i > 0 {
			this.WriteString(" or ")
		}
		this.VisitShared(op)
	}
	this.WriteString(")")
	return nil, nil
}

// Navigation

func (this *Stringer) VisitElement(expr *Element) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString("[")
	this.VisitShared(expr.Second())
	this.WriteString("])")
	return nil, nil
}

func (this *Stringer) VisitField(expr *Field) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.First())
	this.WriteString(".")
	_, ok := expr.Second().(*FieldName)
	if !ok {
		this.WriteString("[")
	}
	this.VisitShared(expr.Second())
	if !ok {
		this.WriteString("]")
		if expr.CaseInsensitive() {
			this.WriteString("i")
		}
	}
	this.WriteString(")")
	return nil, nil
}

func (this *Stringer) VisitFieldName(expr *FieldName) (interface{}, error) {
	this.WriteString("`")
	this.WriteString(expr.name)
	this.WriteString("`")
	if expr.CaseInsensitive() {
		this.WriteString("i")
	}
	return nil, nil
}

func (this *Stringer) VisitSlice(expr *Slice) (interface{}, error) {
	this.WriteString("(")
	this.VisitShared(expr.Operands()[0])
	this.WriteString("[")
	if e := expr.Start(); e != nil {
		this.VisitShared(e)
	}
	this.WriteString(" : ")
	if e := expr.End(); e != nil {
		this.VisitShared(e)
	}
	this.WriteString("])")
	return nil, nil
}

// Self
func (this *Stringer) VisitSelf(expr *Self) (interface{}, error) {
	this.WriteString("self")
	return nil, nil
}

// Function
func (this *Stringer) VisitFunction(expr Function) (interface{}, error) {
	if expr.Aggregate() {
		if ab, ok := expr.(interface{ WriteToStringer(*Stringer) }); ok {
			ab.WriteToStringer(this)
		} else {
			this.WriteString(expr.String())
		}
		return nil, nil
	}
	switch t := expr.(type) {
	case *FlattenKeys:
		return this.visitFlattenKeys(t)
	case *SequenceOperation:
		this.WriteString("(")
		this.WriteString(t.Operator())
		this.WriteString(")")
		return nil, nil
	case *CurrentUser:
		op := t.Operator()
		if op != "" {
			this.WriteString("(")
			this.WriteString(op)
			this.WriteString(")")
			return nil, nil
		}
	case UnaryFunction:
		op := t.Operator()
		if op != "" {
			this.WriteString("(")
			this.VisitShared(t.Operand())
			this.WriteString(op)
			this.WriteString(")")
			return nil, nil
		}
	case BinaryFunction:
		op := t.Operator()
		if op != "" {
			this.WriteString("(")
			this.VisitShared(t.First())
			this.WriteString(op)
			this.VisitShared(t.Second())
			this.WriteString(")")
			return nil, nil
		}
	}

	if udf, ok := expr.(*UserDefinedFunction); ok {
		this.WriteString(udf.ProtectedName())
	} else {
		this.WriteString(expr.Name())
	}
	this.WriteString("(")

	if expr.Distinct() {
		this.WriteString("distinct ")
	}
	for i, op := range expr.Operands() {
		if i > 0 {
			this.WriteString(", ")
		}
		if op == nil {
			this.WriteString("*") // for count(*)
		} else {
			this.VisitShared(op)
		}
	}
	this.WriteString(")")
	return nil, nil
}

// Subquery
func (this *Stringer) VisitSubquery(expr Subquery) (interface{}, error) {
	this.WriteString(expr.String())
	return nil, nil
}

// InferUnderParenthesis
func (this *Stringer) VisitParenInfer(expr ParenInfer) (interface{}, error) {
	this.WriteString(expr.String())
	return nil, nil
}

// NamedParameter
func (this *Stringer) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	this.WriteString("$" + expr.Name())
	return nil, nil
}

// PositionalParameter
func (this *Stringer) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	this.WriteString("$" + strconv.Itoa(expr.Position()))
	return nil, nil
}

// Cover
func (this *Stringer) VisitCover(expr *Cover) (interface{}, error) {
	if expr.FullCover() {
		this.WriteString("cover (")
	} else if expr.IsIndexKey() {
		this.WriteString("_index_key (")
	} else if expr.IsIndexCond() {
		this.WriteString("_index_condition (")
	} else {
		return nil, fmt.Errorf("VisitCover: unexpected cover type")
	}
	this.WriteString(expr.Text())
	this.WriteString(")")
	return nil, nil
}

// All
func (this *Stringer) VisitAll(expr *All) (interface{}, error) {
	if expr.Distinct() {
		this.WriteString("(distinct (")
	} else {
		this.WriteString("(all (")
	}
	this.VisitShared(expr.Array())
	this.WriteString("))")
	return nil, nil
}

// Bindings
func (this *Stringer) visitBindings(bindings Bindings, in string, within string) {
	for i, b := range bindings {
		if i > 0 {
			this.WriteString(", ")
		}
		if b.nameVariable != "" {
			this.WriteString("`")
			this.WriteString(b.nameVariable)
			this.WriteString("` : ")
		}
		this.WriteString("`")
		this.WriteString(b.variable)
		this.WriteString("`")
		if b.descend {
			this.WriteString(within)
		} else {
			this.WriteString(in)
		}
		this.VisitShared(b.expr)
	}
}

func (this *Stringer) visitFlattenKeys(fk *FlattenKeys) (interface{}, error) {
	this.WriteString(fk.Name())
	this.WriteString("(")
	for i, op := range fk.Operands() {
		if i > 0 {
			this.WriteString(", ")
		}
		this.VisitShared(op)
		this.WriteString(fk.AttributeString(i))
	}
	this.WriteString(")")
	return nil, nil
}

type PathToString struct {
	MapperBase

	alias string
	path  strings.Builder
}

func NewPathToString() *PathToString {
	stringer := NewStringer()
	rv := &PathToString{}
	rv.path.Grow(128) // Pre-allocate buffer for path
	rv.SetMapper(rv)
	rv.SetMapFunc(func(expr Expression) (Expression, error) {
		switch expr2 := expr.(type) {
		case *Identifier:
			rv.alias = expr2.Alias()
			return expr, nil
		case *Field:
			var sv string
			second := expr2.Second().Value()
			if second != nil {
				sv = second.ToString()
			}
			if sv != "" {
				_, err := rv.Map(expr2.First())
				if err == nil {
					if rv.path.Len() > 0 {
						rv.path.WriteString(".")
					}
					rv.path.WriteString("`")
					rv.path.WriteString(sv)
					rv.path.WriteString("`")
					if expr2.CaseInsensitive() {
						rv.path.WriteString("i")
					}
					return expr, nil
				}
			}
		case *Element:
			_, err := rv.Map(expr2.First())
			if err == nil {
				rv.path.WriteString("[")
				rv.path.WriteString(stringer.Visit(expr2.Second()))
				rv.path.WriteString("]")
				return expr, nil
			}
			return expr, nil
		}
		return expr, fmt.Errorf("not field name")
	})

	return rv
}

func PathString(expr Expression) (alias, path string, err error) {
	rv := NewPathToString()
	_, err = rv.Map(expr)
	if err != nil {
		return "", "", err
	}
	return rv.alias, rv.path.String(), err
}

const _NAME_CAP = 16

var _NAME_POOL = util.NewStringPool(256)
