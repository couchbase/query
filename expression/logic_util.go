//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/util"
)

func FlattenOr(or *Or) (*Or, bool) {
	length, flatten, truth := orLength(or)
	if !flatten || truth {
		return or, truth
	}

	buffer := make(Expressions, 0, length)
	terms := _STRING_BOOL_POOL.Get()
	buffer = orTerms(or, buffer, terms)
	_STRING_BOOL_POOL.Put(terms)

	return NewOr(buffer...), false
}

func FlattenAnd(and *And) (*And, bool) {
	length, flatten, truth := andLength(and)
	if !flatten || !truth {
		return and, truth
	}

	buffer := make(Expressions, 0, length)
	terms := _STRING_BOOL_POOL.Get()
	buffer = andTerms(and, buffer, terms)
	_STRING_BOOL_POOL.Put(terms)

	return NewAnd(buffer...), true
}

func FlattenOrNoDedup(or *Or) (*Or, bool) {
	length, flatten, truth := orLength(or)
	if !flatten || truth {
		return or, truth
	}

	buffer := make(Expressions, 0, length)
	buffer = orTerms(or, buffer, nil)

	return NewOr(buffer...), false
}

func FlattenAndNoDedup(and *And) (*And, bool) {
	length, flatten, truth := andLength(and)
	if !flatten || !truth {
		return and, truth
	}

	buffer := make(Expressions, 0, length)
	buffer = andTerms(and, buffer, nil)

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
				flatten = true
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
				flatten = true
			} else {
				length++
			}
		}
	}

	return
}

func orTerms(or *Or, buffer Expressions, terms map[string]bool) Expressions {
	for _, op := range or.Operands() {
		switch op := op.(type) {
		case *Or:
			buffer = orTerms(op, buffer, terms)
		default:
			val := op.Value()
			if val == nil || val.Truth() {
				if terms == nil {
					buffer = append(buffer, op)
				} else {
					str := op.String()
					if _, found := terms[str]; !found {
						terms[str] = true
						buffer = append(buffer, op)
					}
				}
			}
		}
	}

	return buffer
}

func andTerms(and *And, buffer Expressions, terms map[string]bool) Expressions {
	for _, op := range and.Operands() {
		switch op := op.(type) {
		case *And:
			buffer = andTerms(op, buffer, terms)
		default:
			val := op.Value()
			if val == nil || !val.Truth() {
				if terms == nil {
					buffer = append(buffer, op)
				} else {
					str := op.String()
					if _, found := terms[str]; !found {
						terms[str] = true
						buffer = append(buffer, op)
					}
				}
			}
		}
	}

	return buffer
}

var _STRING_BOOL_POOL = util.NewStringBoolPool(32)
