//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import (
	"math"
	"strconv"
)

/*
Number, represented by floatValue is defined as type float64.
*/
type floatValue float64

/*
The variables ZERO_VALUE and ONE_VALUE are initialized to
0.0 and 1.0 respectively.
*/
var ZERO_VALUE = NewValue(0.0)
var ONE_VALUE = NewValue(1.0)
var NEG_ONE_VALUE = NewValue(-1.0)

var _NAN_BYTES = []byte("\"NaN\"")
var _POS_INF_BYTES = []byte("\"+Infinity\"")
var _NEG_INF_BYTES = []byte("\"-Infinity\"")

func (this floatValue) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this floatValue) MarshalJSON() ([]byte, error) {
	f := float64(this)

	if math.IsNaN(f) {
		return _NAN_BYTES, nil
	} else if math.IsInf(f, 1) {
		return _POS_INF_BYTES, nil
	} else if math.IsInf(f, -1) {
		return _NEG_INF_BYTES, nil
	} else {
		if f == -0 {
			f = 0
		}

		s := strconv.FormatFloat(f, 'f', -1, 64)
		return []byte(s), nil
	}
}

/*
Type Number
*/
func (this floatValue) Type() Type {
	return NUMBER
}

/*
Cast receiver to float64(Go type).
*/
func (this floatValue) Actual() interface{} {
	return float64(this)
}

func (this floatValue) Equals(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case floatValue:
		if this == other {
			return TRUE_VALUE
		}
	}

	return FALSE_VALUE
}

func (this floatValue) Collate(other Value) int {
	other = other.unwrap()
	switch other := other.(type) {
	case floatValue:
		t := float64(this)
		o := float64(other)

		// NaN sorts first
		if math.IsNaN(t) {
			if math.IsNaN(o) {
				return 0
			} else {
				return -1
			}
		}

		if math.IsNaN(o) {
			return 1
		}

		// NegInfinity sorts next
		if math.IsInf(t, -1) {
			if math.IsInf(o, -1) {
				return 0
			} else {
				return -1
			}
		}

		if math.IsInf(o, -1) {
			return 1
		}

		// PosInfinity sorts last
		if math.IsInf(t, 1) {
			if math.IsInf(o, 1) {
				return 0
			} else {
				return 1
			}
		}

		if math.IsInf(o, 1) {
			return -1
		}

		result := t - o
		switch {
		case result < 0.0:
			return -1
		case result > 0.0:
			return 1
		default:
			return 0
		}
	default:
		return int(NUMBER - other.Type())
	}

}

func (this floatValue) Compare(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	default:
		return NewValue(this.Collate(other))
	}
}

/*
Returns true in the event the receiver is not 0 and it isn’t
a NaN value
*/
func (this floatValue) Truth() bool {
	return !math.IsNaN(float64(this)) && this != 0
}

/*
Return receiver
*/
func (this floatValue) Copy() Value {
	return this
}

/*
Return receiver
*/
func (this floatValue) CopyForUpdate() Value {
	return this
}

/*
Calls missingField.
*/
func (this floatValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

/*
Not valid for NUMBER.
*/
func (this floatValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

/*
Not valid for NUMBER.
*/
func (this floatValue) UnsetField(field string) error {
	return Unsettable(field)
}

/*
Calls missingIndex.
*/
func (this floatValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Not valid for NUMBER.
*/
func (this floatValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

/*
Returns NULL_VALUE
*/
func (this floatValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns NULL_VALUE
*/
func (this floatValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns the input buffer as is.
*/
func (this floatValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

/*
As number has no fields, return nil.
*/
func (this floatValue) Fields() map[string]interface{} {
	return nil
}

/*
NUMBER is succeeded by STRING.
*/
func (this floatValue) Successor() Value {
	// Use smallest float32 instead of smallest float64, to leave
	// room for imprecision
	if float64(this) < 0 || (math.MaxFloat64-float64(this)) > _NUMBER_SUCCESSOR_DELTA {
		return NewValue(float64(this) + _NUMBER_SUCCESSOR_DELTA)
	} else {
		return EMPTY_STRING_VALUE
	}
}

func (this floatValue) unwrap() Value {
	return this
}

var _NUMBER_SUCCESSOR_DELTA = float64(1.0e-8)
