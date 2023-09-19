//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package functionsBridge

// this package solely exists to avoid circular references between parse/n1ql, functions, expression, and functions/javascript

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
)

var NewFunctionName func(elem []string, namespace string, queryContext string) (functions.FunctionName, errors.Error) = func(elem []string, namespace string, queryContext string) (functions.FunctionName, errors.Error) {
	return functions.MockFunction(namespace, elem[len(elem)-1]), nil
}

var NewInlineBody func(expr expression.Expression, text string) (functions.FunctionBody, errors.Error) = func(expr expression.Expression, text string) (functions.FunctionBody, errors.Error) {
	return nil, nil
}

var NewGolangBody func(library, object string) (functions.FunctionBody, errors.Error) = func(library, object string) (functions.FunctionBody, errors.Error) {
	return nil, nil
}

var NewJavascriptBody func(library, object, text string) (functions.FunctionBody, errors.Error) = func(library, object, text string) (functions.FunctionBody, errors.Error) {
	return nil, nil
}

// Created to avoid circular references between functions and expression
type InlineUdfContext interface {
	GetAndSetInlineUdfExprs(udf string, expr expression.Expression, hasSubqueries, hasVariables bool,
		proc func(expression.Expression, bool) error) (expression.Expression, error)
}
