//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"bytes"
	"errors"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/util"
)

// reference encodings: values marshalled one at a time, objects compared over the union of their names
func refMarshal(v any) []byte {
	switch t := NewValue(v).(type) {
	case objectValue:
		if t == nil {
			return []byte("null")
		}
		names := make([]string, 0, len(t))
		for n := range t {
			names = append(names, n)
		}
		sort.Strings(names)
		buf := &bytes.Buffer{}
		buf.WriteByte('{')
		first := true
		for _, n := range names {
			fv := NewValue(t[n])
			if fv.Type() == MISSING {
				continue
			}
			if _, ok := fv.(*annotatedValueSelfReference); ok {
				continue
			}
			b := refMarshal(t[n])
			if !first {
				buf.WriteByte(',')
			}
			first = false
			json.MarshalStringToBuffer(n, buf)
			buf.WriteByte(':')
			buf.Write(b)
		}
		buf.WriteByte('}')
		return buf.Bytes()
	case sliceValue:
		if t == nil {
			return []byte("null")
		}
		buf := &bytes.Buffer{}
		buf.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.Write(refMarshal(e))
		}
		buf.WriteByte(']')
		return buf.Bytes()
	case intValue:
		return []byte(strconv.FormatInt(int64(t), 10))
	case floatValue:
		f := float64(t)
		switch {
		case math.IsNaN(f):
			return []byte(`"NaN"`)
		case math.IsInf(f, 1):
			return []byte(`"+Infinity"`)
		case math.IsInf(f, -1):
			return []byte(`"-Infinity"`)
		}
		return []byte(strconv.FormatFloat(f, 'f', -1, 64))
	case *annotatedValue:
		// an annotated value marshals as the value it wraps
		return refMarshal(t.Value)
	default:
		b, _ := t.MarshalJSON()
		return b
	}
}

func refCompare(obj1, obj2 map[string]any) Value {
	if d := len(obj1) - len(obj2); d != 0 {
		return intValue(d)
	}
	all := map[string]bool{}
	for k := range obj1 {
		all[k] = true
	}
	for k := range obj2 {
		all[k] = true
	}
	names := make([]string, 0, len(all))
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, n := range names {
		v1, ok := obj1[n]
		if !ok {
			return ONE_VALUE
		}
		v2, ok := obj2[n]
		if !ok {
			return NEG_ONE_VALUE
		}
		if c := NewValue(v1).Compare(NewValue(v2)); !c.Equals(ZERO_VALUE).Truth() {
			return c
		}
	}
	return ZERO_VALUE
}

func refCollate(obj1, obj2 map[string]any) int {
	if d := len(obj1) - len(obj2); d != 0 {
		return d
	}
	all := map[string]bool{}
	for k := range obj1 {
		all[k] = true
	}
	for k := range obj2 {
		all[k] = true
	}
	names := make([]string, 0, len(all))
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, n := range names {
		v1, ok := obj1[n]
		if !ok {
			return 1
		}
		v2, ok := obj2[n]
		if !ok {
			return -1
		}
		if c := NewValue(v1).Collate(NewValue(v2)); c != 0 {
			return c
		}
	}
	return 0
}

var strs = []string{"", "a", "b", "<&>", "q\"uote", "back\\slash", "tab\tnl\n", "\x01ctl", "héllo", "日本", " ", "z"}

/*
Return a random value for TestMarshalJSONEquivalent. That test only covers the
types generated here, so keep the cases in step with the type switch in
appendMarshal(): when a case is added there, add one here too.
*/
func randVal(r *rand.Rand, d int) any {
	k := r.Intn(22)
	if d > 3 && k >= 18 {
		k = r.Intn(18)
	}
	switch k {
	case 0:
		return nil
	case 1:
		return strs[r.Intn(len(strs))]
	case 2:
		return stringValue(strs[r.Intn(len(strs))])
	case 3:
		return r.Int63n(2000) - 1000
	case 4:
		return int(r.Intn(100000))
	case 5:
		return float64(r.Intn(100))
	case 6:
		return r.NormFloat64() * 1e6
	case 7:
		return []any{math.NaN(), math.Inf(1), math.Inf(-1), math.Copysign(0, -1), 1e21, 1e-7, float64(1 << 62), -0x1p63, 0x1p63}[r.Intn(9)]
	case 8:
		return float32(r.Float32())
	case 9:
		return r.Intn(2) == 0
	case 10:
		return boolValue(r.Intn(2) == 0)
	case 11:
		return MISSING_VALUE
	case 12:
		return NULL_VALUE
	case 13:
		return intValue(r.Int63())
	case 14:
		return floatValue(r.Float64())
	case 15:
		return []byte(`{"p":[1,"x",null]}`)
	case 16:
		return map[string]any(nil)
	case 17:
		return []any(nil)
	case 18:
		return NewAnnotatedValue(randObj(r, d+1))
	case 19:
		return objectValue(randObj(r, d+1))
	case 20:
		n := r.Intn(5)
		a := make([]any, n)
		for i := range a {
			a[i] = randVal(r, d+1)
		}
		if r.Intn(2) == 0 {
			return sliceValue(a)
		}
		return a
	default:
		return randObj(r, d+1)
	}
}

func randObj(r *rand.Rand, d int) map[string]any {
	n := r.Intn(20)
	m := make(map[string]any, n)
	for i := 0; i < n; i++ {
		m[strs[r.Intn(len(strs))]+strconv.Itoa(r.Intn(30))] = randVal(r, d)
	}
	return m
}

func TestMarshalJSONEquivalent(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		v := randVal(r, 0)
		want := string(refMarshal(v))
		got, err := NewValue(v).MarshalJSON()
		if err != nil || string(got) != want {
			t.Fatalf("iter %d: got %s (%v) want %s", i, got, err, want)
		}
	}
}

func TestObjectCollateEquivalent(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for i := 0; i < 20000; i++ {
		o1 := randObj(r, 2)
		var o2 map[string]any
		switch r.Intn(3) {
		case 0:
			o2 = randObj(r, 2)
		case 1: // same names, maybe different values
			o2 = make(map[string]any, len(o1))
			for k, v := range o1 {
				if r.Intn(4) == 0 {
					v = randVal(r, 3)
				}
				o2[k] = v
			}
		default: // same size, one name swapped
			o2 = make(map[string]any, len(o1))
			for k, v := range o1 {
				o2[k] = v
			}
			for k := range o2 {
				delete(o2, k)
				o2[strs[r.Intn(len(strs))]+"x"] = int64(1)
				break
			}
		}
		if got, want := objectCollate(o1, o2), refCollate(o1, o2); got != want {
			t.Fatalf("collate iter %d: got %d want %d", i, got, want)
		}
		if got, want := objectCompare(o1, o2), refCompare(o1, o2); got.Type() != want.Type() || got.String() != want.String() {
			t.Fatalf("compare iter %d: got %v want %v", i, got, want)
		}
	}
}

func TestObjectWalk(t *testing.T) {
	large1 := make(map[string]any, 3*_NAME_CAP)
	large2 := make(map[string]any, 3*_NAME_CAP)
	for i := 0; i < 3*_NAME_CAP; i++ {
		large1["a"+strconv.Itoa(i)] = i
		large2["a"+strconv.Itoa(i+1)] = i
	}
	equal := func(val1, val2 any) bool { return true }
	for _, c := range []struct {
		obj1, obj2 map[string]any
		want       int
	}{
		{nil, nil, 0},
		{map[string]any{}, nil, 0},
		// fewer names sorts first, whatever the names
		{map[string]any{"b": 1}, map[string]any{"a": 1, "c": 2}, -1},
		{map[string]any{"a": 1, "c": 2}, map[string]any{"b": 1}, 1},
		// same length: the first name only in one object decides
		{map[string]any{"a": 1, "b": 2}, map[string]any{"a": 1, "c": 2}, -1},
		{map[string]any{"a": 1, "c": 2}, map[string]any{"a": 1, "b": 2}, 1},
		{large1, large2, -1}, // "a0" is only in large1
		{large2, large1, 1},
		{large1, large1, 0},
	} {
		if got := objectWalk(c.obj1, c.obj2, equal); got != c.want {
			t.Errorf("objectWalk(%d names, %d names) = %d, want %d", len(c.obj1), len(c.obj2), got, c.want)
		}
	}

	// fewer names sorts first, whatever the names
	small := map[string]any{"b": 1, "c": 2}
	for _, c := range []struct {
		obj1, obj2 map[string]any
		want       int
	}{
		{small, large1, -1},
		{large1, small, 1},
		{nil, small, -1},
		{small, nil, 1},
	} {
		if got := objectCollate(c.obj1, c.obj2); (got > 0) != (c.want > 0) || (got < 0) != (c.want < 0) {
			t.Errorf("objectCollate(%d names, %d names) = %d, want sign %d", len(c.obj1), len(c.obj2), got, c.want)
		}
		if got := int64(objectCompare(c.obj1, c.obj2).(intValue)); (got > 0) != (c.want > 0) || (got < 0) != (c.want < 0) {
			t.Errorf("objectCompare(%d names, %d names) = %d, want sign %d", len(c.obj1), len(c.obj2), got, c.want)
		}
	}
}

// a value whose MarshalJSON fails
type failValue struct {
	Value
}

func (failValue) MarshalJSON() ([]byte, error) {
	return nil, errors.New("fail")
}

func TestMarshalToBufferError(t *testing.T) {
	for _, val := range []any{
		failValue{NULL_VALUE},
		map[string]any{"a": 1, "b": []any{1, 2, map[string]any{"c": failValue{NULL_VALUE}}}},
		[]any{"x", map[string]any{"a": failValue{NULL_VALUE}}},
	} {
		buf := bytes.NewBufferString("prefix")
		if _, err := marshalToBuffer(val, buf, false); err == nil {
			t.Errorf("%v: no error", val)
		}
		if buf.String() != "prefix" {
			t.Errorf("%v: buffer left as %q", val, buf.String())
		}
	}
}

// unwrapValue() skips the call for these types: their unwrap() must return the value itself
func TestUnwrapValueIdentity(t *testing.T) {
	for _, v := range []Value{intValue(1), floatValue(1.5), stringValue("s"), sliceValue{1}, binaryValue{1}, MISSING_VALUE} {
		if !reflect.DeepEqual(v.unwrap(), v) {
			t.Errorf("%T.unwrap() is not the value itself", v)
		}
	}
	if _, ok := unwrapValue(NewAnnotatedValue(intValue(1))).(intValue); !ok {
		t.Errorf("unwrapValue() of an annotated value did not unwrap it")
	}
}

func benchRow(n int) objectValue {
	m := make(map[string]any, n)
	for i := 0; i < n; i++ {
		switch i % 3 {
		case 0:
			m["field"+strconv.Itoa(i)] = "value-" + strconv.Itoa(i)
		case 1:
			m["field"+strconv.Itoa(i)] = int64(i * 1000)
		default:
			m["field"+strconv.Itoa(i)] = float64(i) + 0.25
		}
	}
	return objectValue(m)
}

func benchParallel(b *testing.B, f func(buf *bytes.Buffer)) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		buf := &bytes.Buffer{}
		for pb.Next() {
			buf.Reset()
			f(buf)
		}
	})
}

var benchSink any

func BenchmarkObjectString(b *testing.B) {
	for _, n := range []int{5, 40} {
		row := benchRow(n)
		b.Run("fields="+strconv.Itoa(n), func(b *testing.B) {
			benchParallel(b, func(*bytes.Buffer) { _ = row.String() })
		})
	}
}

func BenchmarkObjectWriteJSON(b *testing.B) {
	for _, n := range []int{5, 40} {
		row := benchRow(n)
		b.Run("fields="+strconv.Itoa(n), func(b *testing.B) {
			benchParallel(b, func(buf *bytes.Buffer) { row.WriteJSON(nil, buf, "", "", false) })
		})
	}
}

func BenchmarkArrayString(b *testing.B) {
	arr := sliceValue([]any{int64(1), 2.5, "x", map[string]any{"a": int64(1), "b": "y"}, []any{true, nil}})
	benchParallel(b, func(*bytes.Buffer) { _ = arr.String() })
}

func BenchmarkObjectCollate(b *testing.B) {
	for _, n := range []int{5, 40} {
		row1, row2 := benchRow(n), benchRow(n)
		row2["field"+strconv.Itoa(n-1)] = "last"
		b.Run("fields="+strconv.Itoa(n), func(b *testing.B) {
			benchParallel(b, func(*bytes.Buffer) { benchSink = objectCollate(row1, row2) })
		})
	}
}

func BenchmarkObjectCompare(b *testing.B) {
	row1, row2 := benchRow(5), benchRow(5)
	row2["field4"] = "last"
	benchParallel(b, func(*bytes.Buffer) { benchSink = objectCompare(row1, row2) })
}

func BenchmarkNumberWriteJSON(b *testing.B) {
	b.Run("int", func(b *testing.B) {
		v := intValue(1234567)
		benchParallel(b, func(buf *bytes.Buffer) { v.WriteJSON(nil, buf, "", "", false) })
	})
	b.Run("float", func(b *testing.B) {
		v := floatValue(98.25)
		benchParallel(b, func(buf *bytes.Buffer) { v.WriteJSON(nil, buf, "", "", false) })
	})
}

func BenchmarkScalarCompare(b *testing.B) {
	s1, s2 := NewValue("alice smith"), NewValue("alice smyth")
	i1, f1 := NewValue(int64(4242)), NewValue(42.0001)
	benchParallel(b, func(*bytes.Buffer) {
		n := s1.Collate(s2) + i1.Collate(f1)
		if s1.Equals(s2).Truth() || i1.Equals(f1).Truth() {
			n++
		}
		benchSink = n > 0
	})
}

func BenchmarkSetHasScalar(b *testing.B) {
	vals := []Value{NewValue("alice"), NewValue(int64(12345)), NewValue(3.5), NewValue(true), NULL_VALUE, NewValue("bob")}
	set := NewSet(8, false, false)
	for _, v := range vals[:3] {
		set.Put(v, v)
	}
	benchParallel(b, func(*bytes.Buffer) {
		n := 0
		for _, v := range vals {
			if set.Has(v) {
				n++
			}
		}
		benchSink = n
	})
}

func BenchmarkBinaryWriteJSON(b *testing.B) {
	v := NewBinaryValue([]byte{1, 2, 3})
	benchParallel(b, func(buf *bytes.Buffer) { v.WriteJSON(nil, buf, "", "", false) })
}

func TestNumberWriteJSONWriter(t *testing.T) {
	for _, c := range []struct {
		v    Value
		want string
	}{
		{intValue(-42), "-42"},
		{floatValue(1.5), "1.5"},
		{floatValue(math.Inf(1)), "\"+Infinity\""},
	} {
		// a writer other than *bytes.Buffer must not panic
		var sb strings.Builder
		if err := c.v.WriteJSON(nil, &sb, "", "", false); err != nil || sb.String() != c.want {
			t.Errorf("WriteJSON(%v) = %q, %v, want %q", c.v, sb.String(), err, c.want)
		}
	}
}

func TestNamesPoolCleared(t *testing.T) {
	obj := make(map[string]any, 2*_NAME_CAP)
	for i := 0; i < 2*_NAME_CAP; i++ {
		obj["f"+strconv.Itoa(i)] = i
	}
	var nameBuf [_NAME_CAP]string
	names, pooled := sortedNamesIn(obj, &nameBuf)
	if len(pooled) != len(obj) {
		t.Fatalf("pooled slice has length %d, want %d", len(pooled), len(obj))
	}
	if &names[0] != &pooled[0] {
		t.Fatalf("names do not use the pooled slice")
	}
	// Put into a private pool, as the global one may hand the slice out again
	util.NewStringPool(cap(pooled)).Put(pooled)
	for i, n := range pooled[:cap(pooled)] {
		if n != "" {
			t.Fatalf("pooled name %d not cleared: %q", i, n)
		}
	}
}

func TestNonFiniteMarshalNoAlloc(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under the race detector")
	}
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		v := floatValue(f)
		if n := testing.AllocsPerRun(100, func() { v.MarshalJSON() }); n != 0 {
			t.Errorf("floatValue(%v).MarshalJSON() allocates %v times", f, n)
		}
	}
}

func TestMarshalSelfReference(t *testing.T) {
	av := NewAnnotatedValue(map[string]any{"a": 1})
	if err := av.SetField("self", av); err != nil {
		t.Fatal(err)
	}
	// as an object field the self reference is dropped
	if s := string(marshalString(av)); s != `{"a":1}` {
		t.Errorf("annotated object with a self reference = %s, want {\"a\":1}", s)
	}
	obj := map[string]any{"b": 2, "o": av}
	if b, err := NewValue(obj).MarshalJSON(); err != nil || string(b) != `{"b":2,"o":{"a":1}}` {
		t.Errorf("nested self reference = %s, %v", b, err)
	}
	if b := refMarshal(obj); string(b) != `{"b":2,"o":{"a":1}}` {
		t.Errorf("reference encoding = %s", b)
	}

	// anywhere else it is written as null, so that the JSON stays valid
	self := (*annotatedValueSelfReference)(av.(*annotatedValue))
	if b, err := self.MarshalJSON(); err != nil || string(b) != "null" {
		t.Errorf("self reference MarshalJSON = %s, %v", b, err)
	}
	arr := []any{1, self, 2}
	if b, err := NewValue(arr).MarshalJSON(); err != nil || string(b) != `[1,null,2]` {
		t.Errorf("self reference in an array = %s, %v", b, err)
	}

	// XML writes it as null too
	xbuf := &bytes.Buffer{}
	if err := NewValue(arr).WriteXML(nil, xbuf, "", "", false); err != nil || !strings.Contains(xbuf.String(), "<null/>") {
		t.Errorf("self reference in an array, WriteXML = %s, %v", xbuf.String(), err)
	}

	// WriteJSON gives the same results, with or without indenting
	only := NewAnnotatedValue(map[string]any{})
	only.SetField("self", only)
	for _, c := range []struct {
		v    Value
		want string
	}{
		{av, `{"a":1}`},
		{NewValue(obj), `{"b":2,"o":{"a":1}}`},
		{NewValue(arr), `[1,null,2]`},
		{only, `{}`},
	} {
		for _, fast := range []bool{false, true} {
			buf := &bytes.Buffer{}
			if err := c.v.WriteXML(nil, buf, "", "", fast); err != nil || strings.Contains(buf.String(), `name="self"`) {
				t.Errorf("WriteXML(fast=%v) = %s, %v", fast, buf.String(), err)
			}
			buf.Reset()
			if err := c.v.WriteJSON(nil, buf, "", "", fast); err != nil || buf.String() != c.want {
				t.Errorf("WriteJSON(fast=%v) = %s, %v, want %s", fast, buf.String(), err, c.want)
			}
			buf.Reset()
			var parsed any
			if err := c.v.WriteJSON(nil, buf, "  ", "  ", fast); err != nil {
				t.Errorf("indented WriteJSON = %v", err)
			} else if err = json.Unmarshal(buf.Bytes(), &parsed); err != nil {
				t.Errorf("indented WriteJSON(fast=%v) = %s: %v", fast, buf.String(), err)
			} else if b, _ := NewValue(parsed).MarshalJSON(); string(b) != c.want {
				t.Errorf("indented WriteJSON(fast=%v) = %s, want %s", fast, buf.String(), c.want)
			}
		}
	}
}

func TestMarshalTrimmed(t *testing.T) {
	// small values leave most of the estimated buffer unused
	obj := make(map[string]any, 1000)
	for i := 0; i < 1000; i++ {
		obj["f"+strconv.Itoa(i)] = i % 10
	}
	b, err := NewValue(obj).MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if cap(b) > 2*len(b) {
		t.Errorf("MarshalJSON kept capacity %d for %d bytes", cap(b), len(b))
	}
	if !bytes.Equal(b, refMarshal(obj)) {
		t.Errorf("trimmed MarshalJSON differs from the reference encoding")
	}
}

func TestNumberWriteJSONFullBuffer(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under the race detector")
	}
	for _, v := range []Value{intValue(math.MinInt64), floatValue(-12345.678901234567)} {
		want, _ := v.MarshalJSON()
		n := testing.AllocsPerRun(100, func() {
			// leave too little spare capacity for the number
			buf := bytes.NewBuffer(make([]byte, 60, 64))
			v.WriteJSON(nil, buf, "", "", false)
			if !bytes.Equal(buf.Bytes()[60:], want) {
				t.Fatalf("WriteJSON(%v) = %s, want %s", v, buf.Bytes()[60:], want)
			}
		})
		// the buffer, its backing array and its growth, with none for the
		// number itself
		if n > 3 {
			t.Errorf("WriteJSON(%v) into a full buffer allocates %v times", v, n)
		}
	}
}

func TestSmallObjectNoAlloc(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under the race detector")
	}
	// values that box without allocating, so that only the names are tested
	obj1 := NewValue(map[string]any{"a": 1, "b": 2, "c": true, "d": nil})
	obj2 := NewValue(map[string]any{"a": 1, "b": 2, "c": true, "d": 7})
	if n := testing.AllocsPerRun(100, func() { obj1.Collate(obj2) }); n != 0 {
		t.Errorf("Collate allocates %v times", n)
	}
	if n := testing.AllocsPerRun(100, func() { obj1.Compare(obj2) }); n != 0 {
		t.Errorf("Compare allocates %v times", n)
	}
	buf := bytes.NewBuffer(make([]byte, 0, 256))
	if n := testing.AllocsPerRun(100, func() { buf.Reset(); obj1.WriteJSON(nil, buf, "", "", true) }); n != 0 {
		t.Errorf("WriteJSON allocates %v times", n)
	}
}

func TestUnwrapNoAlloc(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under the race detector")
	}
	f := floatValue(2.5)
	av := NewAnnotatedValue(f)
	sv := NewScopeValue(map[string]any{}, nil)
	sv.Value = f
	for _, w := range []Value{av, sv} {
		if u := unwrapValue(w); u != Value(f) {
			t.Errorf("unwrapValue(%T) = %v, want %v", w, u, f)
		}
		if n := testing.AllocsPerRun(100, func() { f.Equals(w) }); n != 0 {
			t.Errorf("Equals(%T) allocates %v times", w, n)
		}
	}
	for _, c := range []struct {
		v    Value
		want string
	}{
		{NewValue(map[string]any{}), `{}`},
		{NewValue(map[string]any{"a": map[string]any{}, "b": []any{map[string]any{}}}), `{"a":{},"b":[{}]}`},
		{NewAnnotatedValue(map[string]any{}), `{}`},
		{NewValue(map[string]any{"a": NewAnnotatedValue(map[string]any{})}), `{"a":{}}`},
	} {
		if b, err := c.v.MarshalJSON(); err != nil || string(b) != c.want {
			t.Errorf("MarshalJSON = %s, %v, want %s", b, err, c.want)
		}
	}
	if n := testing.AllocsPerRun(100, func() { objectValue{}.MarshalJSON() }); n != 0 {
		t.Errorf("empty object MarshalJSON allocates %v times", n)
	}
}

func TestSharedBytesCapped(t *testing.T) {
	for _, b := range [][]byte{_FALSE_BYTES, _TRUE_BYTES, _NAN_BYTES, _POS_INF_BYTES, _NEG_INF_BYTES,
		_NULL_BYTES, _NULL_XML, _EMPTY_OBJECT_BYTES} {
		if cap(b) != len(b) {
			t.Errorf("shared bytes %q have cap %d, want %d", b, cap(b), len(b))
		}
	}
	// appending to a returned slice must not write into the shared array
	b, _ := objectValue{}.MarshalJSON()
	_ = append(b, 'x')
	if string(_EMPTY_OBJECT_BYTES) != "{}" || cap(_EMPTY_OBJECT_BYTES) != 2 {
		t.Errorf("_EMPTY_OBJECT_BYTES modified: %q", _EMPTY_OBJECT_BYTES)
	}
}
