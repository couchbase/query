//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import (
	"compress/gzip"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	// "time"

	json "github.com/couchbase/go_json"
)

var codeJSON []byte

func init() {
	f, err := os.Open("testdata/code.json.gz")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(gz)
	if err != nil {
		panic(err)
	}

	codeJSON = data
}

func TestTypeRecognition(t *testing.T) {

	var tests = []struct {
		input        []byte
		expectedType Type
	}{
		{[]byte(`asdf`), BINARY},
		{[]byte(`null`), NULL},
		{[]byte(`3.65`), NUMBER},
		{[]byte(`-3.65`), NUMBER},
		{[]byte(`"hello"`), STRING},
		{[]byte(`["hello"]`), ARRAY},
		{[]byte(`{"hello":7}`), OBJECT},

		// with misc whitespace
		{[]byte(` asdf`), BINARY},
		{[]byte(` null`), NULL},
		{[]byte(` 3.65`), NUMBER},
		{[]byte(` "hello"`), STRING},
		{[]byte("\t[\"hello\"]"), ARRAY},
		{[]byte("\n{\"hello\":7}"), OBJECT},
	}

	for _, test := range tests {
		val := NewValue(test.input)
		actualType := val.Type()
		if actualType != test.expectedType {
			t.Errorf("Expected type of %s to be %d, got %d", string(test.input), test.expectedType, actualType)
		}
	}
}

func TestFieldAccess(t *testing.T) {

	val := NewValue([]byte(`{"name":"marty","address":{"street":"sutton oaks"}}`))

	var tests = []struct {
		field  string
		result Value
		ok     bool
	}{
		{"name", stringValue("marty"), true},
		{"address", &parsedValue{raw: []byte(`{"street":"sutton oaks"}`), parsedType: OBJECT}, true},
		{"dne", missingField("dne"), false},
	}

	for _, test := range tests {
		result, ok := val.Field(test.field)

		if ok != test.ok {
			t.Errorf("Expected ok=%v, got ok=%v for field %s.", test.ok, ok, test.field)
		}

		// parsed types have a state and can't be compared with DeepEqual
		// missing types are never equal and must be compared by type
		if !((result.Equals(test.result) == TRUE_VALUE) ||
			(test.result.Type() == MISSING && result.Type() == MISSING)) {
			t.Errorf("Expected result=%v, got result=%v for field %s.", test.result, result, test.field)
		}
	}
}

func TestIndexAccess(t *testing.T) {

	val := NewValue([]byte(`["marty",{"type":"contact"}]`))

	var tests = []struct {
		index  int
		result Value
		ok     bool
	}{
		{0, stringValue("marty"), true},
		{1, &parsedValue{raw: []byte(`{"type":"contact"}`), parsedType: OBJECT}, true},
		{2, missingIndex(2), false},
	}

	for _, test := range tests {
		result, ok := val.Index(test.index)

		if !reflect.DeepEqual(ok, test.ok) {
			t.Errorf("Expected ok=%v, got ok=%v for index %d.", test.ok, ok, test.index)
		}

		// parsed types have a state and can't be compared with DeepEqual
		// missing types are never equal and must be compared by type
		if !((result.Equals(test.result) == TRUE_VALUE) ||
			(test.result.Type() == MISSING && result.Type() == MISSING)) {
			t.Errorf("Expected result=%v, got result=%v for index %d.", test.result, result, test.index)
		}
	}

	val = NewValue([]interface{}{"marty", map[string]interface{}{"type": "contact"}})

	tests = []struct {
		index  int
		result Value
		ok     bool
	}{
		{0, stringValue("marty"), true},
		{1, objectValue(map[string]interface{}{"type": "contact"}), true},
		{2, missingIndex(2), false},
	}

	for _, test := range tests {
		result, ok := val.Index(test.index)

		if !reflect.DeepEqual(ok, test.ok) {
			t.Errorf("Expected ok=%v, got ok=%v for index %d.", test.ok, ok, test.index)
		}

		if !reflect.DeepEqual(result, test.result) {
			t.Errorf("Expected result=%v, got result=%v for index %d.", test.result, result, test.index)
		}
	}
}

func TestAttachments(t *testing.T) {
	val := NewAnnotatedValue([]byte(`{"name":"marty","address":{"street":"sutton oaks"}}`))
	val.SetId("doc1")

	meta := val.GetMeta()
	if meta == nil {
		t.Errorf("metadata missing")
	} else {
		id := meta["id"].(string)
		if id != "doc1" {
			t.Errorf("Expected id doc1, got %v", id)
		}
	}
	id := val.GetId()
	if id == nil {
		t.Errorf("id missing")
	} else {
		switch id := id.(type) {
		case string:
			if id != "doc1" {
				t.Errorf("Expected id doc1, got %v", id)
			}
		default:
			t.Errorf("Id is not a string")
		}
	}
}

func TestRealWorkflow(t *testing.T) {
	// get a doc from some source
	doc := NewAnnotatedValue([]byte(`{"name":"marty","address":{"street":"sutton oaks"}}`))
	doc.SetId("doc1")

	// mutate the document somehow
	active := NewValue(true)
	doc.SetField("active", active)

	testActiveVal, ok := doc.Field("active")
	if !ok {
		t.Errorf("Error accessing doc.active")
	}

	testActive := testActiveVal.Actual()
	if testActive != true {
		t.Errorf("Expected active true, got %v", testActive)
	}

	// create a reference to doc
	top := NewValue(map[string]interface{}{"bucket": doc, "another": "rad"})

	testDoc, ok := top.Field("bucket")
	if !ok {
		t.Errorf("Error accessing top.bucket")
	}
	if !reflect.DeepEqual(testDoc, doc) {
		t.Errorf("Expected doc %v to match testDoc %v", doc, testDoc)
	}

	testRad, ok := top.Field("another")
	if !ok {
		t.Errorf("Error accessing top.another")
	}
	expectedRad := NewValue("rad")
	if !reflect.DeepEqual(testRad, expectedRad) {
		t.Errorf("Expected %v, got %v for rad", expectedRad, testRad)
	}

	// now project some value from the doc to a top-level alias
	addressVal, ok := doc.Field("address")
	if !ok {
		t.Errorf("Error accessing doc.address")
	}

	top.SetField("a", addressVal)

	// now access "a.street"
	aVal, ok := top.Field("a")
	if !ok {
		t.Errorf("Error accessing top.a")
	}

	streetVal, ok := aVal.Field("street")
	if !ok {
		t.Errorf("Error accessing a.street")
	}

	street := streetVal.Actual()
	if street != "sutton oaks" {
		t.Errorf("Expected sutton oaks, got %v", street)
	}
}

func TestMissing(t *testing.T) {

	x := missingField("property")
	err := x.Error()

	if err != "Missing field or index property." {
		t.Errorf("Expected 'Missing field or index property.', got %v.", err)
	}

	y := missingField("")
	err = y.Error()

	if err != "Missing field or index." {
		t.Errorf("Expected 'Missing field or index.', got %v.", err)
	}

}

func TestValue(t *testing.T) {
	var tests = []struct {
		input         Value
		expectedValue interface{}
	}{
		{NewValue(nil), nil},
		{NewValue(true), true},
		{NewValue(false), false},
		{NewValue(1.0), 1.0},
		{NewValue(3.14), 3.14},
		{NewValue(-7.0), -7.0},
		{NewValue(""), ""},
		{NewValue("marty"), "marty"},
		{NewValue([]interface{}{"marty"}), []interface{}{"marty"}},
		{NewValue([]interface{}{NewValue("marty2")}), []interface{}{stringValue("marty2")}},
		{NewValue(map[string]interface{}{"marty": "cool"}), map[string]interface{}{"marty": "cool"}},
		{NewValue(map[string]interface{}{"marty3": NewValue("cool")}), map[string]interface{}{"marty3": stringValue("cool")}},
		{NewValue([]byte("null")), nil},
		{NewValue([]byte("true")), true},
		{NewValue([]byte("false")), false},
		{NewValue([]byte("1")), 1.0},
		{NewValue([]byte("3.14")), 3.14},
		{NewValue([]byte("-7")), -7.0},
		{NewValue([]byte("\"\"")), ""},
		{NewValue([]byte("\"marty\"")), "marty"},
		{NewValue([]byte("[\"marty\"]")), []interface{}{"marty"}},
		{NewValue([]byte("{\"marty\": \"cool\"}")), map[string]interface{}{"marty": "cool"}},
		{NewValue([]byte("abc")), []byte("abc")},
		// new value from existing value
		{NewValue(NewValue(true)), true},
	}

	for i, test := range tests {
		val := test.input.Actual()
		if !reflect.DeepEqual(val, test.expectedValue) {
			t.Errorf("Expected %#v, got %#v for %#v at index %d.", test.expectedValue, val, test.input, i)
		}
	}
}

func TestValueOverlay(t *testing.T) {
	val := NewValue([]byte("{\"marty\": \"cool\"}"))
	val.SetField("marty", "ok")
	expectedVal := map[string]interface{}{"marty": "ok"}
	actualVal := val.Actual()
	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Expected %v, got %v, for value of %v", expectedVal, actualVal, val)
	}

	val = NewValue(map[string]interface{}{"marty": "cool"})
	val.SetField("marty", "ok")
	actualVal = val.Actual()
	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Expected %v, got %v, for value of %v", expectedVal, actualVal, val)
	}

	val = NewValue([]byte("[\"marty\"]"))
	val.SetIndex(0, "gerald")
	expectedVal2 := []interface{}{"gerald"}
	actualVal = val.Actual()
	if !reflect.DeepEqual(expectedVal2, actualVal) {
		t.Errorf("Expected %v, got %v, for value of %v", expectedVal2, actualVal, val)
	}

	val = NewValue([]interface{}{"marty"})
	val.SetIndex(0, "gerald")
	actualVal = val.Actual()
	if !reflect.DeepEqual(expectedVal2, actualVal) {
		t.Errorf("Expected %v, got %v, for value of %v", expectedVal2, actualVal, val)
	}
}

func TestComplexOverlay(t *testing.T) {
	// in this case we start with JSON bytes
	// then add an alias
	// then call value which causes it to be parsed
	// then call value again, which goes through a different field with the already parsed data
	val := NewValue([]byte("{\"marty\": \"cool\"}"))
	val.SetField("marty", "ok")
	expectedVal := map[string]interface{}{"marty": "ok"}
	actualVal := val.Actual()
	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Expected %v, got %v, for value of %v", expectedVal, actualVal, val)
	}
	// now repeat the call to value
	actualVal = val.Actual()
	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Expected %v, got %v, for value of %v", expectedVal, actualVal, val)
	}
}

func TestUnmodifiedValuesFromBytesBackToBytes(t *testing.T) {
	var tests = []struct {
		input         []byte
		expectedBytes []byte
	}{
		{[]byte(`asdf`), []byte(`"<binary (4 b)>"`)},
		{[]byte(`null`), []byte(`null`)},
		{[]byte(`3.65`), []byte(`3.65`)},
		{[]byte(`-3.65`), []byte(`-3.65`)},
		{[]byte(`"hello"`), []byte(`"hello"`)},
		{[]byte(`["hello"]`), []byte(`["hello"]`)},
		{[]byte(`{"hello":7}`), []byte(`{"hello":7}`)},
	}

	for _, test := range tests {
		val := NewValue(test.input)
		out, _ := val.MarshalJSON()
		if !reflect.DeepEqual(out, test.expectedBytes) {
			t.Errorf("Expected %s to be  %s, got %s", string(test.input), string(test.expectedBytes), string(out))
		}
	}
}

func TestUnmodifiedValuesBackToBytes(t *testing.T) {
	var tests = []struct {
		input         Value
		expectedBytes []byte
	}{
		{NewValue(nil), []byte(`null`)},
		{NewValue(3.65), []byte(`3.65`)},
		{NewValue(-3.65), []byte(`-3.65`)},
		{NewValue("hello"), []byte(`"hello"`)},
		{NewValue([]interface{}{"hello"}), []byte(`["hello"]`)},
		{NewValue(map[string]interface{}{"hello": 7.0}), []byte(`{"hello":7}`)},
	}

	for _, test := range tests {
		out, _ := test.input.MarshalJSON()
		if !reflect.DeepEqual(out, test.expectedBytes) {
			t.Errorf("Expected %v to be  %s, got %s", test.input, string(test.expectedBytes), string(out))
		}
	}
}

func BenchmarkLargeValue(b *testing.B) {
	b.SetBytes(int64(len(codeJSON)))

	keys := []interface{}{
		"tree", "kids", 0, "kids", 0, "kids", 0, "kids", 0, "kids", 0, "name",
	}

	for i := 0; i < b.N; i++ {
		var ok bool
		val := NewValue(codeJSON)
		for _, key := range keys {
			switch key := key.(type) {
			case string:
				val, ok = val.Field(key)
				if !ok {
					b.Errorf("error accessing field %v", key)
				}
			case int:
				val, ok = val.Index(key)
				if !ok {
					b.Errorf("error accessing index %v", key)
				}
			}
		}
		value := val.Actual()
		if value.(string) != "ssh" {
			b.Errorf("expected value ssh, got %v", value)
		}
	}
}

/* This benchmark contains a mix of Value creation of various data
types, Actual() dereferencing, and SetIndex() and SetField(). */
func BenchmarkProcessing(b *testing.B) {
	vals := make([]Value, 1<<16)

	for i := 0; i < b.N; i++ {
		for j := 0; j < len(vals); j++ {
			val := make(map[string]interface{})

			val["str"] = NewValue("string")
			val["boolv"] = NewValue(true)
			val["num"] = NewValue(1.0)
			val["null"] = NewValue(nil)
			arr := NewValue([]interface{}{"string", true, 1.0, nil})
			arr.SetIndex(2, 1.0)
			val["arr"] = arr
			m := NewValue(map[string]interface{}{"string": "string", "bool": true, "num": 1.0, "null": nil})
			m.SetField("string", "string")
			val["map"] = m

			vals[j] = NewValue(val)

			for k, v := range val {
				actual := NewValue(v).Actual()
				if actual == nil {
					val[k] = v
				}
			}
		}
	}
}

func BenchmarkLargeMap(b *testing.B) {
	keys := []string{
		"/tree/kids/0/kids/0/kids/0/kids/0/kids/0/name",
	}
	b.SetBytes(int64(len(codeJSON)))

	for i := 0; i < b.N; i++ {
		m := map[string]interface{}{}
		err := json.Unmarshal(codeJSON, &m)
		if err != nil {
			b.Fatalf("Error parsing JSON: %v", err)
		}
		value := json.Get(m, keys[0])
		if value.(string) != "ssh" {
			b.Errorf("expected value ssh, got %v", value)
		}
	}
}

func TestCopyValueFromBytes(t *testing.T) {

	val := NewValue([]byte(`{"name":"marty","type":"contact","address":{"street":"sutton oaks"}}`))

	val2 := val.Copy()
	val2.SetField("name", "bob")

	name, ok := val.Field("name")
	if !ok {
		t.Errorf("unexpected error accessing val.name")
	}
	name2, ok := val2.Field("name")
	if !ok {
		t.Errorf("unexpected error accessing val2.name")
	}
	if reflect.DeepEqual(name, name2) {
		t.Errorf("expected different names")
	}

	typ, ok := val.Field("type")
	if !ok {
		t.Errorf("unexpected error accessing val.type")
	}
	typ2, ok := val2.Field("type")
	if !ok {
		t.Errorf("unexpected error accessing val2.type")
	}
	if !reflect.DeepEqual(typ, typ2) {
		t.Errorf("expected same types")
	}
}

func TestCopyValueFromValue(t *testing.T) {

	val := NewValue(map[string]interface{}{
		"name": "marty",
		"type": "contact",
		"address": map[string]interface{}{
			"street": "sutton oaks",
		},
	})

	val2 := val.Copy()
	val2.SetField("name", "bob")

	name, ok := val.Field("name")
	if !ok {
		t.Errorf("unexpected error accessing val.name")
	}
	name2, ok := val2.Field("name")
	if !ok {
		t.Errorf("unexpected error accessing val2.name")
	}
	if reflect.DeepEqual(name, name2) {
		t.Errorf("expected different names")
	}

	typ, ok := val.Field("type")
	if !ok {
		t.Errorf("unexpected error accessing val.type")
	}
	typ2, ok := val2.Field("type")
	if !ok {
		t.Errorf("unexpected error accessing val2.type")
	}
	if !reflect.DeepEqual(typ, typ2) {
		t.Errorf("expected same types")
	}
}

func TestArraySetIndexLongerThanExistingArray(t *testing.T) {
	val := NewValue([]byte(`[]`))
	val = val.CopyForUpdate()
	val.SetIndex(0, "gerald")

	valval := val.Actual()
	if !reflect.DeepEqual(valval, []interface{}{"gerald"}) {
		t.Errorf("Expected [gerald] got %v", valval)
	}
}

func TestActual(t *testing.T) {
	val := NewValue(10.5)
	f, ok := val.Actual().(float64)
	if !ok {
		t.Errorf("Expected float64, got %v of type %T", f, f)
	}

	val = NewValue(-10.5)
	f, ok = val.Actual().(float64)
	if !ok {
		t.Errorf("Expected float64, got %v of type %T", f, f)
	}

	val = NewValue(10)
	i, ok := val.Actual().(float64)
	if !ok {
		t.Errorf("Expected float64, got %v of type %T", i, i)
	}

	val = NewValue(-10)
	i, ok = val.Actual().(float64)
	if !ok {
		t.Errorf("Expected float64, got %v of type %T", i, i)
	}
}

func TestActualForIndex(t *testing.T) {
	val := NewValue(10.5)
	f, ok := val.ActualForIndex().(float64)
	if !ok {
		t.Errorf("Expected float64, got %v of type %T", f, f)
	}

	val = NewValue(-10.5)
	f, ok = val.ActualForIndex().(float64)
	if !ok {
		t.Errorf("Expected float64, got %v of type %T", f, f)
	}

	val = NewValue(10)
	i, ok := val.ActualForIndex().(int64)
	if !ok {
		t.Errorf("Expected int64, got %v of type %T", i, i)
	}

	val = NewValue(-10)
	i, ok = val.ActualForIndex().(int64)
	if !ok {
		t.Errorf("Expected int64, got %v of type %T", i, i)
	}
}
