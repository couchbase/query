/*
Copyright 2016-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL.txt.
*/

package expression

import (
	"testing"

	"github.com/couchbase/query/value"
)

func testObjectPut(e1, e2, e3 Expression, er value.Value, t *testing.T) {
	eop := NewObjectPut(e1, e2, e3)
	rv, err := eop.Evaluate(nil, nil)
	if err != nil {
		t.Errorf("received error %v", err)
	}
	if er.Collate(rv) != 0 {
		t.Errorf("mismatch received %v expected %v", rv.Actual(), er.Actual())
	}
}

func TestObjectPut_add(t *testing.T) {

	/* tests insert of value in array at start */
	e1 := NewConstant(value.NewValue(map[string]interface{}{"f1": 1}))
	e2 := NewConstant("f2")
	e3 := NewConstant(2)
	er := value.NewValue(map[string]interface{}{"f1": 1, "f2": 2})
	testObjectPut(e1, e2, e3, er, t)
}

func TestObjectPut_replace(t *testing.T) {

	/* tests insert of value in array at start */
	e1 := NewConstant(value.NewValue(map[string]interface{}{"f1": 1, "f2": 2}))
	e2 := NewConstant("f2")
	e3 := NewConstant(3)
	er := value.NewValue(map[string]interface{}{"f1": 1, "f2": 3})
	testObjectPut(e1, e2, e3, er, t)
}

func TestObjectPut_remove(t *testing.T) {

	/* tests insert of value in array at start */
	e1 := NewConstant(value.NewValue(map[string]interface{}{"f1": 1, "f2": 2}))
	e2 := NewConstant("f2")
	e3 := NewConstant(value.MISSING_VALUE)
	er := value.NewValue(map[string]interface{}{"f1": 1})
	testObjectPut(e1, e2, e3, er, t)
}

func testObjectAdd(e1, e2, e3 Expression, er value.Value, fail bool, t *testing.T) {
	eop := NewObjectAdd(e1, e2, e3)
	rv, err := eop.Evaluate(nil, nil)
	if err != nil {
		if fail && rv.Actual() == nil {
			return
		}
		t.Errorf("received error %v", err)
	} else if fail {
		t.Errorf("error expected, received success")
	}
	if er.Collate(rv) != 0 {
		t.Errorf("mismatch received %v expected %v", rv.Actual(), er.Actual())
	}
}

func TestObjectAdd_add(t *testing.T) {

	/* tests insert of value in array at start */
	e1 := NewConstant(value.NewValue(map[string]interface{}{"f1": 1}))
	e2 := NewConstant("f2")
	e3 := NewConstant(2)
	er := value.NewValue(map[string]interface{}{"f1": 1, "f2": 2})
	testObjectAdd(e1, e2, e3, er, false, t)
}

func TestObjectAdd_replace(t *testing.T) {

	/* tests insert of value in array at start */
	e1 := NewConstant(value.NewValue(map[string]interface{}{"f1": 1, "f2": 2}))
	e2 := NewConstant("f2")
	e3 := NewConstant(3)
	er := value.NewValue(map[string]interface{}{"f1": 1, "f2": 2})
	testObjectAdd(e1, e2, e3, er, false, t)
}

func TestObjectAdd_remove(t *testing.T) {

	/* tests insert of value in array at start */
	e1 := NewConstant(value.NewValue(map[string]interface{}{"f1": 1, "f2": 2}))
	e2 := NewConstant("f2")
	e3 := NewConstant(value.MISSING_VALUE)
	er := value.NewValue(map[string]interface{}{"f1": 1, "f2": 2})
	testObjectAdd(e1, e2, e3, er, false, t)
}

func testObjectRemove(e1, e2 Expression, er value.Value, t *testing.T) {
	eop := NewObjectRemove(e1, e2)
	rv, err := eop.Evaluate(nil, nil)
	if err != nil {
		t.Errorf("received error %v", err)
	}
	if er.Collate(rv) != 0 {
		t.Errorf("mismatch received %v expected %v", rv.Actual(), er.Actual())
	}
}

func TestObjectRemove_remove(t *testing.T) {

	/* tests insert of value in array at start */
	e1 := NewConstant(value.NewValue(map[string]interface{}{"f1": 1, "f2": 2}))
	e2 := NewConstant("f2")
	er := value.NewValue(map[string]interface{}{"f1": 1})
	testObjectRemove(e1, e2, er, t)
}

func TestObjectRemove_null(t *testing.T) {

	/* tests insert of value in array at start */
	e1 := NewConstant(value.NULL_VALUE)
	e2 := NewConstant("f2")
	er := value.NULL_VALUE
	testObjectRemove(e1, e2, er, t)
}
