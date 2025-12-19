// Copyright 2025-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.
package sanitizer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/rewrite"
	"github.com/couchbase/query/semantics"
	"github.com/couchbase/query/value"
)

func SanitizeStatement(statement, namespace, queryContext string, txn, withParamMap bool, args ...logging.Log) (string, value.Value, error) {

	stmt, err := n1ql.ParseStatement2(statement, namespace, queryContext, args...)
	if err != nil {
		return "", nil, err
	}

	semChecker := semantics.GetSemChecker(stmt.Type(), txn)
	_, err = stmt.Accept(semChecker)
	if err != nil {
		return "", nil, err
	}

	rewriter := rewrite.NewRewrite(rewrite.REWRITE_PHASE1)
	_, err = stmt.Accept(rewriter)
	if err != nil {
		return "", nil, err
	}

	var paramMap map[string]value.Value
	switch stmt := stmt.(type) {
	case *algebra.CreateIndex, *algebra.CreateFunction:
		// create_index- where clause cannot be sanitized as namedparamters are not indexable
		// create_function- TODO: would require some thought to sanitize the body specifically for js udfs
		return stmt.String(), nil, nil
	default:
		stmt, paramMap, err = ReplaceConstantsWithNamedParams(stmt, true)
		if err != nil {
			return "", nil, err
		}
		if stringer, ok := stmt.(interface{ String() string }); ok {
			if paramMap == nil || len(paramMap) == 0 {
				return stringer.String(), nil, nil
			}
			return stringer.String(), value.NewValue(paramMap), nil
		} else {
			return "", nil, fmt.Errorf("Cannot sanitize statement of type: %s", stmt.Type())
		}
	}

}

func ReplaceConstantsWithNamedParams(stmt algebra.Statement, withParamMap bool) (algebra.Statement, map[string]value.Value, error) {
	mapper := NewConstantToNamedParam(withParamMap)
	err := stmt.MapExpressions(mapper)
	if err != nil {
		return nil, nil, err
	}

	return stmt, mapper.parametersMap, nil
}

type constantToNamedParam struct {
	expression.MapperBase
	constCounter  int
	parametersMap map[string]value.Value
}

func NewConstantToNamedParam(withParamMap bool) *constantToNamedParam {
	rv := &constantToNamedParam{}
	if withParamMap {
		rv.parametersMap = make(map[string]value.Value, 8)
	}
	rv.SetMapper(rv)
	return rv
}

func (this *constantToNamedParam) constructnamedparam() (string, string) {
	this.constCounter++
	var b strings.Builder

	b.WriteString("_n")
	b.WriteString(strconv.Itoa(this.constCounter))
	param := b.String()

	b.Reset()
	b.WriteByte('$')
	b.WriteString(param)
	key := b.String()
	return key, param
}

func (this *constantToNamedParam) VisitConstant(expr *expression.Constant) (interface{}, error) {
	key, param := this.constructnamedparam()
	if this.parametersMap != nil {
		this.parametersMap[key] = expr.Value()
	}
	return algebra.NewNamedParameter(param), nil
}

func (this *constantToNamedParam) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	if expr.Value() != nil {
		key, param := this.constructnamedparam()
		if this.parametersMap != nil {
			this.parametersMap[key] = expr.Value()
		}
		return algebra.NewNamedParameter(param), nil
	}

	err := expr.MapValues(this)
	return expr, err
}

func (this *constantToNamedParam) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	if expr.Value() != nil {
		key, param := this.constructnamedparam()
		if this.parametersMap != nil {
			this.parametersMap[key] = expr.Value()
		}
		return algebra.NewNamedParameter(param), nil
	}

	err := expr.MapChildren(this)
	return expr, err
}
