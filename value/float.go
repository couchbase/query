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
	"io"
	"math"
	"strconv"

	"github.com/couchbase/query/util"
)

type floatValue float64

var _NAN_BYTES = []byte("\"NaN\"")
var _POS_INF_BYTES = []byte("\"+Infinity\"")
var _NEG_INF_BYTES = []byte("\"-Infinity\"")

func (this floatValue) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this floatValue) ToString() string {
	return this.String()
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

func (this floatValue) WriteJSON(w io.Writer, prefix, indent string, fast bool) error {
	b, err := this.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

/*
Type Number
*/
func (this floatValue) Type() Type {
	return NUMBER
}

func (this floatValue) Actual() interface{} {
	return float64(this)
}

func (this floatValue) ActualForIndex() interface{} {
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
	case intValue:
		if float64(this) == float64(other) {
			return TRUE_VALUE
		}
	}

	return FALSE_VALUE
}

func (this floatValue) EquivalentTo(other Value) bool {
	other = other.unwrap()
	switch other := other.(type) {
	case floatValue:
		return this == other
	case intValue:
		return float64(this) == float64(other)
	default:
		return false
	}
}

func (this floatValue) Collate(other Value) int {
	other = other.unwrap()
	switch other := other.(type) {
	case floatValue:
		t := float64(this)
		o := float64(other)
		return collateFloat(t, o)
	case intValue:
		t := float64(this)
		o := float64(other)
		return collateFloat(t, o)
	default:
		return int(NUMBER - other.Type())
	}

}

func collateFloat(t, o float64) int {
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
}

func (this floatValue) Compare(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	default:
		return intValue(this.Collate(other))
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

func (this floatValue) FieldNames(buffer []string) []string {
	return nil
}

/*
Returns the input buffer as is.
*/
func (this floatValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	return buffer
}

/*
Obey N1QL collation order for numbers. After that, NUMBER is succeeded
by STRING.
*/
func (this floatValue) Successor() Value {
	// NaN sorts lowest
	t := float64(this)

	if math.IsNaN(t) {
		return floatValue(math.Inf(-1))
	}

	// -Inf sorts next
	if math.IsInf(t, -1) {
		return floatValue(-math.MaxFloat64)
	}

	// +Inf sorts last
	if math.IsInf(t, 1) || this >= math.MaxFloat64 {
		return EMPTY_STRING_VALUE
	}

	return floatValue(math.Nextafter(t, math.MaxFloat64))
}

func (this floatValue) Track() {
}

func (this floatValue) Recycle() {
}

func (this floatValue) Tokens(set *Set, options Value) *Set {
	set.Add(this)
	return set
}

func (this floatValue) ContainsToken(token, options Value) bool {
	return this.EquivalentTo(token)
}

func (this floatValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	return matcher(float64(this))
}

func (this floatValue) Size() uint64 {
	return uint64(8)
}

func (this floatValue) unwrap() Value {
	return this
}

/*
NumberValue methods.
*/

func (this floatValue) Add(n NumberValue) NumberValue {
	return floatValue(float64(this) + n.Actual().(float64))
}

func (this floatValue) IDiv(n NumberValue) Value {
	switch n := n.(type) {
	case intValue:
		if n == 0 {
			return NULL_VALUE
		} else {
			return intValue(this) / n
		}
	default:
		f := n.Actual().(float64)
		if f == 0.0 {
			return NULL_VALUE
		} else {
			return intValue(int64(this) / int64(f))
		}
	}
}

func (this floatValue) IMod(n NumberValue) Value {
	switch n := n.(type) {
	case intValue:
		if n == 0 {
			return NULL_VALUE
		} else {
			return intValue(this) % n
		}
	default:
		f := n.Actual().(float64)
		if f == 0.0 {
			return NULL_VALUE
		} else {
			return intValue(int64(this) % int64(f))
		}
	}
}

func (this floatValue) Mult(n NumberValue) NumberValue {
	return floatValue(float64(this) * n.Actual().(float64))
}

func (this floatValue) Neg() NumberValue {
	return -this
}

func (this floatValue) Sub(n NumberValue) NumberValue {
	return floatValue(float64(this) - n.Actual().(float64))
}

func (this floatValue) Int64() int64 {
	return int64(this)
}

func (this floatValue) Float64() float64 {
	return float64(this)
}
