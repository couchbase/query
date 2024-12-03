//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"strings"
	"time"

	"github.com/couchbase/query/logging"
)

func NewField(i interface{}) (Field, error) {
	logging.Tracef("%d", i)
	var err error
	var ok bool
	defn, ok := i.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Field definition is not an object.")
	}
	name, ok := defn["name"].(string)
	if !ok {
		name = fmt.Sprintf("field_%d", nextSerial())
	}
	o, _ := defn["optional"].(bool)
	n, _ := defn["null"].(bool)
	t, ok := defn["type"]
	if !ok {
		return &AnyField{FieldBase{name: name, fixed: false, optional: o, nullable: n}}, nil
	}
	switch t := t.(type) {
	case map[string]interface{}: // array
		f := &ArrayField{FieldBase: FieldBase{typ: _FT_ARRAY, name: name, optional: o, nullable: n}}
		v, ok := defn["length"]
		if !ok {
			f.length = 1
		} else if n, ok := v.(float64); ok {
			f.length = int(n)
		}
		f.elemType, err = NewField(t)
		if err != nil {
			return nil, fmt.Errorf("Error in definition of \"%s\": %v.", name, err)
		}
		return f, nil
	case []interface{}: // object
		f := &ObjectField{FieldBase: FieldBase{typ: _FT_OBJECT, name: name, optional: o, nullable: n}}
		if len(t) > 0 {
			f.fields = make([]Field, 0, len(t))
			for i := range t {
				fld, err := NewField(t[i])
				if err != nil {
					return nil, fmt.Errorf("Error in definition of \"%s\": \"fields\"[%d]: %v", name, i, err)
				}
				f.fields = append(f.fields, fld)
			}
		}
		return f, nil
	case string:
		switch t {
		case "string":
			f := &StringField{FieldBase: FieldBase{typ: _FT_STRING, name: name, optional: o, nullable: n}}
			ok, s, err := getString(defn, "value")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			if ok {
				f.fixed = true
				f.random = false
				f.values = append(f.values, s)
				return f, nil
			}
			f.fixed = false
			_, f.random, err = getBool(defn, "random")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			_, f.min, err = getInt(defn, "min")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			_, f.max, err = getInt(defn, "max")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			if f.min > f.max {
				f.min, f.max = f.max, f.min
			}
			_, f.prefix, err = getString(defn, "prefix")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			_, f.suffix, err = getString(defn, "suffix")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			if v, ok := defn["values"]; ok {
				if l, ok := v.([]interface{}); !ok {
					return nil, fmt.Errorf("Error in definition of \"%s\": \"values\" is not an array.", name)
				} else {
					for i := range l {
						if s, ok := l[i].(string); !ok {
							return nil, fmt.Errorf("Error in definition of \"%s\": \"values\"[%d] is not a string.", name, i)
						} else {
							f.values = append(f.values, s)
						}
					}
				}
			}
			return f, nil
		case "int":
			f := &IntField{FieldBase: FieldBase{typ: _FT_INT, name: name, optional: o, nullable: n}}
			ok, val, err := getInt(defn, "value")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			} else if ok {
				f.values = append(f.values, val)
			}
			v, ok := defn["values"]
			if ok {
				if l, ok := v.([]interface{}); !ok {
					return nil, fmt.Errorf("Error in definition of \"%s\": \"values\" is not an array.", name)
				} else {
					for i := range l {
						if val, ok := l[i].(float64); !ok {
							return nil, fmt.Errorf("Error in definition of \"%s\": \"values\"[%d] is not numeric.", name, i)
						} else {
							f.values = append(f.values, int(val))
						}
					}
				}
			}
			_, f.random, err = getBool(defn, "random")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			_, f.min, err = getInt(defn, "min")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			ok, f.max, err = getInt(defn, "max")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			if !ok {
				f.max = math.MaxInt
			}
			if f.min > f.max {
				f.min, f.max = f.max, f.min
			}
			if f.min == f.max {
				f.fixed = true
				f.random = false
				if len(f.values) == 0 {
					f.values = append(f.values, f.min)
				}
			}
			ok, f.step, err = getInt(defn, "step")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			} else if ok {
				if f.step == 0 {
					f.fixed = true
					f.random = false
				}
			} else {
				f.step = 1
			}
			return f, nil
		case "float":
			f := &FloatField{FieldBase: FieldBase{typ: _FT_FLOAT, name: name, optional: o, nullable: n}}
			ok, val, err := getFloat(defn, "value")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			} else if ok {
				f.values = append(f.values, val)
			}
			v, ok := defn["values"]
			if ok {
				if l, ok := v.([]interface{}); !ok {
					return nil, fmt.Errorf("Error in definition of \"%s\": \"values\" is not an array.", name)
				} else {
					for i := range l {
						if val, ok := l[i].(float64); !ok {
							return nil, fmt.Errorf("Error in definition of \"%s\": \"values\"[%d] is not numeric.", name, i)
						} else {
							f.values = append(f.values, val)
						}
					}
				}
			}
			_, f.random, err = getBool(defn, "random")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			_, f.min, err = getFloat(defn, "min")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			ok, f.max, err = getFloat(defn, "max")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			if !ok {
				f.max = math.MaxFloat64
			}
			if f.min > f.max {
				f.min, f.max = f.max, f.min
			}
			if f.min == f.max {
				f.fixed = true
				f.random = false
				if len(f.values) == 0 {
					f.values = append(f.values, f.min)
				}
			}
			ok, f.step, err = getFloat(defn, "step")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			} else if ok {
				if f.step == 0 {
					f.fixed = true
					f.random = false
				}
			} else {
				f.step = 1
			}
			return f, nil
		case "boolean":
			f := &BooleanField{FieldBase: FieldBase{typ: _FT_BOOLEAN, name: name, optional: o, nullable: n}}
			ok, val, err := getBool(defn, "value")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			} else if ok {
				f.values = append(f.values, val)
			}
			v, ok := defn["values"]
			if ok {
				if l, ok := v.([]interface{}); !ok {
					return nil, fmt.Errorf("Error in definition of \"%s\": \"values\" is not an array.", name)
				} else {
					for i := range l {
						if val, ok := l[i].(bool); !ok {
							return nil, fmt.Errorf("Error in definition of \"%s\": \"values\"[%d] is not boolean.", name, i)
						} else {
							f.values = append(f.values, val)
						}
					}
				}
			}
			_, f.random, err = getBool(defn, "random")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			return f, nil
		case "null":
			f := &NullField{FieldBase: FieldBase{typ: _FT_NULL, name: name, optional: o, nullable: true}}
			return f, nil
		case "date":
			f := &DateField{FieldBase: FieldBase{typ: _FT_DATE, name: name, optional: o, nullable: n}}
			ok, val, err := getTime(defn, "value")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			} else if ok {
				f.values = append(f.values, val)
			}
			v, ok := defn["values"]
			if ok {
				if l, ok := v.([]interface{}); !ok {
					return nil, fmt.Errorf("Error in definition of \"%s\": \"values\" is not an array.", name)
				} else {
					for i := range l {
						if val, ok := l[i].(string); !ok {
							return nil, fmt.Errorf("Error in definition of \"%s\": \"values\"[%d] is not a string.", name, i)
						} else {
							t, err := time.Parse("2006-01-02T15:04:05.999Z07:00", val)
							if err != nil {
								return nil, fmt.Errorf("Error in definition of \"%s\": \"values\"[%d]: %v", name, i, err)
							}
							f.values = append(f.values, t)
						}
					}
				}
			}
			_, f.random, err = getBool(defn, "random")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			_, f.min, err = getTime(defn, "min")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			ok, f.max, err = getTime(defn, "max")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			}
			if !ok {
				f.max = time.Date(9999, 11, 31, 23, 59, 59, 999999999, time.UTC)
			}
			if f.min.After(f.max) {
				f.min, f.max = f.max, f.min
			}
			if f.min.Compare(f.max) == 0 {
				f.fixed = true
				f.random = false
				if len(f.values) == 0 {
					f.values = append(f.values, f.min)
				}
			}
			ok, f.step, err = getInt(defn, "step")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			} else if ok {
				if f.step == 0 {
					f.fixed = true
					f.random = false
				}
			} else {
				f.step = 1
			}
			ok, f.unit, err = getString(defn, "unit")
			if err != nil {
				return nil, fmt.Errorf("Error in definition of \"%s\": %v", name, err)
			} else if !ok {
				f.unit = "millisecond"
			}
			return f, nil
		default:
			return nil, fmt.Errorf("Field \"%s\" lacks a valid type: %s", name, t)
		}
	default:
		return nil, fmt.Errorf("Field \"%s\" lacks a valid type: %T", name, t)
	}

	return nil, fmt.Errorf("Internal error")
}

func getInt(m map[string]interface{}, f string) (bool, int, error) {
	i, ok := m[f]
	if !ok {
		return false, 0, nil
	}
	if n, ok := i.(float64); !ok {
		return false, 0, fmt.Errorf("\"%s\" is not numeric.", f)
	} else {
		return true, int(n), nil
	}
}

func getFloat(m map[string]interface{}, f string) (bool, float64, error) {
	i, ok := m[f]
	if !ok {
		return false, 0, nil
	}
	if n, ok := i.(float64); !ok {
		return false, 0, fmt.Errorf("\"%s\" is not numeric.", f)
	} else {
		return true, n, nil
	}
}

func getString(m map[string]interface{}, f string) (bool, string, error) {
	i, ok := m[f]
	if !ok {
		return false, "", nil
	}
	if s, ok := i.(string); !ok {
		return false, "", fmt.Errorf("\"%s\" is not a string.", f)
	} else {
		return true, s, nil
	}
}

func getBool(m map[string]interface{}, f string) (bool, bool, error) {
	i, ok := m[f]
	if !ok {
		return false, false, nil
	}
	if b, ok := i.(bool); !ok {
		return false, false, fmt.Errorf("\"%s\" is not a boolean.", f)
	} else {
		return true, b, nil
	}
}

func getTime(m map[string]interface{}, f string) (bool, time.Time, error) {
	i, ok := m[f]
	if !ok {
		return false, time.Time{}, nil
	}
	if s, ok := i.(string); !ok {
		return false, time.Time{}, fmt.Errorf("\"%s\" is not a string.", f)
	} else {
		t, err := time.Parse("2006-01-02T15:04:05.999Z07:00", s)
		if err != nil {
			return false, time.Time{}, fmt.Errorf("\"%s\" is not a valid timestamp: %v", f, err)
		}
		return true, t, nil
	}
}

func NewRandomField(name string) Field {
	base := &FieldBase{}
	base.name = name
	base.random = rand.Intn(10)%2 == 0
	base.nullable = rand.Intn(10) == 7
	base.optional = rand.Intn(20) == 7
	base.fixed = rand.Intn(20) == 7
	return newRandomFieldWithBase(base, true, true)
}

func NewRandomJoinField(name string) Field {
	base := &FieldBase{}
	base.name = name
	base.random = rand.Intn(10)%2 == 0
	base.nullable = rand.Intn(10) == 7
	base.optional = rand.Intn(20) == 7
	base.fixed = rand.Intn(20) == 7
	return newRandomFieldWithBase(base, false, false)
}

func newRandomFieldWithBase(base *FieldBase, allowAny bool, allowComposite bool) Field {
	t := rand.Intn(_FT_SIZER)
	if t == _FT_ANY && !allowAny {
		t = _FT_STRING
	}
	if (t == _FT_ARRAY || t == _FT_OBJECT) && !allowComposite {
		t = _FT_INT
	}
	if t == _FT_ANY || t == _FT_OBJECT || t == _FT_ARRAY {
		// check nesting depth using the runtime stack (simpler than carrying a count)
		// if more than _MAX_RANDOM_FIELD_DEPTH random fields deep, then generate a string always
		pc := make([]uintptr, 256)
		n := runtime.Callers(1, pc)
		if n > 0 {
			pc = pc[:n]
			frames := runtime.CallersFrames(pc)
			count := 0
			for {
				frame, more := frames.Next()
				if frame.Function == "main.newRandomFieldWithBase" {
					count++
					if count == _MAX_RANDOM_FIELD_DEPTH {
						t = _FT_STRING
						break
					}
				}
				if !more {
					break
				}
			}
		}
	}

	switch t {
	case _FT_ANY:
		f := &AnyField{FieldBase: *base}
		f.typ = t
		return f
	case _FT_STRING:
		f := &StringField{FieldBase: *base}
		f.typ = t
		if f.fixed {
			f.values = append(f.values, getRandomString(50))
			return f
		}
		if rand.Intn(20) == 13 {
			f.prefix = getRandomString(10)
		}
		if rand.Intn(20) == 13 {
			f.suffix = getRandomString(10)
		}
		if rand.Intn(2) == 0 {
			for n := rand.Intn(10) + 1; n > 0; n-- {
				f.values = append(f.values, getRandomString(50))
			}
			if rand.Intn(5) == 4 {
				f.max = rand.Intn(len(f.values))
			}
		} else {
			f.min = rand.Intn(10)
			f.max = f.min + rand.Intn(40) + 1
		}
		return f
	case _FT_INT:
		f := &IntField{FieldBase: *base}
		f.typ = t
		if f.fixed {
			f.values = append(f.values, rand.Int())
			return f
		}
		if rand.Intn(2) == 0 {
			for n := rand.Intn(10) + 1; n > 0; n-- {
				f.values = append(f.values, rand.Int())
			}
		} else {
			f.min = rand.Intn(1000)
			f.max = rand.Intn(10000) + f.min + 1
			f.step = rand.Intn(10) + 1
		}
		return f
	case _FT_FLOAT:
		f := &FloatField{FieldBase: *base}
		f.typ = t
		if f.fixed {
			f.values = append(f.values, rand.Float64()*1000000)
			return f
		}
		if rand.Intn(2) == 0 {
			for n := rand.Intn(10) + 1; n > 0; n-- {
				f.values = append(f.values, rand.Float64()*1000000)
			}
		} else {
			f.min = math.Trunc(rand.Float64()*10000) / 10.0
			f.max = math.Trunc(rand.Float64()*100000)/10 + f.min + 1
			f.step = math.Trunc(rand.Float64()*10)/10 + 1
		}
		return f
	case _FT_BOOLEAN:
		f := &BooleanField{FieldBase: *base}
		f.typ = t
		if f.fixed {
			f.values = append(f.values, rand.Intn(2) == 0)
			return f
		}
		if rand.Intn(2) == 0 {
			for n := rand.Intn(10) + 1; n > 0; n-- {
				f.values = append(f.values, rand.Intn(2) == 0)
			}
		}
		return f
	case _FT_NULL:
		f := &NullField{FieldBase: *base}
		f.typ = t
		return f
	case _FT_ARRAY:
		f := &ArrayField{FieldBase: *base}
		f.typ = t
		f.length = rand.Intn(20)
		f.elemType = newRandomFieldWithBase(base.NewBase(""), true, true)
		return f
	case _FT_OBJECT:
		f := &ObjectField{FieldBase: *base}
		f.typ = t
		for n := rand.Intn(10); n > 0; n-- {
			fld := newRandomFieldWithBase(base.NewBase(fmt.Sprintf("gf%d", nextSerial())), true, true)
			f.fields = append(f.fields, fld)
		}
		return f
	default: // _FT_DATE:
		f := &DateField{FieldBase: *base}
		f.typ = t
		if f.fixed {
			f.values = append(f.values, randomTimestamp())
			return f
		}
		if rand.Intn(2) == 0 {
			for n := rand.Intn(30) + 2; n > 0; n-- {
				f.values = append(f.values, randomTimestamp())
			}
		} else {
			f.min = randomTimestamp()
			f.max = randomTimestamp()
			if f.min.After(f.max) {
				f.min, f.max = f.max, f.min
			}
			f.values = append(f.values, f.min)
			switch rand.Intn(7) {
			case 0:
				f.unit = "millisecond"
				f.step = rand.Intn(1000)
			case 1:
				f.unit = "second"
				f.step = rand.Intn(600)
			case 2:
				f.unit = "minute"
				f.step = rand.Intn(600)
			case 3:
				f.unit = "hour"
				f.step = rand.Intn(100)
			case 4:
				f.unit = "day"
				f.step = rand.Intn(100)
			case 5:
				f.unit = "month"
				f.step = rand.Intn(12)
			case 6:
				f.unit = "year"
				f.step = rand.Intn(3)
			}
		}
		return f
	}
}

var randomSb strings.Builder

func getRandomString(max int) string {
	randomSb.Reset()
	writeRandomString(&randomSb, rand.Intn(max))
	return randomSb.String()
}

func writeRandomString(sb *strings.Builder, length int) {
	for ; length > 0; length-- {
		sb.WriteByte(_CHR_SET[rand.Intn(len(_CHR_SET))])
	}
}

func randomTimestamp() time.Time {
	return time.Date(1980+rand.Intn(60), time.Month(rand.Intn(12)), rand.Intn(31), rand.Intn(24), rand.Intn(60), rand.Intn(60),
		rand.Int()%1000000000, time.UTC)
}

// ---------------------------------------------------------------------------------------------------------------------------------
type Field interface {
	Name() string
	Copy() Field
	Type() int
	GenerateProjection(allowWildcards bool) string
	GenerateValue() map[string]interface{}
	GenerateFilter(name string) string
}

const (
	_FT_ANY = iota
	_FT_STRING
	_FT_INT
	_FT_FLOAT
	_FT_BOOLEAN
	_FT_NULL
	_FT_ARRAY
	_FT_OBJECT
	_FT_DATE
	_FT_SIZER // must be last
)

var fieldType = map[int]string{
	_FT_ANY:     "any",
	_FT_STRING:  "string",
	_FT_INT:     "int",
	_FT_FLOAT:   "float",
	_FT_BOOLEAN: "boolean",
	_FT_NULL:    "null",
	_FT_ARRAY:   "array",
	_FT_OBJECT:  "object",
	_FT_DATE:    "date",
}

// ---------------------------------------------------------------------------------------------------------------------------------
type FieldBase struct {
	typ      int    // type constant
	name     string // name
	optional bool   // if the field may be omitted when generating values
	fixed    bool   // if the field generates a fixed value
	nullable bool   // if the field may be generated as NULL
	random   bool   // if the field uses a list of values, if a value is picked at random (or sequentially)
}

func (this *FieldBase) NewBase(n string) *FieldBase {
	res := &FieldBase{
		typ:      this.typ,
		name:     n,
		optional: this.optional,
		fixed:    this.fixed,
		nullable: this.nullable,
		random:   this.random,
	}
	return res
}

func (this *FieldBase) Type() int {
	return this.typ
}

func (this *FieldBase) Name() string {
	return this.name
}

func (this *FieldBase) GenerateProjection(allowWildcards bool) string {
	return this.name
}

func (this *FieldBase) GenerateValue() map[string]interface{} {
	return nil
}

func (this *FieldBase) GenerateFilter(name string) string {
	return ""
}

func (this *FieldBase) generateAsMissing() bool {
	return this.optional && rand.Intn(100) == 0
}

func (this *FieldBase) generateAsMissingFilter() string {
	if this.generateAsMissing() {
		if rand.Intn(10)%2 == 0 {
			return "IS MISSING"
		} else {
			return "IS NOT MISSING"
		}
	}
	return ""
}

func (this *FieldBase) generateAsNull() bool {
	return this.nullable && rand.Intn(100) == 0
}

func (this *FieldBase) generateAsNullFilter() string {
	if this.generateAsNull() {
		if rand.Intn(10)%2 == 0 {
			return "IS NULL"
		} else {
			return "IS NOT NULL"
		}
	}
	return ""
}

func (this *FieldBase) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"name": this.name,
	}
	tn, ok := fieldType[this.typ]
	if !ok {
		tn = "invalid type!"
	}
	m["type"] = tn
	if this.optional {
		m["optional"] = true
	}
	if this.fixed {
		m["fixed"] = true
	}
	if this.nullable {
		m["nullable"] = true
	}
	if this.random {
		m["random"] = true
	}
	return json.Marshal(m)
}

// ---------------------------------------------------------------------------------------------------------------------------------
type StringField struct {
	FieldBase
	values []string
	min    int
	max    int
	prefix string
	suffix string
	count  int // used when generating values
}

func (this *StringField) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"base": &this.FieldBase,
	}
	if len(this.values) > 0 {
		m["values"] = this.values
	}
	if this.min != 0 {
		m["min"] = this.min
	}
	if this.max != 0 {
		m["max"] = this.max
	}
	if len(this.prefix) > 0 {
		m["prefix"] = this.prefix
	}
	if len(this.suffix) > 0 {
		m["suffix"] = this.suffix
	}
	if this.count > 0 {
		m["count"] = this.count
	}
	return json.Marshal(m)
}

func (this *StringField) Copy() Field {
	c := &StringField{}
	*c = *this
	c.name += "_copy"
	return c
}

func (this *StringField) GenerateValue() map[string]interface{} {
	if this.generateAsMissing() {
		return nil
	}
	if this.generateAsNull() {
		return map[string]interface{}{this.name: nil}
	}
	if this.fixed {
		if len(this.values) == 0 {
			return map[string]interface{}{this.name: ""}
		} else {
			return map[string]interface{}{this.name: this.values[0]}
		}
	}
	sb := &strings.Builder{}
	sb.WriteString(this.prefix)
	rng := this.max - this.min
	if len(this.values) > 0 {
		if rng == 0 {
			rng = 1
		}
		if this.random {
			for i := 0; i < rng; i++ {
				if i > 0 {
					sb.WriteRune(' ')
				}
				sb.WriteString(this.values[rand.Intn(len(this.values))])
			}
		} else {
			for i := 0; i < rng; i++ {
				if i > 0 {
					sb.WriteRune(' ')
				}
				n := this.count
				this.count++
				if this.count >= len(this.values) {
					this.count = 0
				}
				sb.WriteString(this.values[n])
			}
		}
	} else {
		if this.random && rng > 0 {
			rng = rand.Intn(rng)
		}
		rng += this.min
		if rng > 0 {
			writeRandomString(sb, rng)
		}
	}
	sb.WriteString(this.suffix)
	return map[string]interface{}{this.name: sb.String()}
}

func (this *StringField) GenerateFilter(name string) string {
	if s := this.generateAsMissingFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if s := this.generateAsNullFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if this.fixed {
		if len(this.values) == 0 {
			return fmt.Sprintf("%s %s \"\"", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))])
		} else {
			return fmt.Sprintf("%s %s \"%s\"", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))], this.values[0])
		}
	}
	sb := &strings.Builder{}
	sb.WriteString(this.prefix)
	rng := this.max - this.min
	like := rand.Intn(10) == 5
	if like {
		sb.WriteRune('%')
	}
	if len(this.values) > 0 {
		if rng == 0 {
			rng = 1
		}
		for i := 0; i < rng; i++ {
			if i > 0 {
				if like {
					sb.WriteRune('%')
				} else {
					sb.WriteRune(' ')
				}
			}
			sb.WriteString(this.values[rand.Intn(len(this.values))])
		}
	} else {
		if this.random && rng > 0 {
			rng = rand.Intn(rng)
		}
		rng += this.min
		if rng > 0 {
			writeRandomString(sb, rng)
		}
	}
	if like {
		sb.WriteRune('%')
	}
	sb.WriteString(this.suffix)
	if like {
		if rand.Intn(10)%2 == 0 {
			return fmt.Sprintf("%s LIKE \"%s\"", name, sb.String())
		} else {
			return fmt.Sprintf("%s NOT LIKE \"%s\"", name, sb.String())
		}
	} else {
		return fmt.Sprintf("%s %s \"%s\"", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))], sb.String())
	}
}

// ---------------------------------------------------------------------------------------------------------------------------------
type IntField struct {
	FieldBase
	values []int
	min    int
	max    int
	step   int
	count  int // used when generating values
	hwm    int // used for filters
	lwm    int
}

func (this *IntField) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"base": &this.FieldBase,
	}
	if len(this.values) > 0 {
		m["values"] = this.values
	}
	if this.min != 0 {
		m["min"] = this.min
	}
	if this.max != 0 && this.max != math.MaxInt {
		m["max"] = this.max
	}
	if this.step != 0 {
		m["step"] = this.step
	}
	if this.hwm != 0 {
		m["hwm"] = this.hwm
	}
	if this.lwm != 0 {
		m["lwm"] = this.lwm
	}
	if this.count != 0 {
		m["count"] = this.count
	}
	return json.Marshal(m)
}

func (this *IntField) Copy() Field {
	c := &IntField{}
	*c = *this
	c.name += "_copy"
	return c
}

func (this *IntField) GenerateValue() map[string]interface{} {
	if this.generateAsMissing() {
		return nil
	}
	if this.generateAsNull() {
		return map[string]interface{}{this.name: nil}
	}
	if this.fixed {
		if len(this.values) == 0 {
			return map[string]interface{}{this.name: this.min}
		} else {
			return map[string]interface{}{this.name: this.values[0]}
		}
	}
	if len(this.values) > 1 {
		if this.random {
			return map[string]interface{}{this.name: this.values[rand.Intn(len(this.values))]}
		}
		n := this.count
		this.count++
		if this.count >= len(this.values) {
			this.count = 0
		}
		return map[string]interface{}{this.name: this.values[n]}
	} else {
		if this.random {
			var n int
			if this.min == this.max {
				n = this.min
			} else {
				n = this.min + rand.Intn(this.max-this.min)
			}
			if this.hwm < n {
				this.hwm = n
			}
			if this.lwm > n {
				this.lwm = n
			}
			return map[string]interface{}{this.name: n}
		}
		var v int
		if len(this.values) >= 1 {
			v = this.values[0]
		} else {
			this.values = append(this.values, 0)
		}

		this.values[0] += this.step
		if this.values[0] > this.max {
			this.values[0] = this.min
		} else if this.values[0] < this.min {
			this.values[0] = this.max
		}
		if this.hwm < this.values[0] {
			this.hwm = this.values[0]
		}
		if this.lwm > this.values[0] {
			this.lwm = this.values[0]
		}
		return map[string]interface{}{this.name: v}
	}
}

func (this *IntField) GenerateFilter(name string) string {
	if s := this.generateAsMissingFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if s := this.generateAsNullFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if this.fixed {
		if len(this.values) == 0 {
			return fmt.Sprintf("%s %s 0", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))])
		} else {
			return fmt.Sprintf("%s %s %d", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))], this.values[0])
		}
	}
	if len(this.values) > 1 {
		return fmt.Sprintf("%s %s %d", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))],
			this.values[rand.Intn(len(this.values))])
	} else {
		n := 0
		if this.hwm > this.lwm {
			n = rand.Intn(this.hwm-this.lwm) + this.lwm
		}
		return fmt.Sprintf("%s %s %d", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))], n)
	}
}

// ---------------------------------------------------------------------------------------------------------------------------------
type FloatField struct {
	FieldBase
	values []float64
	min    float64
	max    float64
	step   float64
	count  int // used when generating values
	hwm    float64
	lwm    float64
}

func (this *FloatField) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"base": &this.FieldBase,
	}
	if len(this.values) > 0 {
		m["values"] = this.values
	}
	if this.min != 0 {
		m["min"] = this.min
	}
	if this.max != 0 && this.max != math.MaxFloat64 {
		m["max"] = this.max
	}
	if this.step != 0 {
		m["step"] = this.step
	}
	if this.hwm != 0 {
		m["hwm"] = this.hwm
	}
	if this.lwm != 0 {
		m["lwm"] = this.lwm
	}
	if this.count != 0 {
		m["count"] = this.count
	}
	return json.Marshal(m)
}

func (this *FloatField) Copy() Field {
	c := &FloatField{}
	*c = *this
	c.name += "_copy"
	return c
}

func (this *FloatField) GenerateValue() map[string]interface{} {
	if this.generateAsMissing() {
		return nil
	}
	if this.generateAsNull() {
		return map[string]interface{}{this.name: nil}
	}
	if this.fixed {
		if len(this.values) == 0 {
			return map[string]interface{}{this.name: this.min}
		} else {
			return map[string]interface{}{this.name: this.values[0]}
		}
	}
	if len(this.values) > 1 {
		if this.random {
			return map[string]interface{}{this.name: this.values[rand.Intn(len(this.values))]}
		}
		n := this.count
		this.count++
		if this.count >= len(this.values) {
			this.count = 0
		}
		return map[string]interface{}{this.name: this.values[n]}
	} else {
		if this.random {
			n := this.min + rand.Float64()*(this.max-this.min)
			if this.hwm < n {
				this.hwm = n
			}
			if this.lwm > n {
				this.lwm = n
			}
			return map[string]interface{}{this.name: n}
		}
		var v float64
		if len(this.values) >= 1 {
			v = this.values[0]
		} else {
			this.values = append(this.values, 0)
		}

		this.values[0] += this.step
		if this.values[0] > this.max {
			this.values[0] = this.min
		} else if this.values[0] < this.min {
			this.values[0] = this.max
		}
		if this.hwm < this.values[0] {
			this.hwm = this.values[0]
		}
		if this.lwm > this.values[0] {
			this.lwm = this.values[0]
		}
		return map[string]interface{}{this.name: v}
	}
}

func (this *FloatField) GenerateFilter(name string) string {
	if s := this.generateAsMissingFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if s := this.generateAsNullFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if this.fixed {
		if len(this.values) == 0 {
			return fmt.Sprintf("%s %s 0.0", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))])
		} else {
			return fmt.Sprintf("%s %s %f", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))], this.values[0])
		}
	}
	if len(this.values) > 1 {
		return fmt.Sprintf("%s %s %f", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))],
			this.values[rand.Intn(len(this.values))])
	} else {
		n := float64(0)
		if this.hwm > this.lwm {
			n = rand.Float64()*(this.hwm-this.lwm) + this.lwm
		}
		return fmt.Sprintf("%s %s %f", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))], n)
	}
}

// ---------------------------------------------------------------------------------------------------------------------------------
type BooleanField struct {
	FieldBase
	values []bool
	count  int // used when generating values
}

func (this *BooleanField) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"base": &this.FieldBase,
	}
	if len(this.values) > 0 {
		m["values"] = this.values
	}
	if this.count != 0 {
		m["count"] = this.count
	}
	return json.Marshal(m)
}

func (this *BooleanField) Copy() Field {
	c := &BooleanField{}
	*c = *this
	c.name += "_copy"
	return c
}

func (this *BooleanField) GenerateValue() map[string]interface{} {
	if this.generateAsMissing() {
		return nil
	}
	if this.generateAsNull() {
		return map[string]interface{}{this.name: nil}
	}
	if this.fixed {
		if len(this.values) == 0 {
			return map[string]interface{}{this.name: false}
		} else {
			return map[string]interface{}{this.name: this.values[0]}
		}
	}
	if len(this.values) > 1 {
		if this.random {
			return map[string]interface{}{this.name: this.values[rand.Intn(len(this.values))]}
		}
		n := this.count
		this.count++
		if this.count >= len(this.values) {
			this.count = 0
		}
		return map[string]interface{}{this.name: this.values[n]}
	} else {
		if this.random {
			return map[string]interface{}{this.name: rand.Intn(10)%2 == 0}
		}
		if len(this.values) == 0 {
			this.values = append(this.values, false)
		}
		v := this.values[0]
		this.values[0] = !this.values[0]
		return map[string]interface{}{this.name: v}
	}
}

func (this *BooleanField) GenerateFilter(name string) string {
	if s := this.generateAsMissingFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if s := this.generateAsNullFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	op := "!="
	if rand.Intn(10)%2 == 0 {
		op = "="
	}
	if this.fixed {
		if len(this.values) == 0 {
			return fmt.Sprintf("%s %s false", name, op)
		} else {
			return fmt.Sprintf("%s %s %v", name, op, this.values[0])
		}
	}
	v := rand.Intn(10)%2 == 0
	return fmt.Sprintf("%s %s %v", name, op, v)
}

// ---------------------------------------------------------------------------------------------------------------------------------
type NullField struct {
	FieldBase
}

func (this *NullField) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"base": &this.FieldBase,
	}
	return json.Marshal(m)
}

func (this *NullField) Copy() Field {
	c := &NullField{}
	*c = *this
	c.name += "_copy"
	return c
}

func (this *NullField) GenerateValue() map[string]interface{} {
	if this.generateAsMissing() {
		return nil
	}
	return map[string]interface{}{this.name: nil}
}

func (this *NullField) GenerateFilter(name string) string {
	if s := this.generateAsMissingFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if s := this.generateAsNullFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if rand.Intn(10)%2 == 0 {
		return fmt.Sprintf("%s IS NULL", name)
	} else {
		return fmt.Sprintf("%s IS NOT NULL", name)
	}
}

// ---------------------------------------------------------------------------------------------------------------------------------
type AnyField struct {
	FieldBase
}

func (this *AnyField) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"base": &this.FieldBase,
	}
	return json.Marshal(m)
}

func (this *AnyField) Copy() Field {
	c := &AnyField{}
	*c = *this
	c.name += "_copy"
	return c
}

func (this *AnyField) randomField() Field {
	this.random = true
	return newRandomFieldWithBase(&this.FieldBase, false, true)
}

func (this *AnyField) GenerateValue() map[string]interface{} {
	// no missing/any processing here so we don't amplify the chances of generating one of them
	return this.randomField().GenerateValue()
}

func (this *AnyField) GenerateFilter(name string) string {
	// no missing/any processing here so we don't amplify the chances of generating one of them
	return this.randomField().GenerateFilter(name)
}

// ---------------------------------------------------------------------------------------------------------------------------------
type ArrayField struct {
	FieldBase
	length   int
	elemType Field
}

func (this *ArrayField) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"base":         &this.FieldBase,
		"length":       this.length,
		"element_type": this.elemType,
	}
	return json.Marshal(m)
}

func (this *ArrayField) Copy() Field {
	c := &ArrayField{}
	*c = *this
	c.name += "_copy"
	return c
}

func (this *ArrayField) GenerateProjection(allowWildcards bool) string {
	n := rand.Intn(this.length + 1)
	if n >= this.length {
		return this.name
	}
	if this.elemType.Type() == _FT_OBJECT && rand.Intn(10) == 8 {
		return fmt.Sprintf("%s[%d]%s", this.name, rand.Intn(this.length), this.elemType.GenerateProjection(allowWildcards))
	} else {
		return fmt.Sprintf("%s[%d]", this.name, rand.Intn(this.length))
	}
}

func (this *ArrayField) GenerateValue() map[string]interface{} {
	if this.generateAsMissing() {
		return nil
	}
	if this.generateAsNull() {
		return map[string]interface{}{this.name: nil}
	}
	n := this.length
	if this.random && n > 0 {
		n = rand.Intn(n)
	}
	if n == 0 {
		return map[string]interface{}{this.name: []interface{}{}}
	}
	array := make([]interface{}, 0, n)
	for ; n > 0; n-- {
		if val := this.elemType.GenerateValue(); val != nil {
			for _, v := range val {
				array = append(array, v)
				break
			}
		}
	}
	return map[string]interface{}{this.name: array}
}

func (this *ArrayField) GenerateFilter(name string) string {
	if s := this.generateAsMissingFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if s := this.generateAsNullFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if this.length == 0 {
		return fmt.Sprintf("%s = []", name)
	}
	// pick a single element to generate the filter on
	n := rand.Intn(this.length)
	return this.elemType.GenerateFilter(fmt.Sprintf("%s[%d]", name, n))
}

// ---------------------------------------------------------------------------------------------------------------------------------
type ObjectField struct {
	FieldBase
	fields []Field
}

func (this *ObjectField) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"base":   &this.FieldBase,
		"fields": this.fields,
	}
	return json.Marshal(m)
}

func (this *ObjectField) Copy() Field {
	c := &ObjectField{}
	*c = *this
	c.name += "_copy"
	return c
}

func (this *ObjectField) GenerateProjection(allowWildcards bool) string {
	n := rand.Intn(len(this.fields) + 1)
	if n >= len(this.fields) {
		if rand.Intn(5) == 3 && allowWildcards {
			return fmt.Sprintf("%s.*", this.name)
		}
		return this.name
	}
	return fmt.Sprintf("%s.%s", this.name, this.fields[n].GenerateProjection(allowWildcards))
}

func (this *ObjectField) GenerateValue() map[string]interface{} {
	if this.generateAsMissing() {
		return nil
	}
	if this.generateAsNull() {
		return map[string]interface{}{this.name: nil}
	}
	obj := make(map[string]interface{}, len(this.fields))
	for i := range this.fields {
		if fd := this.fields[i].GenerateValue(); fd != nil {
			for k, v := range fd {
				obj[k] = v
			}
		}
	}
	return map[string]interface{}{this.name: obj}
}

func (this *ObjectField) GenerateFilter(name string) string {
	if s := this.generateAsMissingFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if s := this.generateAsNullFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if len(this.fields) == 0 {
		return fmt.Sprintf("%s = {}", name)
	}
	// pick a single field to generate the filter on
	n := rand.Intn(len(this.fields))
	return this.fields[n].GenerateFilter(fmt.Sprintf("%s.%s", name, this.fields[n].Name()))
}

// ---------------------------------------------------------------------------------------------------------------------------------
type DateField struct {
	FieldBase
	values []time.Time
	min    time.Time
	max    time.Time
	step   int
	unit   string
	count  int // used when generating values
	hwm    time.Time
	lwm    time.Time
}

func (this *DateField) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"base": &this.FieldBase,
		"unit": this.unit,
	}
	if len(this.values) > 0 {
		m["values"] = this.values
	}
	if !this.min.IsZero() {
		m["min"] = this.min.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if !this.max.IsZero() {
		m["max"] = this.max.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if this.step != 0 {
		m["step"] = this.step
	}
	if !this.hwm.IsZero() {
		m["hwm"] = this.hwm.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if !this.lwm.IsZero() {
		m["lwm"] = this.lwm.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if this.count != 0 {
		m["count"] = this.count
	}
	return json.Marshal(m)
}
func (this *DateField) Copy() Field {
	c := &DateField{}
	*c = *this
	c.name += "_copy"
	return c
}

func (this *DateField) GenerateValue() map[string]interface{} {
	if this.generateAsMissing() {
		return nil
	}
	if this.generateAsNull() {
		return map[string]interface{}{this.name: nil}
	}
	if this.fixed {
		if len(this.values) == 0 {
			return map[string]interface{}{this.name: this.min.Format("2006-01-02T15:04:05.999999999Z07:00")}
		} else {
			return map[string]interface{}{this.name: this.values[0].Format("2006-01-02T15:04:05.999999999Z07:00")}
		}
	}
	if len(this.values) > 1 {
		if this.random {
			return map[string]interface{}{
				this.name: this.values[rand.Intn(len(this.values))].Format("2006-01-02T15:04:05.999999999Z07:00"),
			}
		}
		n := this.count
		this.count++
		if this.count >= len(this.values) {
			this.count = 0
		}
		return map[string]interface{}{this.name: this.values[n].Format("2006-01-02T15:04:05.999999999Z07:00")}
	} else {
		if this.random {
			var res time.Time
			switch this.unit {
			case "millisecond":
				rng := int(this.max.Sub(this.min))
				if rng == 0 {
					rng = 1
				}
				res = this.min.Add(time.Duration(rand.Intn(rng)))
			case "second":
				rng := int(this.max.Sub(this.min) / time.Second)
				if rng == 0 {
					rng = 1
				}
				res = this.min.Add(time.Duration(rand.Intn(rng)) * time.Second)
			case "minute":
				rng := int(this.max.Sub(this.min) / time.Minute)
				if rng == 0 {
					rng = 1
				}
				res = this.min.Add(time.Duration(rand.Intn(rng)) * time.Minute)
			case "hour":
				rng := int(this.max.Sub(this.min) / time.Hour)
				if rng == 0 {
					rng = 1
				}
				res = this.min.Add(time.Duration(rand.Intn(rng)) * time.Hour)
			default: // day, month, year all in discreet days
				rng := int(this.max.Sub(this.min) / (24 * time.Hour))
				if rng == 0 {
					rng = 1
				}
				res = this.min.Add(time.Duration(rand.Intn(rng)) * (24 * time.Hour))
			}
			if this.hwm.Before(res) {
				this.hwm = res
			}
			if this.lwm.After(res) {
				this.lwm = res
			}
			return map[string]interface{}{this.name: res.Format("2006-01-02T15:04:05.999999999Z07:00")}
		}
		var v time.Time
		if len(this.values) >= 1 {
			v = this.values[0]
		} else {
			this.values = append(this.values, this.min)
			v = this.min
		}

		switch this.unit {
		case "millisecond":
			this.values[0] = this.values[0].Add(time.Duration(this.step) * time.Millisecond)
		case "second":
			this.values[0] = this.values[0].Add(time.Duration(this.step) * time.Second)
		case "minute":
			this.values[0] = this.values[0].Add(time.Duration(this.step) * time.Minute)
		case "hour":
			this.values[0] = this.values[0].Add(time.Duration(this.step) * time.Hour)
		case "day":
			this.values[0] = this.values[0].AddDate(0, 0, this.step)
		case "month":
			this.values[0] = this.values[0].AddDate(0, this.step, 0)
		case "year":
			this.values[0] = this.values[0].AddDate(this.step, 0, 0)
		}

		if this.values[0].After(this.max) {
			this.values[0] = this.min
		} else if this.values[0].Before(this.min) {
			this.values[0] = this.max
		}
		if this.hwm.Before(this.values[0]) {
			this.hwm = this.values[0]
		}
		if this.lwm.After(this.values[0]) {
			this.lwm = this.values[0]
		}
		return map[string]interface{}{this.name: v.Format("2006-01-02T15:04:05.999999999Z07:00")}
	}
}

func (this *DateField) GenerateFilter(name string) string {
	if s := this.generateAsMissingFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if s := this.generateAsNullFilter(); s != "" {
		return fmt.Sprintf("%s %s", name, s)
	}
	if this.fixed {
		if len(this.values) == 0 {
			return fmt.Sprintf("%s %s \"%s\"", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))],
				this.min.Format("2006-01-02T15:04:05.999999999Z07:00"))
		} else {
			return fmt.Sprintf("%s %s \"%s\"", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))],
				this.values[0].Format("2006-01-02T15:04:05.999999999Z07:00"))
		}
	}
	if len(this.values) > 1 {
		return fmt.Sprintf("%s %s \"%s\"", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))],
			this.values[rand.Intn(len(this.values))].Format("2006-01-02T15:04:05.999999999Z07:00"))
	} else {
		var res time.Time
		switch this.unit {
		case "millisecond":
			rng := int(this.max.Sub(this.min))
			res = this.min.Add(time.Duration(rand.Intn(rng)))
		case "second":
			rng := int(this.max.Sub(this.min) / time.Second)
			res = this.min.Add(time.Duration(rand.Intn(rng)) * time.Second)
		case "minute":
			rng := int(this.max.Sub(this.min) / time.Minute)
			res = this.min.Add(time.Duration(rand.Intn(rng)) * time.Minute)
		case "hour":
			rng := int(this.max.Sub(this.min) / time.Hour)
			res = this.min.Add(time.Duration(rand.Intn(rng)) * time.Hour)
		default: // day, month, year all in discreet days
			rng := int(this.max.Sub(this.min) / (24 * time.Hour))
			res = this.min.Add(time.Duration(rand.Intn(rng)) * (24 * time.Hour))
		}
		return fmt.Sprintf("%s %s \"%s\"", name, _FILTER_OPS[rand.Intn(len(_FILTER_OPS))],
			res.Format("2006-01-02T15:04:05.999999999Z07:00"))
	}
}

// ---------------------------------------------------------------------------------------------------------------------------------
