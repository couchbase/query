//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Stringer struct {
	replacee string
	replacer string
	omit     bool
}

func NewStringer() *Stringer { return &Stringer{} }

/*
To replace the identifier in expression with the user-defined replacement:

	replacee: identifier to be replaced
	replacer: replacement for the identifier to be replaced
	omit: flag for the expression to skip the identifier replacement or not
	E.g.:
	replacee: `a`, replacer: `self`, omit: false:
	`a`.`b` -> `self`.`b`
	`a`.`a`.`b` -> `self`.`a`.`b`
	replacee: `a`, replacer: `self`, omit: true
	`a`.`a`.`b` -> `a`.`b`
*/
func (this *Stringer) SetReplace(replacee, replacer string, omit bool) {
	this.replacee = replacee
	this.replacer = replacer
	this.omit = omit
}

func (this *Stringer) Visit(expr Expression) string {
	s, err := expr.Accept(this)
	if err != nil {
		panic(fmt.Sprintf("Unexpected error in Stringer. expr: %v, error: %v", expr, err))
	}

	switch s := s.(type) {
	case []byte:
		return string(s)
	}

	return s.(string)
}

// Arithmetic

func (this *Stringer) VisitAdd(expr *Add) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	for i, op := range expr.operands {
		if i > 0 {
			buf.WriteString(" + ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitDiv(expr *Div) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" / ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitMod(expr *Mod) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" % ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitMult(expr *Mult) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	for i, op := range expr.operands {
		if i > 0 {
			buf.WriteString(" * ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitNeg(expr *Neg) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(-")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitSub(expr *Sub) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" - ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

// Case

func (this *Stringer) VisitSearchedCase(expr *SearchedCase) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("case")

	for _, when := range expr.whenTerms {
		buf.WriteString(" when ")
		buf.WriteString(this.Visit(when.When))
		buf.WriteString(" then ")
		buf.WriteString(this.Visit(when.Then))
	}

	if expr.elseTerm != nil {
		buf.WriteString(" else ")
		buf.WriteString(this.Visit(expr.elseTerm))
	}

	buf.WriteString(" end")
	return buf.String(), nil
}

func (this *Stringer) VisitSimpleCase(expr *SimpleCase) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("case ")
	buf.WriteString(this.Visit(expr.searchTerm))

	for _, when := range expr.whenTerms {
		buf.WriteString(" when ")
		buf.WriteString(this.Visit(when.When))
		buf.WriteString(" then ")
		buf.WriteString(this.Visit(when.Then))
	}

	if expr.elseTerm != nil {
		buf.WriteString(" else ")
		buf.WriteString(this.Visit(expr.elseTerm))
	}

	buf.WriteString(" end")
	return buf.String(), nil
}

// Collection

func (this *Stringer) VisitAny(expr *Any) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("any ")
	this.visitBindings(expr.bindings, &buf, " in ", " within ")
	buf.WriteString(" satisfies ")
	buf.WriteString(this.Visit(expr.satisfies))
	buf.WriteString(" end")
	return buf.String(), nil
}

func (this *Stringer) VisitEvery(expr *Every) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("every ")
	this.visitBindings(expr.bindings, &buf, " in ", " within ")
	buf.WriteString(" satisfies ")
	buf.WriteString(this.Visit(expr.satisfies))
	buf.WriteString(" end")
	return buf.String(), nil
}

func (this *Stringer) VisitAnyEvery(expr *AnyEvery) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("any and every ")
	this.visitBindings(expr.bindings, &buf, " in ", " within ")
	buf.WriteString(" satisfies ")
	buf.WriteString(this.Visit(expr.satisfies))
	buf.WriteString(" end")
	return buf.String(), nil
}

func (this *Stringer) VisitArray(expr *Array) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("array ")
	buf.WriteString(this.Visit(expr.valueMapping))
	buf.WriteString(" for ")
	this.visitBindings(expr.bindings, &buf, " in ", " within ")

	if expr.when != nil {
		buf.WriteString(" when ")
		buf.WriteString(this.Visit(expr.when))
	}

	buf.WriteString(" end")
	return buf.String(), nil
}

func (this *Stringer) VisitFirst(expr *First) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("first ")
	buf.WriteString(this.Visit(expr.valueMapping))
	buf.WriteString(" for ")
	this.visitBindings(expr.bindings, &buf, " in ", " within ")

	if expr.when != nil {
		buf.WriteString(" when ")
		buf.WriteString(this.Visit(expr.when))
	}

	buf.WriteString(" end")
	return buf.String(), nil
}

func (this *Stringer) VisitObject(expr *Object) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("object ")
	buf.WriteString(this.Visit(expr.nameMapping))
	buf.WriteString(" : ")
	buf.WriteString(this.Visit(expr.valueMapping))
	buf.WriteString(" for ")
	this.visitBindings(expr.bindings, &buf, " in ", " within ")

	if expr.when != nil {
		buf.WriteString(" when ")
		buf.WriteString(this.Visit(expr.when))
	}

	buf.WriteString(" end")
	return buf.String(), nil
}

func (this *Stringer) VisitExists(expr *Exists) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(exists ")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitIn(expr *In) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" in ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitWithin(expr *Within) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" within ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

// Comparison

func (this *Stringer) VisitBetween(expr *Between) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" between ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(" and ")
	buf.WriteString(this.Visit(expr.Third()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitEq(expr *Eq) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" = ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitLE(expr *LE) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" <= ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitLike(expr *Like) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" like ")
	buf.WriteString(this.Visit(expr.Second()))
	if !expr.IsDefaultEscape() {
		buf.WriteString(" escape ")
		buf.WriteString(this.Visit(expr.Escape()))
	}
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitLT(expr *LT) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString(" < ")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitIsMissing(expr *IsMissing) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" is missing)")
	return buf.String(), nil
}

func (this *Stringer) VisitIsNotMissing(expr *IsNotMissing) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" is not missing)")
	return buf.String(), nil
}

func (this *Stringer) VisitIsNotNull(expr *IsNotNull) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" is not null)")
	return buf.String(), nil
}

func (this *Stringer) VisitIsNotValued(expr *IsNotValued) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" is not valued)")
	return buf.String(), nil
}

func (this *Stringer) VisitIsNull(expr *IsNull) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" is null)")
	return buf.String(), nil
}

func (this *Stringer) VisitIsValued(expr *IsValued) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(" is valued)")
	return buf.String(), nil
}

// Concat
func (this *Stringer) VisitConcat(expr *Concat) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	for i, op := range expr.operands {
		if i > 0 {
			buf.WriteString(" || ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

// Constant
func (this *Stringer) VisitConstant(expr *Constant) (interface{}, error) {
	if expr.value.Type() == value.MISSING {
		return expr.value.String(), nil
	}

	b, _ := expr.value.MarshalJSON()
	return string(b), nil
}

// Identifier
func (this *Stringer) VisitIdentifier(expr *Identifier) (interface{}, error) {
	identifier := expr.identifier

	if this.replacee != "" && this.replacee == identifier {
		identifier = this.replacer
	}

	buf := bytes.NewBuffer(make([]byte, 0, len(expr.identifier)+3))
	buf.WriteString("`")
	buf.WriteString(identifier)
	buf.WriteString("`")

	if expr.CaseInsensitive() {
		buf.WriteString("i")
	}

	return buf.String(), nil
}

// Construction

func (this *Stringer) VisitArrayConstruct(expr *ArrayConstruct) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("[")

	for i, op := range expr.operands {
		if i > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString("]")
	return buf.String(), nil
}

func (this *Stringer) VisitObjectConstruct(expr *ObjectConstruct) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("{")

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
			buf.WriteString(", ")
		}

		// MB-21231 value.stringvalue.String() marshals strings already,
		// so string values have quotes prepepended.
		// We must avoid re-marshalling or we'll enter quoting hell.
		buf.WriteString(n)
		buf.WriteString(": ")
		v := expr.bindings[n]
		buf.WriteString(this.Visit(v))
		i++
	}

	buf.WriteString("}")
	return buf.String(), nil
}

// Logic

func (this *Stringer) VisitAnd(expr *And) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	for i, op := range expr.operands {
		if i > 0 {
			buf.WriteString(" and ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitNot(expr *Not) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(not ")
	buf.WriteString(this.Visit(expr.Operand()))
	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitOr(expr *Or) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	for i, op := range expr.operands {
		if i > 0 {
			buf.WriteString(" or ")
		}

		buf.WriteString(this.Visit(op))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

// Navigation

func (this *Stringer) VisitElement(expr *Element) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.First()))
	buf.WriteString("[")
	buf.WriteString(this.Visit(expr.Second()))
	buf.WriteString("])")
	return buf.String(), nil
}

func (this *Stringer) VisitField(expr *Field) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")

	if first, ok := expr.First().(*Identifier); !ok || this.replacee == "" ||
		first.identifier != this.replacee || !this.omit {
		buf.WriteString(this.Visit(expr.First()))
		buf.WriteString(".")
	}

	_, ok := expr.Second().(*FieldName)
	if !ok {
		buf.WriteString("[")
	}

	buf.WriteString(this.Visit(expr.Second()))

	if !ok {
		buf.WriteString("]")
		if expr.CaseInsensitive() {
			buf.WriteString("i")
		}
	}

	buf.WriteString(")")
	return buf.String(), nil
}

func (this *Stringer) VisitFieldName(expr *FieldName) (interface{}, error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(expr.name)+3))
	buf.WriteString("`")
	buf.WriteString(expr.name)
	buf.WriteString("`")

	if expr.CaseInsensitive() {
		buf.WriteString("i")
	}

	return buf.String(), nil
}

func (this *Stringer) VisitSlice(expr *Slice) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString("(")
	buf.WriteString(this.Visit(expr.Operands()[0]))
	buf.WriteString("[")
	if e := expr.Start(); e != nil {
		buf.WriteString(this.Visit(e))
	}
	buf.WriteString(" : ")

	if e := expr.End(); e != nil {
		buf.WriteString(this.Visit(e))
	}

	buf.WriteString("])")
	return buf.String(), nil
}

// Self
func (this *Stringer) VisitSelf(expr *Self) (interface{}, error) {
	return "self", nil
}

// Function
func (this *Stringer) VisitFunction(expr Function) (interface{}, error) {
	if expr.Aggregate() {
		return expr.String(), nil
	}
	if fk, ok := expr.(*FlattenKeys); ok {
		return this.visitFlattenKeys(fk)
	}

	var buf bytes.Buffer
	buf.WriteString(expr.Name())
	buf.WriteString("(")

	if expr.Distinct() {
		buf.WriteString("distinct ")
	}

	for i, op := range expr.Operands() {
		if i > 0 {
			buf.WriteString(", ")
		}

		if op == nil {
			buf.WriteString("*") // for count(*)
		} else {
			buf.WriteString(this.Visit(op))
		}
	}

	buf.WriteString(")")
	return buf.String(), nil
}

// Subquery
func (this *Stringer) VisitSubquery(expr Subquery) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString(expr.String())
	return buf.String(), nil
}

// NamedParameter
func (this *Stringer) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return "$" + expr.Name(), nil
}

// PositionalParameter
func (this *Stringer) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return "$" + strconv.Itoa(expr.Position()), nil
}

// Cover
func (this *Stringer) VisitCover(expr *Cover) (interface{}, error) {
	var buf bytes.Buffer
	if expr.FullCover() {
		buf.WriteString("cover (")
	} else if expr.IsIndexKey() {
		buf.WriteString("_index_key (")
	} else if expr.IsIndexCond() {
		buf.WriteString("_index_condition (")
	} else {
		return nil, fmt.Errorf("VisitCover: unexpected cover type")
	}
	buf.WriteString(expr.Text())
	buf.WriteString(")")
	return buf.String(), nil
}

// All
func (this *Stringer) VisitAll(expr *All) (interface{}, error) {
	var buf bytes.Buffer
	if expr.Distinct() {
		buf.WriteString("(distinct (")
	} else {
		buf.WriteString("(all (")
	}
	buf.WriteString(this.Visit(expr.Array()))
	buf.WriteString("))")
	return buf.String(), nil
}

// Bindings
func (this *Stringer) visitBindings(bindings Bindings, w io.Writer, in, within string) {
	for i, b := range bindings {
		if i > 0 {
			io.WriteString(w, ", ")
		}

		if b.nameVariable != "" {
			io.WriteString(w, "`")
			io.WriteString(w, b.nameVariable)
			io.WriteString(w, "` : ")
		}

		io.WriteString(w, "`")
		io.WriteString(w, b.variable)
		io.WriteString(w, "`")

		if b.descend {
			io.WriteString(w, within)
		} else {
			io.WriteString(w, in)
		}

		io.WriteString(w, this.Visit(b.expr))
	}
}

func (this *Stringer) visitFlattenKeys(fk *FlattenKeys) (interface{}, error) {
	var buf bytes.Buffer
	buf.WriteString(fk.Name())
	buf.WriteString("(")
	for i, op := range fk.Operands() {
		if i > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(this.Visit(op))
		buf.WriteString(fk.AttributeString(i))
	}

	buf.WriteString(")")
	return buf.String(), nil
}

type PathToString struct {
	MapperBase

	alias string
	path  string
}

func NewPathToString() *PathToString {
	rv := &PathToString{}
	stringer := NewStringer()

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
					if rv.path != "" {
						rv.path += "."
					}
					rv.path += "`" + sv + "`"
					if expr2.CaseInsensitive() {
						rv.path += "i"
					}
					return expr, nil
				}
			}

		case *Element:
			_, err := rv.Map(expr2.First())
			if err == nil {
				rv.path += "[" + stringer.Visit(expr2.Second()) + "]"
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

	return rv.alias, rv.path, err
}

const _NAME_CAP = 16

var _NAME_POOL = util.NewStringPool(256)
