//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package search

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
//        FTS SEARCH function
//
///////////////////////////////////////////////////

var ParseXattrs func(expression.Expression) ([]string, error)

func RegisterParseXattrs(f func(expression.Expression) ([]string, error)) {
	ParseXattrs = f
}

type SearchVerify interface {
	Evaluate(item value.Value) (bool, errors.Error)
}

type Search struct {
	expression.FunctionBase
	keyspacePath string
	verify       SearchVerify
	err          error
}

func NewSearch(operands ...expression.Expression) expression.Function {
	rv := &Search{}
	rv.Init("search", operands...)
	rv.SetExpr(rv)
	return rv
}

func (this *Search) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Search) Type() value.Type                           { return value.BOOLEAN }
func (this *Search) MinArgs() int                               { return 2 }
func (this *Search) MaxArgs() int                               { return 3 }
func (this *Search) Indexable() bool                            { return false }
func (this *Search) DependsOn(other expression.Expression) bool { return false }

func (this *Search) CoveredBy(keyspace string, exprs expression.Expressions,
	options expression.CoveredOptions) expression.Covered {

	if this.KeyspaceAlias() != keyspace {
		return expression.CoveredSkip
	}

	for _, expr := range exprs {
		if this.EquivalentTo(expr) {
			return expression.CoveredEquiv
		}
	}

	return expression.CoveredFalse
}

func (this *Search) FieldNames(base expression.Expression, names map[string]bool) (present bool) {

	xattrs := false

	if expr, ok := base.(*expression.Field); ok {
		if _, ok := expr.First().(*expression.Meta); ok {
			if base.Alias() == "xattrs" {
				this.addXattrsFields(names)
				xattrs = true
			}
		}
	}
	if xattrs {
		return xattrs
	} else {
		return this.ExpressionBase.FieldNames(base, names)
	}
}

func (this *Search) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	if this.verify == nil {
		return value.FALSE_VALUE, this.err
	}

	// Evaluate document for keyspace. If MISSING or NULL return (For OUTER Join)
	val, err := this.Keyspace().Evaluate(item, context)
	if err != nil || val.Type() <= value.NULL {
		return val, err
	}

	cond, err := this.verify.Evaluate(val)
	if err != nil || !cond {
		return value.FALSE_VALUE, err
	}

	return value.TRUE_VALUE, nil

}

/*
Factory method pattern.
*/
func (this *Search) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewSearch(operands...)
	}
}

func (this *Search) SetVerify(v SearchVerify, err error) {
	this.verify = v
	this.err = err
}

func (this *Search) Keyspace() *expression.Identifier {
	ident := expression.NewIdentifier(this.KeyspaceAlias())
	ident.SetKeyspaceAlias(true)
	return ident
}

func (this *Search) KeyspaceAlias() string {
	s, _, _ := expression.PathString(this.Operands()[0])
	return s
}

func (this *Search) KeyspacePath() string {
	return this.keyspacePath
}

func (this *Search) SetKeyspacePath(path string) {
	this.keyspacePath = path
}

func (this *Search) FieldName() string {
	_, s, _ := expression.PathString(this.Operands()[0])
	return s
}

func (this *Search) Query() expression.Expression {
	return this.Operands()[1]
}

func (this *Search) Options() expression.Expression {
	if len(this.Operands()) > 2 {
		return this.Operands()[2]
	}

	return nil
}

func (this *Search) IndexName() (name string) {
	name, _, _ = this.getIndexNameAndOutName(this.Options())
	return
}

func (this *Search) OutName() string {
	_, outName, _ := this.getIndexNameAndOutName(this.Options())
	if outName == "" {
		outName = expression.DEF_OUTNAME
	}

	return outName
}

func (this *Search) IndexMetaField() expression.Expression {
	return expression.NewField(this.Keyspace(), expression.NewFieldName(this.OutName(), false))

}

func (this *Search) getIndexNameAndOutName(arg expression.Expression) (index, outName string, err error) {
	if arg == nil {
		return
	}
	options := arg.Value()
	if options == nil {
		if oc, ok := arg.(*expression.ObjectConstruct); ok {
			for name, val := range oc.Mapping() {
				n := name.Value()
				if n == nil || n.Type() != value.STRING {
					continue
				}

				if n.ToString() == "index" {
					v := val.Value()
					if v == nil || (v.Type() != value.STRING && v.Type() != value.OBJECT) {
						err = fmt.Errorf("%s() not valid third argument: %v", this.Name(),
							arg.String())
						return

					}
					index = v.ToString()
				}

				if n.ToString() == "out" {
					v := val.Value()
					if v == nil || v.Type() != value.STRING {
						err = fmt.Errorf("%s() not valid third argument: %v", this.Name(),
							arg.String())
						return
					}
					outName = v.ToString()
				}
			}
		}
	} else if options.Type() == value.OBJECT {
		if val, ok := options.Field("index"); ok {
			if val == nil || (val.Type() != value.STRING && val.Type() != value.OBJECT) {
				err = fmt.Errorf("%s() not valid third argument: %v", this.Name(), arg.String())
				return
			}
			index = val.ToString()
		}

		if val, ok := options.Field("out"); ok {
			if val == nil || val.Type() != value.STRING {
				err = fmt.Errorf("%s() not valid third argument: %v", this.Name(), arg.String())
				return
			}
			outName = val.ToString()
		}
	}

	return

}

func (this *Search) ValidOperands() error {
	op := this.Operands()[0]
	a, _, e := expression.PathString(op)
	if a == "" || e != nil {
		return fmt.Errorf("%s() not valid first argument: %s", this.Name(), op.String())
	}

	op = this.Query()
	val := op.Value()
	if (val != nil && val.Type() != value.STRING && val.Type() != value.OBJECT) || op.StaticNoVariable() == nil {
		return fmt.Errorf("%s() not valid second argument: %s", this.Name(), op.String())
	}

	_, _, err := this.getIndexNameAndOutName(this.Options())
	return err
}

func (this *Search) addXattrsFields(names map[string]bool) {

	if ParseXattrs == nil {
		return
	}

	fields, err := ParseXattrs(this.Query())
	if err != nil || fields == nil {
		return
	}

	for _, fieldName := range fields {
		names[fieldName] = true
	}
}

type SearchMeta struct {
	expression.FunctionBase
	keyspace *expression.Identifier
	field    *expression.Field
	second   value.Value
}

func NewSearchMeta(operands ...expression.Expression) expression.Function {
	rv := &SearchMeta{}
	rv.Init("search_meta", operands...)
	rv.SetExpr(rv)
	return rv
}

func (this *SearchMeta) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *SearchMeta) Type() value.Type                           { return value.OBJECT }
func (this *SearchMeta) MinArgs() int                               { return 0 }
func (this *SearchMeta) MaxArgs() int                               { return 1 }
func (this *SearchMeta) Indexable() bool                            { return false }
func (this *SearchMeta) DependsOn(other expression.Expression) bool { return false }

func (this *SearchMeta) CoveredBy(keyspace string, exprs expression.Expressions,
	options expression.CoveredOptions) expression.Covered {

	if this.KeyspaceAlias() != keyspace {
		return expression.CoveredSkip
	}

	for _, expr := range exprs {
		if this.EquivalentTo(expr) {
			return expression.CoveredEquiv
		}
	}

	return expression.CoveredFalse
}

func (this *SearchMeta) Keyspace() *expression.Identifier {
	op := this.Operands()[0]
	switch op := op.(type) {
	case *expression.Identifier:
		return op
	case *expression.Field:
		keyspace, _ := op.First().(*expression.Identifier)
		return keyspace
	default:
		return nil
	}
}

func (this *SearchMeta) KeyspaceAlias() string {
	keyspace := this.Keyspace()
	if keyspace != nil {
		return keyspace.Alias()
	}
	return ""
}

func (this *SearchMeta) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	if this.keyspace == nil {

		// Transform argument FROM ks.idxname TO META(ks).idxname
		this.keyspace = this.Keyspace()
		if this.keyspace == nil {
			return value.NULL_VALUE, nil
		}

		op := this.Operands()[0]

		if field, ok := op.(*expression.Field); ok {
			if _, ok = field.First().(*expression.Identifier); !ok {
				return value.NULL_VALUE, nil
			}
			this.second = field.Second().Value()
			this.field = expression.NewField(nil, field.Second())
		}
	}

	val, err := this.getSmeta(this.keyspace, item, context)
	if err != nil {
		return value.NULL_VALUE, err
	}

	if this.field != nil {
		return this.field.DoEvaluate(context, val, this.second)
	} else {
		return val, err
	}
}

func (this *SearchMeta) getSmeta(keyspace *expression.Identifier, item value.Value,
	context expression.Context) (value.Value, error) {

	if keyspace == nil {
		return value.NULL_VALUE, nil
	}

	val, err := keyspace.Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if val.Type() == value.MISSING {
		return val, nil
	}

	switch val := val.(type) {
	case value.AnnotatedValue:
		return value.NewValue(val.GetAttachment(value.ATT_SMETA)), nil
	default:
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *SearchMeta) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewSearchMeta(operands...)
	}
}

type SearchScore struct {
	expression.FunctionBase
	score expression.Expression
}

func NewSearchScore(operands ...expression.Expression) expression.Function {
	rv := &SearchScore{}
	rv.Init("search_score", operands...)

	rv.SetExpr(rv)
	return rv
}

func (this *SearchScore) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *SearchScore) Type() value.Type                           { return value.NUMBER }
func (this *SearchScore) MinArgs() int                               { return 0 }
func (this *SearchScore) MaxArgs() int                               { return 1 }
func (this *SearchScore) Indexable() bool                            { return false }
func (this *SearchScore) DependsOn(other expression.Expression) bool { return false }
func (this *SearchScore) IndexMetaField() expression.Expression      { return this.Operands()[0] }

func (this *SearchScore) CoveredBy(keyspace string, exprs expression.Expressions,
	options expression.CoveredOptions) expression.Covered {

	if this.KeyspaceAlias() != keyspace {
		return expression.CoveredSkip
	}

	for _, expr := range exprs {
		if this.EquivalentTo(expr) {
			return expression.CoveredEquiv
		}
	}

	return expression.CoveredFalse
}

func (this *SearchScore) Keyspace() *expression.Identifier {
	op := this.Operands()[0]
	switch op := op.(type) {
	case *expression.Identifier:
		return op
	case *expression.Field:
		keyspace, _ := op.First().(*expression.Identifier)
		return keyspace
	default:
		return nil
	}
}

func (this *SearchScore) KeyspaceAlias() string {
	keyspace := this.Keyspace()
	if keyspace != nil {
		return keyspace.Alias()
	}
	return ""
}

func (this *SearchScore) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	if this.score == nil {
		// Transform argument FROM ks.idxname TO META(ks).idxname.score
		this.score = expression.NewField(NewSearchMeta(this.Operands()...),
			expression.NewFieldName("score", false))
	}
	return this.score.Evaluate(item, context)
}

/*
Factory method pattern.
*/
func (this *SearchScore) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewSearchScore(operands...)
	}
}
