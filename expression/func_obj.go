//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type nameList []interface{}

func (s nameList) Len() int {
	return len(s)
}

func (s nameList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s nameList) Less(i, j int) bool {
	a, _ := s[i].(string)
	b, _ := s[j].(string)
	return a < b
}

type mapList []interface{}

func (s mapList) Len() int {
	return len(s)
}

func (s mapList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s mapList) Less(i, j int) bool {
	a, _ := s[i].(map[string]interface{})
	b, _ := s[j].(map[string]interface{})
	an, _ := a["name"].(string)
	bn, _ := b["name"].(string)
	return an < bn
}

///////////////////////////////////////////////////
//
// ObjectAdd
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_ADD(expr, expr, expr).
It returns an object containing the source object augmented
with the new name, attribute pair.
It does not do key substitution.
Type ObjectAdd is a struct that implements TernaryFunctionBase.
*/
type ObjectAdd struct {
	TernaryFunctionBase
}

func NewObjectAdd(first, second, third Expression) Function {
	rv := &ObjectAdd{
		*NewTernaryFunctionBase("object_add", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectAdd) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectAdd) Type() value.Type { return value.OBJECT }

/*
This method takes in an object, a name and a value and returns a new
object that contains the name / attribute pair. If the first input is
missing then return a missing value, and if not an object return a
null value.
*/
func (this *ObjectAdd) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	third, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	// Check for type mismatches
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.OBJECT || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	field := second.ToString()

	// we don't overwrite
	_, exists := first.Field(field)
	if exists {
		return first, nil
	}

	// SetField will remove if the attribute is missing, but we don't
	// overwrite anyway, so we might just skip now
	if third.Type() != value.MISSING {
		rv := first.CopyForUpdate()
		rv.SetField(field, third)
		return rv, nil
	}
	return first, nil
}

func (this *ObjectAdd) PropagatesMissing() bool {
	return false
}

func (this *ObjectAdd) PropagatesNull() bool {
	return false
}

/*
Factory method pattern.
*/
func (this *ObjectAdd) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectAdd(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ObjectConcat
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_CONCAT(expr1, expr2 ...).
It returns a new object with the concatenation of the input
objects.
*/
type ObjectConcat struct {
	FunctionBase
}

func NewObjectConcat(operands ...Expression) Function {
	rv := &ObjectConcat{
		*NewFunctionBase("object_concat", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectConcat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectConcat) Type() value.Type { return value.OBJECT }

func (this *ObjectConcat) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	null := false
	missing := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.OBJECT {
			null = true
		} else if !null && !missing {
			if i == 0 {
				rv = arg.CopyForUpdate()
			} else {
				fields := arg.Fields()
				for n, v := range fields {
					rv.SetField(n, v)
				}
			}
		}

	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	return rv, nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ObjectConcat) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ObjectConcat) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ObjectConcat) Constructor() FunctionConstructor {
	return NewObjectConcat
}

///////////////////////////////////////////////////
//
// ObjectInnerPairs
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_INNERPAIRS(expr).
It returns an array containing the attribute name and
value pairs of the object, in N1QL collation order of
the names.
*/
type ObjectInnerPairs struct {
	UnaryFunctionBase
}

func NewObjectInnerPairs(operand Expression) Function {
	rv := &ObjectInnerPairs{
		*NewUnaryFunctionBase("object_innerpairs", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectInnerPairs) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectInnerPairs) Type() value.Type { return value.ARRAY }

/*
This method takes in an object and returns a map of name
value pairs. If the type of input is missing then return
a missing value, and if not an object return a null value.
Convert it to a valid Go type. Cast it to a map from
string to interface. Range over this map and save the keys.
Sort the keys and range over the keys to create name and value
pairs. Return this object.
*/
func (this *ObjectInnerPairs) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := removeMissing(arg)

	var localBuf [_FIELD_CAP]interface{}
	var fields []interface{}
	if len(oa) <= len(localBuf) {
		fields = localBuf[0:0]
	} else {
		fields = _FIELD_POOL.GetCapped(len(oa))
		defer _FIELD_POOL.Put(fields)
	}

	for n, v := range oa {
		fields = append(fields, map[string]interface{}{"name": n, "val": v})
	}

	sort.Sort(mapList(fields))

	return value.NewValue(fields), nil
}

/*
Factory method pattern.
*/
func (this *ObjectInnerPairs) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectInnerPairs(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectInnerValues
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_INNERVALUES(expr).
It returns an array containing the attribute values of
the object, in N1QL collation order.
*/
type ObjectInnerValues struct {
	UnaryFunctionBase
}

func NewObjectInnerValues(operand Expression) Function {
	rv := &ObjectInnerValues{
		*NewUnaryFunctionBase("object_innervalues", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectInnerValues) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectInnerValues) Type() value.Type { return value.ARRAY }

/*
This method takes in an object and returns a slice of values
that contains the attribute names. If the type of input is
missing then return a missing value, and if not an
object return a null value. Convert it to a valid Go type.
Cast it to a map from string to interface. Range over this
map and retrieve the keys. Sort it and then use it to save
the corresponding values into a slice of interfaces. Return
the slice.
*/
func (this *ObjectInnerValues) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := removeMissing(arg)

	var localBuf [_FIELD_CAP]interface{}
	var values []interface{}
	if len(oa) <= len(localBuf) {
		values = localBuf[0:0]
	} else {
		values = _FIELD_POOL.GetCapped(len(oa))
		defer _FIELD_POOL.Put(values)
	}

	for name, _ := range oa {
		values = append(values, name)
	}

	sort.Sort(nameList(values))
	for i, n := range values {
		ns, _ := n.(string)
		values[i] = oa[ns]
	}

	return value.NewValue(values), nil
}

/*
Factory method pattern.
*/
func (this *ObjectInnerValues) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectInnerValues(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectLength
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_LENGTH(expr).
It returns the number of name-value pairs in the object.
*/
type ObjectLength struct {
	UnaryFunctionBase
}

func NewObjectLength(operand Expression) Function {
	rv := &ObjectLength{
		*NewUnaryFunctionBase("object_length", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectLength) Type() value.Type { return value.NUMBER }

func (this *ObjectLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	return value.NewValue(len(oa)), nil
}

/*
Factory method pattern.
*/
func (this *ObjectLength) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectLength(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectNames
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_NAMES(expr).
It returns an array containing the attribute names of
the object, in N1QL collation order.
*/
type ObjectNames struct {
	UnaryFunctionBase
}

func NewObjectNames(operand Expression) Function {
	rv := &ObjectNames{
		*NewUnaryFunctionBase("object_names", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectNames) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectNames) Type() value.Type { return value.ARRAY }

func (this *ObjectNames) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})

	var localBuf [_FIELD_CAP]interface{}
	var names []interface{}
	if len(oa) <= len(localBuf) {
		names = localBuf[0:0]
	} else {
		names = _FIELD_POOL.GetCapped(len(oa))
		defer _FIELD_POOL.Put(names)
	}

	for name, _ := range oa {
		names = append(names, name)
	}

	sort.Sort(nameList(names))

	return value.NewValue(names), nil
}

/*
Factory method pattern.
*/
func (this *ObjectNames) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectNames(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectPaths
//
///////////////////////////////////////////////////

type ObjectPaths struct {
	FunctionBase
}

func NewObjectPaths(operands ...Expression) Function {
	rv := &ObjectPaths{
		*NewFunctionBase("object_paths", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *ObjectPaths) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectPaths) Type() value.Type { return value.ARRAY }

type aNotation int

const (
	subscript aNotation = iota // indexed notation, e.g. [0]
	star                       // all element notation, [*]
	belowStar                  // once under [*], all arrays need [*] too to be selectable
)

func (this *ObjectPaths) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT && arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	unique := true
	aNote := subscript
	comps := true
	index := false
	var re *regexp.Regexp

	if len(this.operands) > 1 {
		options, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if options.Type() != value.OBJECT {
			return value.NULL_VALUE, nil
		}

		if u, ok := options.Field("unique"); ok && u.Type() == value.BOOLEAN {
			unique = u.Truth()
		}
		if as, ok := options.Field("arraysubscript"); ok && as.Type() == value.BOOLEAN {
			if !as.Truth() {
				aNote = star
			}
		}
		if c, ok := options.Field("composites"); ok && c.Type() == value.BOOLEAN {
			comps = c.Truth()
		}
		if p, ok := options.Field("pattern"); ok {
			pattern := p.ToString()
			if len(pattern) > 0 {
				re, err = regexp.Compile(pattern)
				if err != nil {
					return nil, err
				}
			}

		}
		if c, ok := options.Field("index"); ok && c.Type() == value.BOOLEAN {
			index = c.Truth()
		}
	}

	// index==true forces no composites and no array subscript
	if index == true {
		comps = false
		aNote = star
	}

	var nameBuf [_NAME_CAP]string
	var names []string

	if arg.Type() == value.OBJECT {
		oa := arg.Actual().(map[string]interface{})

		l := len(oa)
		l *= 3
		if l <= len(nameBuf) {
			names = nameBuf[0:0]
		} else {
			names = _NAME_POOL.GetCapped(l)
			defer _NAME_POOL.Put(names)
		}
		names = getNames(names, "", oa, aNote, comps, re, index)

	} else { // value.ARRAY
		a := arg.Actual().([]interface{})

		// assume an average of 3 fields per element
		l := len(a) * 3
		if l <= len(nameBuf) {
			names = nameBuf[0:0]
		} else {
			names = _NAME_POOL.GetCapped(l)
			defer _NAME_POOL.Put(names)
		}
		names = getNamesFromArray(names, "", a, aNote, comps, re, index)
	}

	sort.Strings(names)
	ra := make([]interface{}, len(names))
	i := 0
	if unique {
		nprev := ""
		for _, n := range names {
			if nprev != n {
				ra[i] = n
				i++
				nprev = n
			}
		}
	} else {
		for i, n := range names {
			ra[i] = n
		}
		i = len(names)
	}

	return value.NewValue(ra[:i]), nil
}

func getNamesFromArray(names []string, prefix string, a []interface{}, aNote aNotation, comps bool,
	re *regexp.Regexp, index bool) []string {

	for i, val := range a {
		if aNote == subscript {
			names = processValueForNames(names, prefix+fmt.Sprintf("[%d]", i), val, aNote, comps, re, index)
		} else {
			names = processValueForNames(names, prefix, val, belowStar, comps, re, index)
		}
	}
	return names
}

func getNames(names []string, prefix string, m map[string]interface{}, aNote aNotation, comps bool,
	re *regexp.Regexp, index bool) []string {

	if len(prefix) > 0 {
		if aNote == belowStar {
			if index {
				prefix = prefix + "[]."
			} else {
				prefix = prefix + "[*]."
			}
		} else {
			prefix = prefix + "."
		}
	}
	for name, val := range m {
		if strings.IndexAny(name, " \t.`") != -1 || index == true {
			name = strings.Replace(name, "`", "\\u0060", -1)
			name = prefix + "`" + name + "`"
			if comps && (re == nil || re.MatchString(name)) {
				names = append(names, name)
			}
			names = processValueForNames(names, name, val, aNote, comps, re, index)
		} else {
			name = prefix + name
			if comps && (re == nil || re.MatchString(name)) {
				names = append(names, name)
			}
			names = processValueForNames(names, name, val, aNote, comps, re, index)
		}
	}
	return names
}

func processValueForNames(names []string, prefix string, val interface{}, aNote aNotation, comps bool,
	re *regexp.Regexp, index bool) []string {

	withAct, ok := val.(interface{ Actual() interface{} })
	if ok {
		val = withAct.Actual()
	}
	switch ov := val.(type) {
	case []interface{}:
		names = getNamesFromArray(names, prefix, ov, aNote, comps, re, index)
	case map[string]interface{}:
		names = getNames(names, prefix, ov, aNote, comps, re, index)
	default:
		if !comps && (re == nil || re.MatchString(prefix)) {
			names = append(names, prefix)
		}
	}
	return names
}

func (this *ObjectPaths) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectPaths(operands...)
	}
}

/*
Minimum input arguments required is 1.
*/
func (this *ObjectPaths) MinArgs() int { return 1 }

/*
Maximum input arguments allowed.
*/
func (this *ObjectPaths) MaxArgs() int { return 2 }

///////////////////////////////////////////////////
//
// ObjectPairs
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_PAIRS(expr).
It returns an array containing the attribute name and
value pairs of the object, in N1QL collation order of
the names.
*/
type ObjectPairs struct {
	UnaryFunctionBase
}

func NewObjectPairs(operand Expression) Function {
	rv := &ObjectPairs{
		*NewUnaryFunctionBase("object_pairs", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectPairs) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectPairs) Type() value.Type { return value.ARRAY }

/*
This method takes in an object and returns a map of name
value pairs. If the type of input is missing then return
a missing value, and if not an object return a null value.
Convert it to a valid Go type. Cast it to a map from
string to interface. Range over this map and save the keys.
Sort the keys and range over the keys to create name and value
pairs. Return this object.
*/
func (this *ObjectPairs) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})

	var localBuf [_FIELD_CAP]interface{}
	var fields []interface{}
	if len(oa) <= len(localBuf) {
		fields = localBuf[0:0]
	} else {
		fields = _FIELD_POOL.GetCapped(len(oa))
		defer _FIELD_POOL.Put(fields)
	}

	for n, v := range oa {
		fields = append(fields, map[string]interface{}{"name": n, "val": v})
	}

	sort.Sort(mapList(fields))

	return value.NewValue(fields), nil
}

/*
Factory method pattern.
*/
func (this *ObjectPairs) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectPairs(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectPairsNested
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_PAIRS(expr).
It returns an array containing the attribute name and
value pairs of the object, in N1QL collation order of
the names.
*/
type ObjectPairsNested struct {
	FunctionBase
}

func NewObjectPairsNested(operands ...Expression) Function {
	rv := &ObjectPairsNested{
		*NewFunctionBase("object_pairs_nested", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *ObjectPairsNested) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectPairsNested) Type() value.Type { return value.ARRAY }

func (this *ObjectPairsNested) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT && arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	var re *regexp.Regexp
	comps := false
	index := false
	if len(this.operands) > 1 {
		options, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if options.Type() != value.OBJECT {
			return value.NULL_VALUE, nil
		}

		if c, ok := options.Field("composites"); ok && c.Type() == value.BOOLEAN {
			comps = c.Truth()
		}
		if p, ok := options.Field("pattern"); ok {
			pattern := p.ToString()
			if len(pattern) > 0 {
				re, err = regexp.Compile(pattern)
				if err != nil {
					return nil, err
				}
			}

		}
		if c, ok := options.Field("index"); ok && c.Type() == value.BOOLEAN {
			index = c.Truth()
		}
	}

	// index==true forces no composites
	if index == true {
		comps = false
	}

	var pairs util.Pairs

	if arg.Type() == value.OBJECT {
		oa := arg.Actual().(map[string]interface{})

		l := len(oa) * 2
		pairs = make(util.Pairs, 0, l)
		pairs = getPairs(pairs, "", oa, comps, re, index)

	} else { // value.ARRAY
		a := arg.Actual().([]interface{})

		l := len(a) * 3
		pairs = make(util.Pairs, 0, l)
		pairs = getPairsFromArray(pairs, "", a, comps, re, index)
	}

	sort.Sort(pairs)

	rv := make([]interface{}, len(pairs))
	for i, m := range pairs {
		rv[i] = map[string]interface{}{"name": m.Name, "val": m.Value}
	}

	return value.NewValue(rv), nil
}

func getPairsFromArray(pairs util.Pairs, prefix string, a []interface{}, comps bool, re *regexp.Regexp, index bool) util.Pairs {
	if index == true {
		for _, val := range a {
			pairs = processPairValue(pairs, prefix+"[]", val, comps, re, index)
		}
	} else {
		for i, val := range a {
			pairs = processPairValue(pairs, prefix+fmt.Sprintf("[%d]", i), val, comps, re, index)
		}
	}
	return pairs
}

func getPairs(pairs util.Pairs, prefix string, m map[string]interface{}, comps bool, re *regexp.Regexp, index bool) util.Pairs {
	if len(prefix) > 0 {
		prefix = prefix + "."
	}
	for name, val := range m {
		if strings.IndexAny(name, " \t.`") != -1 || index == true {
			name = strings.Replace(name, "`", "\\u0060", -1)
			name = prefix + "`" + name + "`"
		} else {
			name = prefix + name
		}
		if comps && (re == nil || re.MatchString(name)) {
			// only add if it is actually a composite value
			withAct, ok := val.(interface{ Actual() interface{} })
			if ok {
				val = withAct.Actual()
			}
			add := false
			switch val.(type) {
			case []interface{}:
				add = true
			case map[string]interface{}:
				add = true
			}
			if add {
				pairs = append(pairs, util.Pair{Name: name, Value: val})
			}
		}
		pairs = processPairValue(pairs, name, val, comps, re, index)
	}
	return pairs
}

func processPairValue(pairs util.Pairs, prefix string, val interface{}, comps bool, re *regexp.Regexp, index bool) util.Pairs {
	withAct, ok := val.(interface{ Actual() interface{} })
	if ok {
		val = withAct.Actual()
	}
	switch ov := val.(type) {
	case []interface{}:
		pairs = getPairsFromArray(pairs, prefix, ov, comps, re, index)
	case map[string]interface{}:
		pairs = getPairs(pairs, prefix, ov, comps, re, index)
	default:
		if re == nil || re.MatchString(prefix) {
			pairs = append(pairs, util.Pair{Name: prefix, Value: val})
		}
	}
	return pairs
}

/*
Factory method pattern.
*/
func (this *ObjectPairsNested) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectPairsNested(operands...)
	}
}

/*
Minimum input arguments required is 1.
*/
func (this *ObjectPairsNested) MinArgs() int { return 1 }

/*
Maximum input arguments allowed.
*/
func (this *ObjectPairsNested) MaxArgs() int { return 2 }

///////////////////////////////////////////////////
//
// ObjectPut
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_PUT(expr, expr, expr).
It returns an object containing the source object augmented
with the new name, attribute pair.
If the key is found in the object, the corresponding attribute
is replaced by the third argument.
If the third argument is MISSING, the existing key is deleted.
*/
type ObjectPut struct {
	TernaryFunctionBase
}

func NewObjectPut(first, second, third Expression) Function {
	rv := &ObjectPut{
		*NewTernaryFunctionBase("object_put", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectPut) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectPut) Type() value.Type { return value.OBJECT }

/*
This method takes in an object, a name and a value
and returns a new object that contains the name /
attribute pair. If the type of input is missing
then return a missing value, and if not an object
return a null value.
If the key passed already exists, then the attribute
replaces the old attribute. If the attribute is missing
this function behaves like OBJECT_REMOVE.
*/
func (this *ObjectPut) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	third, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	// Check for type mismatches
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.OBJECT || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	field := second.ToString()

	rv := first.CopyForUpdate()
	rv.SetField(field, third)
	return rv, nil
}

func (this *ObjectPut) PropagatesMissing() bool {
	return false
}

func (this *ObjectPut) PropagatesNull() bool {
	return false
}

/*
Factory method pattern.
*/
func (this *ObjectPut) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectPut(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ObjectRemove
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_REMOVE(expr, name ...).  It
returns an object with the name / attribute pair for the name passed
as second parameter removed.
*/
type ObjectRemove struct {
	FunctionBase
}

func NewObjectRemove(operands ...Expression) Function {
	rv := &ObjectRemove{
		*NewFunctionBase("object_remove", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectRemove) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectRemove) Type() value.Type { return value.OBJECT }

/*
This method takes in an object and names and returns
an object with the name / attribute pairs removed.
If the type of input is missing then return a missing value, and
if not an object return a null value.
*/
func (this *ObjectRemove) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	null := false
	missing := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() == value.NULL {
			null = true
		} else if !null && !missing {
			if i == 0 {
				if arg.Type() != value.OBJECT {
					null = true
				} else {
					rv = arg.CopyForUpdate()
				}
			} else {
				if arg.Type() != value.STRING {
					null = true
				} else {
					n := arg.ToString()
					rv.UnsetField(n)
				}
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	return rv, nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ObjectRemove) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ObjectRemove) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ObjectRemove) Constructor() FunctionConstructor {
	return NewObjectRemove
}

///////////////////////////////////////////////////
//
// ObjectRename
//
///////////////////////////////////////////////////

/*
This represents the function OBJECT_RENAME(obj, old_name, new_name).
Returns a new object with the name old_name replaced by new_name.
*/
type ObjectRename struct {
	TernaryFunctionBase
}

func NewObjectRename(obj, old_name, new_name Expression) Function {
	rv := &ObjectRename{
		*NewTernaryFunctionBase("object_rename", obj, old_name, new_name),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectRename) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectRename) Type() value.Type { return value.OBJECT }

func (this *ObjectRename) Evaluate(item value.Value, context Context) (value.Value, error) {
	obj, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	old_name, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	new_name, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	// Check for type mismatches
	if obj.Type() == value.MISSING || old_name.Type() == value.MISSING || new_name.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if obj.Type() != value.OBJECT || old_name.Type() != value.STRING || new_name.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	old := old_name.ToString()
	val, ok := obj.Field(old)
	if !ok {
		return obj, nil
	}

	rv := obj.CopyForUpdate()
	rv.UnsetField(old)
	rv.SetField(new_name.ToString(), val)
	return rv, nil
}

/*
Factory method pattern.
*/
func (this *ObjectRename) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectRename(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ObjectReplace
//
///////////////////////////////////////////////////

/*
This represents the function OBJECT_REPLACE(obj, old_val,
new_val).  Returns a new object with all occurrences of old_val
replaced by new_val.
*/
type ObjectReplace struct {
	TernaryFunctionBase
}

func NewObjectReplace(obj, old_val, new_val Expression) Function {
	rv := &ObjectReplace{
		*NewTernaryFunctionBase("object_replace", obj, old_val, new_val),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectReplace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectReplace) Type() value.Type { return value.OBJECT }

func (this *ObjectReplace) Evaluate(item value.Value, context Context) (value.Value, error) {
	obj, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	old_val, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	new_val, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	// Check for type mismatches
	if obj.Type() == value.MISSING || old_val.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if obj.Type() != value.OBJECT || old_val.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	dup := obj.CopyForUpdate()
	fields := dup.Fields()
	for name, val := range fields {
		if old_val.Equals(value.NewValue(val)).Truth() {
			dup.SetField(name, new_val)
		}
	}

	return dup, nil
}

func (this *ObjectReplace) PropagatesMissing() bool {
	return false
}

func (this *ObjectReplace) PropagatesNull() bool {
	return false
}

/*
Factory method pattern.
*/
func (this *ObjectReplace) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectReplace(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ObjectUnwrap
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_UNWRAP(expr).
Given an object with precisely one name / value pair, it
returns the value.
*/
type ObjectUnwrap struct {
	UnaryFunctionBase
}

func NewObjectUnwrap(operand Expression) Function {
	rv := &ObjectUnwrap{
		*NewUnaryFunctionBase("object_unwrap", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectUnwrap) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectUnwrap) Type() value.Type { return value.JSON }

/*
This method takes in an object and returns the
attribute value. If the type of input is missing
then return a missing value, and if not an object
return a null value.
*/
func (this *ObjectUnwrap) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	if len(oa) == 1 {
		for _, v := range oa {
			return value.NewValue(v), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *ObjectUnwrap) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectUnwrap(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectValues
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_VALUES(expr).
It returns an array containing the attribute values of
the object, in N1QL collation order of the corresponding
names.
*/
type ObjectValues struct {
	UnaryFunctionBase
}

func NewObjectValues(operand Expression) Function {
	rv := &ObjectValues{
		*NewUnaryFunctionBase("object_values", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectValues) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectValues) Type() value.Type { return value.ARRAY }

/*
This method takes in an object and returns a slice
that contains the attribute values. If the type of
input is missing then return a missing value, and
if not an object return a null value.
*/
func (this *ObjectValues) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})

	var localBuf [_FIELD_CAP]interface{}
	var values []interface{}
	if len(oa) <= len(localBuf) {
		values = localBuf[0:0]
	} else {
		values = _FIELD_POOL.GetCapped(len(oa))
		defer _FIELD_POOL.Put(values)
	}

	for name, _ := range oa {
		values = append(values, name)
	}

	sort.Sort(nameList(values))
	for i, n := range values {
		ns, _ := n.(string)
		values[i] = oa[ns]
	}

	return value.NewValue(values), nil
}

/*
Factory method pattern.
*/
func (this *ObjectValues) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectValues(operands[0])
	}
}

/*
Utility function to remove missing array elements for OBJECT_INNERVALUES
and OBJECT_INNERPAIRS
*/
func removeMissing(arg value.Value) map[string]interface{} {
	if len(arg.Actual().(map[string]interface{})) == 1 {
		return arg.Actual().(map[string]interface{})
	}

	oa := arg.Copy().Actual().(map[string]interface{})
	for name, val := range oa {
		valSlice, ok := val.([]interface{})
		if !ok {
			continue
		}
		newSlice := make([]interface{}, 0, len(valSlice))
		for _, subVal := range valSlice {
			if value.NewValue(subVal).Type() != value.MISSING {
				newSlice = append(newSlice, subVal)
			}
		}
		if len(newSlice) == 1 {
			oa[name] = newSlice[0]
		} else {
			oa[name] = newSlice
		}
	}
	return oa
}

///////////////////////////////////////////////////
//
// ObjectExtract
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_EXTRACT(expr...).
It returns an array containing the attribute name and
value pairs of the object, in N1QL collation order of
the names.
*/
type ObjectExtract struct {
	FunctionBase
	re *regexp.Regexp
}

func NewObjectExtract(operands ...Expression) Function {
	rv := &ObjectExtract{
		*NewFunctionBase("object_extract", operands...),
		nil,
	}

	if 2 == len(operands) && operands[1].Type() == value.OBJECT {
		if p, ok := operands[1].Value().Field("pattern"); ok {
			rv.re, _ = precompileRegexp(value.NewValue(p), false)
		}
	}
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectExtract) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectExtract) Type() value.Type { return value.ARRAY }

/*
Filtered ObjectExtract
*/
func (this *ObjectExtract) Evaluate(item value.Value, context Context) (value.Value, error) {
	var min, max, pattern string
	var fields []interface{}

	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	if len(this.operands) > 1 {
		options, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if options.Type() != value.OBJECT {
			return value.NULL_VALUE, nil
		}

		if c, ok := options.Field("min"); ok && c.Type() == value.STRING {
			min = c.ToString()
			if c, ok := options.Field("max"); ok && c.Type() == value.STRING {
				max = c.ToString()
			}
		} else if c, ok := options.Field("max"); ok && c.Type() == value.STRING {
			max = c.ToString()
		} else if p, ok := options.Field("pattern"); ok {
			pattern = p.ToString()
		}
	}

	pfArg, ok := arg.(interface {
		ParsedFields(min, max string, re interface{}) []interface{}
	})
	if ok {
		if len(pattern) != 0 {
			re := this.re
			if re == nil {
				var err error
				re, err = regexp.Compile(pattern)
				if err != nil {
					return nil, err
				}
			}
			fields = pfArg.ParsedFields("", "", re)
		} else {
			fields = pfArg.ParsedFields(min, max, nil)
		}
	} else {
		// fall-back if we don't have the parsed-only interface
		oa := arg.Actual().(map[string]interface{})

		var localBuf [_FIELD_CAP]interface{}
		if len(oa) <= len(localBuf) {
			fields = localBuf[0:0]
		} else {
			fields = _FIELD_POOL.GetCapped(len(oa))
			defer _FIELD_POOL.Put(fields)
		}

		if len(pattern) != 0 {
			re := this.re
			if re == nil {
				var err error
				re, err = regexp.Compile(pattern)
				if err != nil {
					return nil, err
				}
			}
			for n, v := range oa {
				if re.FindStringSubmatchIndex(n) != nil {
					fields = append(fields, map[string]interface{}{"name": n, "val": v})
				}
			}
		} else {
			if len(min) != 0 || len(max) != 0 {
				for n, v := range oa {
					if (len(min) == 0 || strings.Compare(min, n) <= 0) && (len(max) == 0 || strings.Compare(max, n) == 1) {
						fields = append(fields, map[string]interface{}{"name": n, "val": v})
					}
				}
			} else {
				for n, v := range oa {
					fields = append(fields, map[string]interface{}{"name": n, "val": v})
				}
			}
		}
	}
	if nil == fields {
		return value.NULL_VALUE, nil
	}
	return value.NewValue(fields), nil
}

/*
Minimum input arguments required.
*/
func (this *ObjectExtract) MinArgs() int { return 1 }

/*
Maximum input arguments allowed.
*/
func (this *ObjectExtract) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *ObjectExtract) Constructor() FunctionConstructor {
	return NewObjectExtract
}

///////////////////////////////////////////////////
//
// ObjectField
//
///////////////////////////////////////////////////

type ObjectField struct {
	BinaryFunctionBase
	cache Expression
}

func NewObjectField(first, second Expression) Function {
	rv := &ObjectField{
		*NewBinaryFunctionBase("object_field", first, second),
		nil,
	}
	rv.expr = rv
	return rv
}

func (this *ObjectField) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectField) Type() value.Type { return value.ARRAY }

func (this *ObjectField) Evaluate(item value.Value, context Context) (value.Value, error) {

	obj, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	fld, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if obj.Type() == value.MISSING || fld.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if obj.Type() != value.OBJECT || fld.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	exp := this.cache
	static := this.operands[1].Static() != nil
	// only consider using a cached value if second argument is static
	if exp == nil || !static {
		// parse field descriptor expression
		r, e := context.Parse(fld.Actual().(string))
		if e != nil {
			e = errors.NewParsingError(e, this.operands[1].ErrorContext())
			return nil, e
		} else if r == nil {
			return value.NULL_VALUE, nil
		}
		exp, _ = r.(Expression)
		if static {
			this.cache = exp
		}
	}
	res, err := exp.Evaluate(obj, context)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (this *ObjectField) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectField(operands[0], operands[1])
	}
}

const _FIELD_CAP = 16

var _FIELD_POOL = util.NewInterfacePool(256)
