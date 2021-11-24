/*
Copyright 2016-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package value

import (
	"reflect"
	"testing"
)

func getSecondLevelArray(m interface{}, first int, second int) string {
	mValue, ok := m.(Value)
	if ok {
		m = mValue.Actual()
	}
	topArr, ok := m.([]interface{})
	if !ok {
		return "BAD_FIRST_INTERFACE"
	}

	s := topArr[first]
	sValue, ok := s.(Value)
	if ok {
		s = sValue.Actual()
	}
	bottomArr, ok := s.([]interface{})
	if !ok {
		return "BAD_SECOND_INTERFACE"
	}
	r := bottomArr[second]

	rs, ok := r.(string)
	if ok {
		return rs
	}

	sv, ok := r.(stringValue)
	if ok {
		return sv.Actual().(string)
	}

	return "(bad return value of type " + reflect.TypeOf(r).String() + ")"
}

func TestShallowCopiedArraySharesElements(t *testing.T) {
	lowArr := make([]interface{}, 3)
	lowArr[0] = "lowval0"
	lowArr[1] = "lowval1"
	lowArr[2] = "originalvalue"

	highArr := make([]interface{}, 2)
	highArr[0] = "highval0"
	highArr[1] = lowArr

	ov := NewValue(highArr)
	_, ok := ov.(sliceValue)
	if !ok {
		t.Fatalf("Original object is not of type objectValue.")
	}

	copy := ov.Copy()
	_, ok = copy.(copiedSliceValue)
	if !ok {
		t.Fatalf("Copy is not of type copiedArrayValue, instead %s.", reflect.TypeOf(copy).String())
	}
	retOv := getSecondLevelArray(ov.Actual(), 1, 2)
	if retOv != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from original: %s, expected originalvalue.", retOv)
	}
	retCopy := getSecondLevelArray(copy.Actual(), 1, 2)
	if retCopy != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from copy: %s, expected originalvalue.", retCopy)
	}

	// Now change the underlying value. Both the original and the copy should reflect the change.
	lowArr[2] = "changedvalue"
	retOv = getSecondLevelArray(ov.Actual(), 1, 2)
	if retOv != "changedvalue" {
		t.Fatalf("Unexpected value retrieved from original: %s, expected changedvalue.", retOv)
	}
	retCopy = getSecondLevelArray(copy.Actual(), 1, 2)
	if retCopy != "changedvalue" {
		t.Fatalf("Unexpected value retrieved from copy: %s, expected changedvalue.", retCopy)
	}
}

func TestDeepCopiedArrayDoesNotShareElements(t *testing.T) {
	lowArr := make([]interface{}, 3)
	lowArr[0] = "lowval0"
	lowArr[1] = "lowval1"
	lowArr[2] = "originalvalue"

	highArr := make([]interface{}, 2)
	highArr[0] = "highval0"
	highArr[1] = lowArr

	ov := NewValue(highArr)
	_, ok := ov.(sliceValue)
	if !ok {
		t.Fatalf("Original object is not of type objectValue.")
	}

	copy := ov.CopyForUpdate()
	_, ok = copy.(*listValue)
	if !ok {
		t.Fatalf("Copy is not of type *listValue, instead %s.", reflect.TypeOf(copy).String())
	}
	retOv := getSecondLevelArray(ov.Actual(), 1, 2)
	if retOv != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from original: %s, expected originalvalue.", retOv)
	}
	retCopy := getSecondLevelArray(copy.Actual(), 1, 2)
	if retCopy != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from copy: %s, expected originalvalue.", retCopy)
	}

	// Now change the underlying value. Only the original should reflect the change.
	lowArr[2] = "changedvalue"
	retOv = getSecondLevelArray(ov.Actual(), 1, 2)
	if retOv != "changedvalue" {
		t.Fatalf("Unexpected value retrieved from original: %s, expected changedvalue.", retOv)
	}
	retCopy = getSecondLevelArray(copy.Actual(), 1, 2)
	if retCopy != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from copy: %s, expected originalvalue.", retCopy)
	}
}
