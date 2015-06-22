//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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

	s1 = NewJSConverter().Visit(expression.NewSubstr(constant("dfgabc"), constant(1), constant(4)))
	s2 = "\"dfgabc\".substring(1,4)"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewAdd(expression.NewContains(constant("dfgabc"), constant("abc")), expression.NewSubstr(constant("dfgabc"), constant(1), constant(4))))
	s2 = "(\"dfgabc\".indexOf(\"abc\") + \"dfgabc\".substring(1,4))"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	doc := expression.NewIdentifier("bucket")
	m1 := expression.NewField(doc, expression.NewFieldName("id"))
	m2 := expression.NewField(doc, expression.NewFieldName("type"))

	s1 = NewJSConverter().Visit(expression.NewOr(
		expression.NewUpper(m1), expression.NewLower(m2)))

	s2 = "(`bucket`.`id`.toUpperCase() || `bucket`.`type`.toLowerCase())"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	doc = expression.NewIdentifier("bucket")
	m1 = expression.NewField(doc, expression.NewFieldName("geo"))
	m2 = expression.NewField(m1, expression.NewFieldName("accuracy"))

	s1 = NewJSConverter().Visit(m2)
	s2 = "`bucket`.`geo`.`accuracy`"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	doc = expression.NewIdentifier("bucket")
	m1 = expression.NewField(doc, expression.NewElement(expression.NewFieldName("address"), constant(0)))

	s1 = NewJSConverter().Visit(m1)
	s2 = "`bucket`.`address`[0]"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

	s1 = NewJSConverter().Visit(expression.NewLength(expression.NewElement(doc, expression.NewFieldName("type"))))
	s2 = "`bucket`[`type`].length"
	if s1 != s2 {
		t.Errorf(" mismatch s1 %s s2 %s", s1, s2)
	}

}
