/*
Copyright 2015-Present Couchbase, Inc.

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

func testArrayInsert_eval(e1, e2, e3 Expression, er value.Value, t *testing.T) {
	eai := NewArrayInsert(e1, e2, e3)
	rv, err := eai.Evaluate(nil, nil)
	if err != nil {
		t.Errorf("received error %v", err)
	}
	if er.Collate(rv) != 0 {
		t.Errorf("mismatch received %v expected %v", rv.Actual(), er.Actual())
	}
}

func TestArrayInsert_start(t *testing.T) {

	/* tests insert of value in array at start */
	e1 := NewConstant([]interface{}{1, 2, 3})
	e2 := NewConstant(0)
	e3 := NewConstant(4)
	er := value.NewValue([]interface{}{4, 1, 2, 3})
	testArrayInsert_eval(e1, e2, e3, er, t)
}

func TestArrayInsert_end(t *testing.T) {

	/* tests insert of value in array at end */
	e1 := NewConstant([]interface{}{1, 2, 3})
	e2 := NewConstant(0)
	e3 := NewConstant(4)
	er := value.NewValue([]interface{}{4, 1, 2, 3})
	testArrayInsert_eval(e1, e2, e3, er, t)
}

func TestArrayInsert_null(t *testing.T) {

	/* tests insert of value in null array */
	e1 := NewConstant(nil)
	e2 := NewConstant(0)
	e3 := NewConstant(4)
	er := value.NewValue(nil)
	testArrayInsert_eval(e1, e2, e3, er, t)
}

func TestArrayInsert_missing(t *testing.T) {

	/* tests insert of value in missing array */
	e1 := NewConstant(value.MISSING_VALUE)
	e2 := NewConstant(0)
	e3 := NewConstant(4)
	er := value.MISSING_VALUE
	testArrayInsert_eval(e1, e2, e3, er, t)
}
