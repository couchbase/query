//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package couchbase

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
	"testing"
)

func constant(constant interface{}) expression.Expression {
	return expression.NewConstant(value.NewValue(constant))
}

func constantArray(constant []interface{}) expression.Expression {
	return expression.NewArrayConstruct(expression.NewConstant(value.NewValue(constant)))
}

func TestConverter(t *testing.T) {

	s1 := NewJSConverter().Visit(
		expression.NewLT(constant("a"), constant("b")))

	s2 := "(\"a\" < \"b\")"

	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(
		expression.NewBetween(constant("a"),
			constant("b"),
			constant("c")))

	s2 = "(\"a\" > \"b\" && \"a\" < \"c\")"

	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewAdd(
		expression.NewSub(constant("a"), constant("b")),
		expression.NewDiv(constant("a"), constant("b"))))

	s2 = "((\"a\" - \"b\") + (\"a\" / \"b\"))"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewLength(constant("abc")))
	s2 = "\"abc\".length"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewUpper(constant("abc")))
	s2 = "\"abc\".toUpperCase()"

	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewStrToMillis(constant("Wed, 09 Aug 1995 00:00:00")))
	s2 = "Date.parse(\"Wed, 09 Aug 1995 00:00:00\")"

	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewContains(constant("dfgabc"), constant("abc")))
	s2 = "\"dfgabc\".indexOf(\"abc\")"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewSubstr0(constant("dfgabc"), constant(1), constant(4)))
	s2 = "\"dfgabc\".substring(1,5)"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewSubstr1(constant("dfgabc"), constant(1), constant(4)))
	s2 = "\"dfgabc\".substring(0,4)"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewAdd(expression.NewContains(constant("dfgabc"), constant("abc")), expression.NewSubstr0(constant("dfgabc"), constant(1), constant(4))))
	s2 = "(\"dfgabc\".indexOf(\"abc\") + \"dfgabc\".substring(1,5))"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	doc := expression.NewIdentifier("bucket")
	m1 := expression.NewField(doc, expression.NewFieldName("id", false))
	m2 := expression.NewField(doc, expression.NewFieldName("type", false))

	s1 = NewJSConverter().Visit(expression.NewOr(
		expression.NewUpper(m1), expression.NewLower(m2)))

	s2 = "(`bucket`.`id`.toUpperCase() || `bucket`.`type`.toLowerCase())"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	doc = expression.NewIdentifier("bucket")
	m1 = expression.NewField(doc, expression.NewFieldName("geo", false))
	m2 = expression.NewField(m1, expression.NewFieldName("accuracy", false))

	s1 = NewJSConverter().Visit(m2)
	s2 = "`bucket`.`geo`.`accuracy`"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	doc = expression.NewIdentifier("bucket")
	m1 = expression.NewField(doc, expression.NewElement(expression.NewFieldName("address", false), constant(0)))

	s1 = NewJSConverter().Visit(m1)
	s2 = "`bucket`.`address`[0]"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewLength(expression.NewElement(doc, expression.NewFieldName("type", false))))
	s2 = "`bucket`[`type`].length"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

}
