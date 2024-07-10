//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	// "time"

	json "github.com/couchbase/go_json"
	diffpkg "github.com/kylelemons/godebug/diff"
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

	mv := val.GetMetaField(META_ID)
	if mv == nil {
		t.Errorf("metadata missing")
	} else {
		id := mv.(string)
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

/*
	This benchmark contains a mix of Value creation of various data

types, Actual() dereferencing, and SetIndex() and SetField().
*/
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

func TestValueSpilling(t *testing.T) {
	var f *os.File
	var err error

	defer func() {
		r := recover()
		if r != nil {
			if f != nil {
				pos, _ := f.Seek(0, os.SEEK_CUR)
				fmt.Printf("Panic: %v\nFile: %v\nPos: %v\n", r, f.Name(), pos)
			}
			panic(r)
		}
	}()

	type pair struct {
		name  string
		value interface{}
	}

	list := make([]*pair, 0, 32)

	list = append(list, &pair{"nil", nil})
	list = append(list, &pair{"bool", true})
	b := make([]byte, 1)
	b[0] = 0xaa
	list = append(list, &pair{"[]byte", b})
	list = append(list, &pair{"int", int(1)})
	list = append(list, &pair{"int32", int32(31)})
	list = append(list, &pair{"uint32", uint32(32)})
	list = append(list, &pair{"int64", int64(63)})
	list = append(list, &pair{"uint64", uint64(64)})
	list = append(list, &pair{"float32", float32(31.5)})
	list = append(list, &pair{"float64", float64(32.5)})
	list = append(list, &pair{"string", "correct"})

	m := make(map[string]interface{})
	m["test"] = "correct"
	list = append(list, &pair{"map[string]interface{}", m})
	mv := make(map[string]Value)
	mv["test"] = NewValue("correct")
	list = append(list, &pair{"map[string]Value", mv})
	a := make([]interface{}, 1)
	a[0] = "correct"
	list = append(list, &pair{"[]interface", a})
	a2 := make([]Value, 1)
	a2[0] = NewValue("correct")
	list = append(list, &pair{"[]Value", a2})
	a3 := make(Values, 1)
	a3[0] = NewValue("correct")
	list = append(list, &pair{"Values", a3})
	a4 := make([]interface{}, 2)
	a4[0] = NewValue("correct")
	a4[1] = NewMissingValue()
	list = append(list, &pair{"sliceValue", NewValue(a4)})
	list = append(list, &pair{"listValue", &listValue{NewValue(make([]interface{}, 1)).(sliceValue)}})
	list = append(list, &pair{"binaryValue", NewValue(make([]byte, 1))})
	list = append(list, &pair{"boolValue", NewValue(true)})
	list = append(list, &pair{"floatValue", NewValue(32.5)})
	list = append(list, &pair{"intValue", NewValue(64)})
	list = append(list, &pair{"missingValue", NewMissingValue()})
	list = append(list, &pair{"nullValue", NewNullValue()})
	list = append(list, &pair{"objectValue", NewValue(make(map[string]interface{}))})
	list = append(list, &pair{"stringValue", NewValue("correct")})
	m2 := make(map[string]Value)
	m2["test"] = NewValue("correct")
	list = append(list, &pair{"map[string]Value", NewValue(m2)})

	// annotatedValue
	pv := &parsedValue{raw: []byte(`{"street":"sutton oaks"}`), parsedType: OBJECT}
	list = append(list, &pair{"parsedValue", pv})

	av := NewAnnotatedValue([]byte(`{"name":"marty","address":{"street":"sutton oaks"}}`))
	av.SetId("doc1")
	av.SetField("selfref", av)
	av.Size()
	list = append(list, &pair{"AnnotatedValue", av})

	f, err = os.CreateTemp("", "value_test-*")
	if err != nil {
		t.Errorf("Failed to create spill file: %v", err)
		return
	}

	for _, p := range list {
		err = writeSpillValue(f, p.value, make([]byte, 0, 128))
		if err != nil {
			t.Errorf("Failed to write '%v' to spill file: %v", p.name, err)
			return
		}
	}

	f.Sync()

	_, err = f.Seek(0, os.SEEK_SET)
	if err != nil {
		t.Errorf("Failed to rewind spill file: %v", err)
		return
	}

	re := regexp.MustCompile("\\(0x[0-9a-f]+\\)")
	for _, p := range list {
		v, err := readSpillValue(nil, f, nil)
		if err != nil {
			t.Errorf("Failed to read '%v' from spill file: %v", p.name, err)
			return
		}
		if v != nil {
			if val, ok := v.(Value); ok {
				val.Size()
			}
		}
		bts := fmt.Sprintf("%T: %#v", p.value, p.value)
		bts = re.ReplaceAllString(bts, "(<address>)") // erase pointer address values
		rts := fmt.Sprintf("%T: %#v", v, v)
		rts = re.ReplaceAllString(rts, "(<address>)") // erase pointer address values
		if rts != bts {
			rts = strings.ReplaceAll(rts, ",", ",\n")
			bts = strings.ReplaceAll(bts, ",", ",\n")

			d := diffpkg.Diff(bts, rts)

			t.Errorf("'%v' has incorrect reconstructed value:\n%v", p.name, d)
			continue
		}
		if av, ok := v.(Value); ok {
			if !p.value.(Value).EquivalentTo(av) {
				t.Errorf("'%v' reconstructed value is not equivalent:\n%#v\nshould be\n%#v", p.name, av, p.value)
			}
			if p.value.(Value).Size() != av.Size() {
				t.Errorf("'%v' reconstructed annotated value size differs: %v should be %v",
					p.name, av.Size(), p.value.(Value).Size())
			}
		}
	}

	f.Close()
	os.Remove(f.Name()) // remove after so files can be examined if necessary
}

func TestSpillingArray(t *testing.T) {

	tracking := int64(0)
	spillThreshold := uint64(0)

	shouldSpill := func(c uint64, n uint64) bool {
		return c > spillThreshold
	}
	acquire := func(size int) AnnotatedValues { return make(AnnotatedValues, 0, size) }
	trackMem := func(sz int64) error {
		tracking -= sz
		return nil
	}
	lessThan := func(v1 AnnotatedValue, v2 AnnotatedValue) bool {
		m1 := v1.GetValue().Actual().(map[string]interface{})
		m2 := v2.GetValue().Actual().(map[string]interface{})
		n1 := m1["name"].(string) + m1["surname"].(string)
		n2 := m2["name"].(string) + m2["surname"].(string)
		return strings.Compare(n1, n2) < 0
	}
	array := NewAnnotatedArray(acquire, nil, shouldSpill, trackMem, lessThan, false)
	check := make([]string, 4)

	av := NewAnnotatedValue([]byte(`{"name":"Marty","surname":"McFly"}`))
	av.SetId("doc1")
	av.SetField("selfref", av)
	array.Append(av)
	check[3] = av.GetId().(string)
	spillThreshold += av.Size()

	av = NewAnnotatedValue([]byte(`{"name":"Emmett","surname":"Brown"}`))
	av.SetId("doc2")
	av.SetField("selfref", av)
	array.Append(av)
	check[0] = av.GetId().(string)
	spillThreshold += av.Size()

	av = NewAnnotatedValue([]byte(`{"name":"Loraine","surname":"Baines"}`))
	av.SetId("doc3")
	av.SetField("selfref", av)
	array.Append(av)
	check[2] = av.GetId().(string)

	av = NewAnnotatedValue([]byte(`{"name":"George","surname":"McFly"}`))
	av.SetId("doc4")
	av.SetField("selfref", av)
	array.Append(av)
	check[1] = av.GetId().(string)

	checkIndex := 0
	err := array.Foreach(func(av AnnotatedValue) bool {
		if check[checkIndex] != av.GetId().(string) {
			t.Errorf("documents not in order: expected '%v' at position %v found '%v'",
				check[checkIndex], checkIndex, av.GetId().(string))
			return false
		}
		checkIndex++
		return true
	})
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if tracking != 0 {
		t.Errorf("memory accounting error, found %v (should be 0)", tracking)
	}

}
