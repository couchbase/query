//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"fmt"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
)

func (this *builder) PatternFor(baseKeyspace *base.BaseKeyspace, indexes []datastore.Index,
	formalizer *expression.Formalizer) error {

	pred := baseKeyspace.OrigPred()

	suffixes := _PATTERN_INDEX_POOL.Get()
	defer _PATTERN_INDEX_POOL.Put(suffixes)
	tokens := _PATTERN_INDEX_POOL.Get()
	defer _PATTERN_INDEX_POOL.Put(tokens)

	collectPatternIndexes(pred, indexes, formalizer, suffixes, tokens)
	if len(suffixes) == 0 && len(tokens) == 0 {
		return nil
	}

	var err error
	pred = pred.Copy()
	dnf := base.NewDNF(pred, false, true)
	pred, err = dnf.Map(pred)
	if err != nil {
		return err
	}

	pat := newPattern(suffixes, tokens)
	rv, err := pred.Accept(pat)
	if err != nil {
		return err
	}

	// update filters list in baseKeyspace since new filters are generated above
	baseKeyspaces := base.CopyBaseKeyspaces(this.baseKeyspaces)
	_, err = ClassifyExpr(rv.(expression.Expression), baseKeyspaces, this.keyspaceNames, false,
		this.useCBO, this.advisorValidate(), this.context)
	if err != nil {
		return err
	}

	newKeyspace, ok := baseKeyspaces[baseKeyspace.Name()]
	if !ok {
		return errors.NewPlanInternalError(fmt.Sprintf("PatternFor: missing baseKeyspace %s", baseKeyspace.Name()))
	}

	addUnnestPreds(baseKeyspaces, newKeyspace)
	baseKeyspace.SetFilters(newKeyspace.Filters(), newKeyspace.JoinFilters())
	err = CombineFilters(baseKeyspace, true, false)
	if err != nil {
		return err
	}

	return nil
}

type pattern struct {
	expression.MapperBase

	suffixes map[string]string
	tokens   map[string]string
}

func newPattern(suffixes, tokens map[string]string) *pattern {
	rv := &pattern{
		suffixes: suffixes,
		tokens:   tokens,
	}

	rv.SetMapper(rv)
	return rv
}

func (this *pattern) VisitLike(expr *expression.Like) (interface{}, error) {
	source := expr.First()
	variable, ok := this.suffixes[source.String()]
	if !ok {
		return expr, nil
	}

	suffixes := expression.NewSuffixes(source)
	binding := expression.NewSimpleBinding(variable, suffixes)
	suffix := expression.NewLikeSuffix(expr.Second())
	sat := expression.NewLike(expression.NewIdentifier(variable), suffix)
	any := expression.NewAny(expression.Bindings{binding}, sat)
	return expression.NewAnd(expr, any), nil
}

func (this *pattern) VisitFunction(expr expression.Function) (interface{}, error) {
	switch expr := expr.(type) {
	case *expression.Contains:
		return this.visitContains(expr)
	case *expression.ContainsToken:
		return this.visitContainsToken(expr)
	case *expression.ContainsTokenLike:
		return this.visitContainsTokenLike(expr)
	case *expression.ContainsTokenRegexp:
		return this.visitContainsTokenRegexp(expr)
	case *expression.RegexpContains:
		return this.visitRegexpContains(expr)
	case *expression.RegexpLike:
		return this.visitRegexpLike(expr)
	default:
		return expr, nil
	}
}

func (this *pattern) visitContains(expr *expression.Contains) (interface{}, error) {
	source := expr.First()
	variable, ok := this.suffixes[source.String()]
	if !ok {
		return expr, nil
	}

	suffixes := expression.NewSuffixes(source)
	binding := expression.NewSimpleBinding(variable, suffixes)
	suffix := expression.NewConcat(expr.Second(), expression.NewConstant("%"))
	sat := expression.NewLike(expression.NewIdentifier(variable), suffix)
	any := expression.NewAny(expression.Bindings{binding}, sat)
	return expression.NewAnd(expr, any), nil
}

func (this *pattern) visitContainsToken(expr *expression.ContainsToken) (interface{}, error) {
	operands := expr.Operands()
	source := operands[0]
	variable, ok := this.tokens[source.String()]
	if !ok {
		return expr, nil
	}

	var tokens expression.Expression
	if len(operands) > 2 {
		tokens = expression.NewTokens(source, operands[2])
	} else {
		tokens = expression.NewTokens(source)
	}

	binding := expression.NewSimpleBinding(variable, tokens)
	sat := expression.NewEq(expression.NewIdentifier(variable), operands[1])
	any := expression.NewAny(expression.Bindings{binding}, sat)
	return expression.NewAnd(expr, any), nil
}

func (this *pattern) visitContainsTokenLike(expr *expression.ContainsTokenLike) (interface{}, error) {
	operands := expr.Operands()
	source := operands[0]
	variable, ok := this.tokens[source.String()]
	if !ok {
		return expr, nil
	}

	var tokens expression.Expression
	if len(operands) > 2 {
		tokens = expression.NewTokens(source, operands[2])
	} else {
		tokens = expression.NewTokens(source)
	}

	binding := expression.NewSimpleBinding(variable, tokens)
	sat := expression.NewLike(expression.NewIdentifier(variable), operands[1])
	any := expression.NewAny(expression.Bindings{binding}, sat)
	return expression.NewAnd(expr, any), nil
}

func (this *pattern) visitContainsTokenRegexp(expr *expression.ContainsTokenRegexp) (interface{}, error) {
	operands := expr.Operands()
	source := operands[0]
	variable, ok := this.tokens[source.String()]
	if !ok {
		return expr, nil
	}

	var tokens expression.Expression
	if len(operands) > 2 {
		tokens = expression.NewTokens(source, operands[2])
	} else {
		tokens = expression.NewTokens(source)
	}

	binding := expression.NewSimpleBinding(variable, tokens)
	sat := expression.NewRegexpLike(expression.NewIdentifier(variable), operands[1])
	any := expression.NewAny(expression.Bindings{binding}, sat)
	return expression.NewAnd(expr, any), nil
}

func (this *pattern) visitRegexpContains(expr *expression.RegexpContains) (interface{}, error) {
	source := expr.First()
	variable, ok := this.suffixes[source.String()]
	if !ok {
		return expr, nil
	}

	suffixes := expression.NewSuffixes(source)
	binding := expression.NewSimpleBinding(variable, suffixes)
	suffix := expression.NewConcat(expr.Second(), expression.NewConstant("(.*)"))
	sat := expression.NewRegexpLike(expression.NewIdentifier(variable), suffix)
	any := expression.NewAny(expression.Bindings{binding}, sat)
	return expression.NewAnd(expr, any), nil
}

func (this *pattern) visitRegexpLike(expr *expression.RegexpLike) (interface{}, error) {
	source := expr.First()
	variable, ok := this.suffixes[source.String()]
	if !ok {
		return expr, nil
	}

	suffixes := expression.NewSuffixes(source)
	binding := expression.NewSimpleBinding(variable, suffixes)
	suffix := expression.NewRegexpSuffix(expr.Second())
	sat := expression.NewRegexpLike(expression.NewIdentifier(variable), suffix)
	any := expression.NewAny(expression.Bindings{binding}, sat)
	return expression.NewAnd(expr, any), nil
}

func collectPatternIndexes(pred expression.Expression, indexes []datastore.Index,
	formalizer *expression.Formalizer, suffixes, tokens map[string]string) {

	var err error
outer:
	for _, index := range indexes {
		cond := index.Condition()
		if cond != nil {
			cond = cond.Copy()

			cond, err = formalizer.Map(cond)
			if err != nil {
				continue
			}

			dnf := base.NewDNF(cond, true, true)
			cond, err = dnf.Map(cond)
			if err != nil {
				return
			}

			if !base.SubsetOf(pred, cond) {
				continue
			}
		}

		for _, key := range index.RangeKey() {
			if all, ok := key.(*expression.All); ok {
				sufVar := _DEFAULT_SUFFIXES_VARIABLE
				suf, _ := all.Array().(*expression.Suffixes)

				tokVar := _DEFAULT_SUFFIXES_VARIABLE
				tok, _ := all.Array().(*expression.Tokens)

				if array, ok := all.Array().(*expression.Array); ok && len(array.Bindings()) == 1 {
					binding := array.Bindings()[0]

					if variable, ok := array.ValueMapping().(*expression.Identifier); ok &&
						variable.Identifier() == binding.Variable() {

						if suf, ok = binding.Expression().(*expression.Suffixes); ok {
							sufVar = binding.Variable()
						}

						if tok, ok = binding.Expression().(*expression.Tokens); ok {
							tokVar = binding.Variable()
						}
					}
				}

				if suf != nil {
					op := suf.Operand().Copy()
					op, err = formalizer.Map(op)
					if err != nil {
						continue outer
					}

					suffixes[op.String()] = sufVar
					continue outer
				}

				if tok != nil {
					op := tok.Operands()[0].Copy()
					op, err = formalizer.Map(op)
					if err != nil {
						continue outer
					}

					tokens[op.String()] = tokVar
					continue outer
				}
			}
		}
	}
}

var _PATTERN_INDEX_POOL = util.NewStringStringPool(64)
var _DEFAULT_SUFFIXES_VARIABLE = "s"
var _DEFAULT_TOKENS_VARIABLE = "t"
