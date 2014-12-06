//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/expression/parser"
	"github.com/couchbaselabs/query/value"
)

func (this *builder) VisitExecute(stmt *algebra.Execute) (interface{}, error) {

	// stmt contains a JSON representation of a plan.Prepared
	prepared_object := stmt.Prepared().Value()

	sig, ok := prepared_object.Field("signature")

	if !ok {
		return nil, errors.NewError(nil, "prepared is missing signature")
	}

	operator, ok := prepared_object.Field("operator")

	if !ok {
		return nil, errors.NewError(nil, "prepared is missing operator")
	}

	op_name, has_op_field := operator.Field("#operator")
	if !has_op_field {
		return nil, errors.NewError(nil, "Missing operator")
	}

	op, err := this.makeOperator(op_name, operator)

	return &Prepared{Operator: op, signature: sig}, err
}

func (this *builder) makeOperator(name value.Value, v value.Value) (Operator, error) {

	if name.Type() != value.STRING {
		return nil, errors.NewError(nil, "Missing operator name (#operator)")
	}
	switch name.Actual().(string) {
	case "Sequence":
		op_name, has_op_field := v.Field("~children")
		if !has_op_field {
			return nil, errors.NewError(nil, "Sequence operator is missing children")
		}
		return this.makeSequence(op_name)
	case "Parallel":
		op_name, has_op_field := v.Field("~child")
		if !has_op_field {
			return nil, errors.NewError(nil, "Parallel operator is missing child")
		}
		return this.makeParallel(op_name)
	case "KeyScan":
		op_name, has_op_field := v.Field("keys")
		if !has_op_field {
			return nil, errors.NewError(nil, "KeyScan operator is missing keys")
		}
		return this.makeKeyScan(op_name)
	case "PrimaryScan":
		index, _ := v.Field("index")
		keys, _ := v.Field("keyspace")
		names, _ := v.Field("namespace")
		return this.makePrimaryScan(index, keys, names)
	case "InitialProject":
		distinct, _ := v.Field("distinct")
		result_terms, _ := v.Field("result_terms")
		return this.makeInitialProject(distinct, result_terms)
	case "Fetch":
		keys, _ := v.Field("keyspace")
		names, _ := v.Field("namespace")
		as, _ := v.Field("as")
		return this.makeFetch(keys, names, as)
	case "Filter":
		condition, _ := v.Field("condition")
		return this.makeFilter(condition)
	case "FinalProject":
		return NewFinalProject(), nil
	case "DummyScan":
		return NewDummyScan(), nil
	case "Discard":
		return NewDiscard(), nil
	case "Distinct":
		return NewDistinct(), nil
	case "":
		return nil, errors.NewError(nil, "Missing operator")
	default:
		return nil, errors.NewError(nil, "Unrecognized operator")
	}
}

func (this *builder) makeInitialProject(distinct, result_terms value.Value) (Operator, error) {
	rts := []*algebra.ResultTerm{}
	i := 0
	for o, ok := result_terms.Index(i); ok; o, ok = result_terms.Index(i) {
		as_val, _ := o.Field("as")
		expr_val, _ := o.Field("expr")
		star_val, _ := o.Field("star")

		var expr expression.Expression
		var err error
		if expr_val.Type() == value.STRING {
			expr, err = parser.Parse(expr_val.Actual().(string))
			if err != nil {
				return nil, err
			}
		}
		as := ""
		if as_val.Type() == value.STRING {
			as = as_val.Actual().(string)
		}
		rt := algebra.NewResultTerm(expr, star_val.Truth(), as)
		rts = append(rts, rt)
		i = i + 1
	}
	projection := algebra.NewProjection(distinct.Truth(), rts)
	return NewInitialProject(projection), nil
}

func (this *builder) makeKeyScan(keys value.Value) (Operator, error) {
	if keys.Type() != value.STRING {
		return nil, errors.NewError(nil, "keys has incorrect type")
	}
	keys_expr, err := parser.Parse(keys.Actual().(string))
	if err != nil {
		return nil, err
	}
	return NewKeyScan(keys_expr), nil
}

func (this *builder) makeFetch(keyspace, namespace, as value.Value) (Operator, error) {
	ns_name := namespace.Actual().(string)
	ks_name := keyspace.Actual().(string)
	as_name := ""
	if as != nil && as.Type() == value.STRING {
		as_name = as.Actual().(string)
	}
	keyspaceTerm := algebra.NewKeyspaceTerm(ns_name, ks_name, nil, as_name, nil)

	n, err := this.datastore.NamespaceByName(ns_name)
	if err != nil {
		return nil, err
	}

	k, err := n.KeyspaceByName(ks_name)
	if err != nil {
		return nil, err
	}

	return NewFetch(k, keyspaceTerm), nil
}

func (this *builder) makePrimaryScan(index, keyspace, namespace value.Value) (Operator, error) {
	ns_name := namespace.Actual().(string)
	ks_name := keyspace.Actual().(string)
	keyspaceTerm := algebra.NewKeyspaceTerm(ns_name, ks_name, nil, "", nil)

	n, err := this.datastore.NamespaceByName(ns_name)
	if err != nil {
		return nil, err
	}

	k, err := n.KeyspaceByName(ks_name)
	if err != nil {
		return nil, err
	}

	idx, err := k.IndexByPrimary()
	if err != nil {
		return nil, err
	}

	return NewPrimaryScan(idx, keyspaceTerm), nil
}

func (this *builder) makeFilter(condition value.Value) (Operator, error) {
	if condition.Type() != value.STRING {
		return nil, errors.NewError(nil, "condition has incorrect type")
	}
	cond_expr, err := parser.Parse(condition.Actual().(string))
	if err != nil {
		return nil, err
	}
	return NewFilter(cond_expr), nil
}

func (this *builder) makeParallel(child value.Value) (Operator, error) {
	op_name, has_op_field := child.Field("#operator")

	if !has_op_field {
		return nil, errors.NewError(nil, "Parallel - Missing operator")
	}
	op, err := this.makeOperator(op_name, child)
	if err != nil {
		return nil, err
	}
	return NewParallel(op), nil
}

func (this *builder) makeSequence(children value.Value) (Operator, error) {
	ops := []Operator{}
	i := 0
	for o, ok := children.Index(i); ok; o, ok = children.Index(i) {
		op_name, has_op_field := o.Field("#operator")
		if !has_op_field {
			_, check_readonly := o.Field("readonly")
			if !check_readonly {
				return nil, errors.NewError(nil, "Sequence - Missing operator")
			} else {
				i = i + 1
				continue
			}
		}
		op, err := this.makeOperator(op_name, o)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
		i = i + 1
	}
	return NewSequence(ops...), nil
}
