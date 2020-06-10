//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

func FlattenOr(or *Or) (*Or, bool) {
	length, flatten, truth := orLength(or)
	if !flatten || truth {
		return or, truth
	}

	buffer := make(Expressions, 0, length)
	terms := _STRING_EXPRESSION_POOL.Get()
	defer _STRING_EXPRESSION_POOL.Put(terms)
	buffer = orTerms(or, buffer, terms)

	return NewOr(buffer...), false
}

func FlattenAnd(and *And) (*And, bool) {
	length, flatten, truth := andLength(and)
	if !flatten || !truth {
		return and, truth
	}

	buffer := make(Expressions, 0, length)
	terms := _STRING_EXPRESSION_POOL.Get()
	defer _STRING_EXPRESSION_POOL.Put(terms)
	buffer = andTerms(and, buffer, terms)

	return NewAnd(buffer...), true
}

func orLength(or *Or) (length int, flatten, truth bool) {
	l := 0
	for _, op := range or.Operands() {
		switch op := op.(type) {
		case *Or:
			l, _, truth = orLength(op)
			if truth {
				return
			}
			length += l
			flatten = true
		default:
			val := op.Value()
			if val != nil {
				if val.Truth() {
					truth = true
					return
				}
			} else {
				length++
			}
		}
	}

	return
}

func andLength(and *And) (length int, flatten, truth bool) {
	truth = true
	l := 0
	for _, op := range and.Operands() {
		switch op := op.(type) {
		case *And:
			l, _, truth = andLength(op)
			if !truth {
				return
			}
			length += l
			flatten = true
		default:
			val := op.Value()
			if val != nil {
				if !val.Truth() {
					truth = false
					return
				}
			} else {
				length++
			}
		}
	}

	return
}

func orTerms(or *Or, buffer Expressions,
	terms map[string]Expression) Expressions {
	for _, op := range or.Operands() {
		switch op := op.(type) {
		case *Or:
			buffer = orTerms(op, buffer, terms)
		default:
			val := op.Value()
			if val == nil || val.Truth() {
				str := op.String()
				if _, found := terms[str]; !found {
					terms[str] = op
					buffer = append(buffer, op)
				}
			}
		}
	}

	return buffer
}

func andTerms(and *And, buffer Expressions,
	terms map[string]Expression) Expressions {
	for _, op := range and.Operands() {
		switch op := op.(type) {
		case *And:
			buffer = andTerms(op, buffer, terms)
		default:
			val := op.Value()
			if val == nil || !val.Truth() {
				str := op.String()
				if _, found := terms[str]; !found {
					terms[str] = op
					buffer = append(buffer, op)
				}
			}
		}
	}

	return buffer
}

var _STRING_EXPRESSION_POOL = NewStringExpressionPool(1024)
