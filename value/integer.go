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

type intValue int64

var ZERO_NUMBER NumberValue = intValue(0)
var ONE_NUMBER NumberValue = intValue(1)
var NEG_ONE_NUMBER NumberValue = intValue(-1)

var ZERO_VALUE Value = ZERO_NUMBER
var ONE_VALUE Value = ONE_NUMBER
var NEG_ONE_VALUE Value = NEG_ONE_NUMBER

func (this intValue) String() string {
	return strconv.FormatInt(int64(this), 10)
}

func (this intValue) ToString() string {
	return this.String()
}

func (this intValue) MarshalJSON() ([]byte, error) {
	s := strconv.FormatInt(int64(this), 10)
	return []byte(s), nil
}

func (this intValue) WriteJSON(w io.Writer, prefix, indent string, fast bool) error {
	s := strconv.FormatInt(int64(this), 10)
	b := []byte(s)
	_, err := w.Write(b)
	return err
}

/*
Type NUMBER
*/
func (this intValue) Type() Type {
	return NUMBER
}

/*
Cast receiver to float64. We cannot use int64 unless all Expressions
can handle both float64 and int64.
*/
func (this intValue) Actual() interface{} {
	return float64(this)
}

/*
Return int64 and avoid any lossiness due to rounding / representation.
*/
func (this intValue) ActualForIndex() interface{} {
	return int64(this)
}

func (this intValue) Equals(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case intValue:
		if this == other {
			return TRUE_VALUE
		}
	case floatValue:
		if float64(this) == float64(other) {
			return TRUE_VALUE
		}
	}

	return FALSE_VALUE
}

func (this intValue) EquivalentTo(other Value) bool {
	other = other.unwrap()
	switch other := other.(type) {
	case intValue:
		return this == other
	case floatValue:
		return float64(this) == float64(other)
	default:
		return false
	}
}

func (this intValue) Collate(other Value) int {
	other = other.unwrap()
	switch other := other.(type) {
	case intValue:
		switch {
		case this < other:
			return -1
		case this > other:
			return 1
		default:
			return 0
		}
	case floatValue:
		return -other.Collate(this)
	default:
		return int(NUMBER - other.Type())
	}

}

func (this intValue) Compare(other Value) Value {
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
func (this intValue) Truth() bool {
	return this != 0
}

/*
Return receiver
*/
func (this intValue) Copy() Value {
	return this
}

/*
Return receiver
*/
func (this intValue) CopyForUpdate() Value {
	return this
}

/*
Calls missingField.
*/
func (this intValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

/*
Not valid for NUMBER.
*/
func (this intValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

/*
Not valid for NUMBER.
*/
func (this intValue) UnsetField(field string) error {
	return Unsettable(field)
}

/*
Calls missingIndex.
*/
func (this intValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Not valid for NUMBER.
*/
func (this intValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

/*
Returns NULL_VALUE
*/
func (this intValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns NULL_VALUE
*/
func (this intValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns the input buffer as is.
*/
func (this intValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

/*
As number has no fields, return nil.
*/
func (this intValue) Fields() map[string]interface{} {
	return nil
}

func (this intValue) FieldNames(buffer []string) []string {
	return nil
}

/*
Returns the input buffer as is.
*/
func (this intValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	return buffer
}

/*
NUMBER is succeeded by STRING.
*/
func (this intValue) Successor() Value {
	if this < math.MaxInt64 {
		return intValue(this + 1)
	} else {
		return EMPTY_STRING_VALUE
	}
}

func (this intValue) Track() {
}

func (this intValue) Recycle() {
}

func (this intValue) Tokens(set *Set, options Value) *Set {
	set.Add(this)
	return set
}

func (this intValue) ContainsToken(token, options Value) bool {
	return this.EquivalentTo(token)
}

func (this intValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	return matcher(int64(this))
}

func (this intValue) Size() uint64 {
	return uint64(8)
}

func (this intValue) unwrap() Value {
	return this
}

/*
NumberValue methods.
*/

/*
Handle overflow per http://blog.regehr.org/archives/1139
*/
func (this intValue) Add(n NumberValue) NumberValue {
	switch n := n.(type) {
	case intValue:
		rv := intValue(uint64(this) + uint64(n))
		overFlow := (this < 0 && n < 0 && rv >= 0) || (this >= 0 && n >= 0 && rv < 0)
		if !overFlow {
			return rv
		}
	}

	return floatValue(float64(this) + n.Actual().(float64))
}

func (this intValue) IDiv(n NumberValue) Value {
	switch n := n.(type) {
	case intValue:
		if n == 0 {
			return NULL_VALUE
		} else {
			return this / n
		}
	default:
		f := n.Actual().(float64)
		if f == 0.0 {
			return NULL_VALUE
		} else {
			return this / intValue(f)
		}
	}
}

func (this intValue) IMod(n NumberValue) Value {
	switch n := n.(type) {
	case intValue:
		if n == 0 {
			return NULL_VALUE
		} else {
			return this % n
		}
	default:
		f := n.Actual().(float64)
		if f == 0.0 {
			return NULL_VALUE
		} else {
			return this % intValue(f)
		}
	}
}

/*
Handle overflow per
http://stackoverflow.com/questions/1815367/multiplication-of-large-numbers-how-to-catch-overflow
*/
func (this intValue) Mult(n NumberValue) NumberValue {
	switch n := n.(type) {
	case intValue:
		rv := this * n
		if this == 0 || rv/this == n {
			return rv
		}
	}

	return floatValue(float64(this) * n.Actual().(float64))
}

func (this intValue) Neg() NumberValue {
	if this == math.MinInt64 {
		return -floatValue(this)
	}

	return -this
}

func (this intValue) Sub(n NumberValue) NumberValue {
	switch n := n.(type) {
	case intValue:
		if n > math.MinInt64 {
			return this.Add(-n)
		}
	}

	return floatValue(float64(this) - n.Actual().(float64))
}

func (this intValue) Int64() int64 {
	return int64(this)
}

func (this intValue) Float64() float64 {
	return float64(this)
}

func IsInt(x float64) bool {
	return x == float64(int64(x))
}

func IsIntValue(val Value) (int64, bool) {
	actual := val.ActualForIndex()
	switch actual := actual.(type) {
	case float64:
		if IsInt(actual) {
			return int64(actual), true
		}
	case int64:
		return int64(actual), true
	}
	return 0, false
}
