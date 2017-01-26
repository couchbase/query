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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
)

func PatternFor(pred expression.Expression, indexes []datastore.Index,
	formalizer *expression.Formalizer) (
	expression.Expression, error) {

	suffixes := _PATTERN_INDEX_POOL.Get()
	defer _PATTERN_INDEX_POOL.Put(suffixes)
	tokens := _PATTERN_INDEX_POOL.Get()
	defer _PATTERN_INDEX_POOL.Put(tokens)

	collectPatternIndexes(pred, indexes, formalizer, suffixes, tokens)
	if len(suffixes) == 0 && len(tokens) == 0 {
		return pred, nil
	}

	var err error
	pred = pred.Copy()
	dnf := NewDNF(pred, false)
	pred, err = dnf.Map(pred)
	if err != nil {
		return nil, err
	}

	pat := newPattern(suffixes, tokens)
	rv, err := pred.Accept(pat)
	if err != nil {
		return nil, err
	}

	return rv.(expression.Expression), nil
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
	case *expression.RegexpLike:
		return this.visitRegexpLike(expr)
	case *expression.HasToken:
		return this.visitHasToken(expr)
	default:
		return expr, nil
	}
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

func (this *pattern) visitHasToken(expr *expression.HasToken) (interface{}, error) {
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

			dnf := NewDNF(cond, true)
			cond, err = dnf.Map(cond)
			if err != nil {
				return
			}

			if !SubsetOf(pred, cond) {
				continue
			}
		}

		for _, key := range index.RangeKey() {
			if all, ok := key.(*expression.All); ok {
				if array, ok := all.Array().(*expression.Array); ok && len(array.Bindings()) == 1 {
					binding := array.Bindings()[0]

					if variable, ok := array.ValueMapping().(*expression.Identifier); ok &&
						variable.Identifier() == binding.Variable() {

						if suf, ok := binding.Expression().(*expression.Suffixes); ok {
							op := suf.Operand()
							op, err = formalizer.Map(op)
							if err != nil {
								continue outer
							}

							suffixes[op.String()] = binding.Variable()
							continue outer
						}

						if tok, ok := binding.Expression().(*expression.Tokens); ok {
							op := tok.Operands()[0]
							op, err = formalizer.Map(op)
							if err != nil {
								continue outer
							}

							tokens[op.String()] = binding.Variable()
							continue outer
						}
					}
				}
			}
		}
	}
}

var _PATTERN_INDEX_POOL = util.NewStringStringPool(64)
