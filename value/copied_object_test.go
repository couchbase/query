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

func getSecondLevel(m interface{}, first string, second string) string {
	mValue, ok := m.(Value)
	if ok {
		m = mValue.Actual()
	}
	topMap, ok := m.(map[string]interface{})
	if !ok {
		return "BAD_FIRST_INTERFACE"
	}

	s := topMap[first]
	sValue, ok := s.(Value)
	if ok {
		s = sValue.Actual()
	}
	bottomMap, ok := s.(map[string]interface{})
	if !ok {
		return "BAD_SECOND_INTERFACE"
	}
	r := bottomMap[second]

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

func TestShallowCopySharesObjects(t *testing.T) {
	lowMap := make(map[string]interface{})
	lowMap["element"] = "originalvalue"
	highMap := make(map[string]interface{})
	highMap["where"] = lowMap

	ov := NewValue(highMap)
	_, ok := ov.(objectValue)
	if !ok {
		t.Fatalf("Original object is not of type objectValue.")
	}

	copy := ov.Copy()
	_, ok = copy.(copiedObjectValue)
	if !ok {
		t.Fatalf("Copy is not of type copiedObjectValue, instead %s.", reflect.TypeOf(copy).String())
	}

	retOv := getSecondLevel(ov.Actual(), "where", "element")
	if retOv != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from original: %s, expected originalvalue.", retOv)
	}
	retCopy := getSecondLevel(copy.Actual(), "where", "element")
	if retCopy != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from copy: %s, expected originalvalue.", retCopy)
	}

	// Now change the underlying value. Both the original and the copy should reflect the change.
	lowMap["element"] = "changedvalue"
	retOv = getSecondLevel(ov.Actual(), "where", "element")
	if retOv != "changedvalue" {
		t.Fatalf("Unexpected value retrieved from original: %s, expected changedvalue.", retOv)
	}
	retCopy = getSecondLevel(copy.Actual(), "where", "element")
	if retCopy != "changedvalue" {
		t.Fatalf("Unexpected value retrieved from copy: %s, expected changedvalue.", retCopy)
	}
}

func TestDeepCopyDoesNotShareObjects(t *testing.T) {
	lowMap := make(map[string]interface{})
	lowMap["element"] = "originalvalue"
	highMap := make(map[string]interface{})
	highMap["where"] = lowMap

	ov := NewValue(highMap)
	_, ok := ov.(objectValue)
	if !ok {
		t.Fatalf("Original object is not of type objectValue.")
	}

	copy := ov.CopyForUpdate()
	_, ok = copy.(objectValue)
	if !ok {
		t.Fatalf("Copy is not of type objectValue, instead %s.", reflect.TypeOf(copy).String())
	}

	retOv := getSecondLevel(ov.Actual(), "where", "element")
	if retOv != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from original: %s, expected originalvalue.", retOv)
	}

	retCopy := getSecondLevel(copy.Actual(), "where", "element")
	if retCopy != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from copy: %s, expected originalvalue.", retCopy)
	}

	// Now change the underlying value. Only the original should change.
	lowMap["element"] = "changedvalue"
	retOv = getSecondLevel(ov.Actual(), "where", "element")
	if retOv != "changedvalue" {
		t.Fatalf("Unexpected value retrieved from original: %s, expected changedvalue.", retOv)
	}
	retCopy = getSecondLevel(copy.Actual(), "where", "element")
	if retCopy != "originalvalue" {
		t.Fatalf("Unexpected value retrieved from copy: %s, expected originalvalue.", retCopy)
	}
}
