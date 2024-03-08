//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
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

	fields := make([]interface{}, 0, len(oa))
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

	values := make([]interface{}, 0, len(oa))
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

	names := make([]interface{}, 0, len(oa))

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
	re *regexp.Regexp
}

func NewObjectPaths(operands ...Expression) Function {
	rv := &ObjectPaths{
		*NewFunctionBase("object_paths", operands...),
		nil,
	}

	if 2 == len(operands) && operands[1].Type() == value.OBJECT {
		rv.re = precompilePattern(operands[1].Value())
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
	var pf pathFilter
	pf.aNote = subscript
	pf.comps = true
	pf.index = false
	pf.fieldPattern = false

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
				pf.aNote = star
			}
		}
		if c, ok := options.Field("composites"); ok && c.Type() == value.BOOLEAN {
			pf.comps = c.Truth()
		}
		if ps, ok := options.Field("patternspace"); ok && ps.Type() == value.STRING {
			switch ps.ToString() {
			case "field":
				pf.fieldPattern = true
			case "path":
				pf.fieldPattern = false
			}
		}
		if i, ok := options.Field("ignorecase"); ok && i.Type() == value.BOOLEAN && i.Truth() {
			pf.ignoreCase = true
		}
		if p, ok := options.Field("pattern"); ok {
			pattern := p.ToString()
			if len(pattern) > 0 {
				pf.re = this.re
				if pf.re == nil {
					if rex, ok := options.Field("regex"); ok && rex.Type() == value.BOOLEAN {
						if !rex.Truth() {
							pattern = regexp.QuoteMeta(pattern)
						}
					}
					if e, ok := options.Field("exact"); ok && e.Type() == value.BOOLEAN {
						if e.Truth() {
							if pattern[0] != '^' {
								pattern = "^" + pattern
							}
							// doesn't matter if we double up on the end anchor
							pattern = pattern + "$"
						}
					}
					if pf.ignoreCase {
						pattern = strings.ToLower(pattern)
					}
					pf.re, err = regexp.Compile(pattern)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		if c, ok := options.Field("index"); ok && c.Type() == value.BOOLEAN {
			pf.index = c.Truth()
		}
	}

	// index==true forces no composites and no array subscript
	if pf.index == true {
		pf.comps = false
		pf.aNote = star
	}

	var nameBuf [_NAME_CAP]string
	var names []string
	var l int
	if arg.Type() == value.OBJECT {
		o := arg.Actual().(map[string]interface{})

		l = len(o)
	} else { // value.ARRAY
		a := arg.Actual().([]interface{})

		l = len(a)
	}

	// assume an average of 3 fields per element
	l *= 3
	if l <= len(nameBuf) {
		names = nameBuf[0:0]
	} else {
		names = _NAME_POOL.GetCapped(l)
		defer _NAME_POOL.Put(names)
	}

	names = pf.processValueForNames(names, "", arg)

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

type pathFilter struct {
	aNote        aNotation
	comps        bool
	re           *regexp.Regexp
	fieldPattern bool
	index        bool
	ignoreCase   bool
}

func (this *pathFilter) getNamesFromArray(names []string, prefix string, a []interface{}) []string {

	var keep aNotation
	for i, val := range a {
		if this.aNote == subscript {
			names = this.processValueForNames(names, prefix+fmt.Sprintf("[%d]", i), val)
		} else {
			keep, this.aNote = this.aNote, belowStar
			names = this.processValueForNames(names, prefix, val)
			this.aNote = keep
		}
	}
	return names
}

func (this *pathFilter) getNames(names []string, prefix string, m map[string]interface{}) []string {

	if len(prefix) > 0 {
		if this.aNote == belowStar {
			if this.index {
				prefix = prefix + "[]."
			} else {
				prefix = prefix + "[*]."
			}
		} else {
			prefix = prefix + "."
		}
	}
	for name, val := range m {
		if strings.IndexAny(name, " \t.`") != -1 || this.index == true {
			name = strings.Replace(name, "`", "\\u0060", -1)
			name = prefix + "`" + name + "`"
		} else {
			name = prefix + name
		}
		if this.comps && matchPattern(name, this.re, this.fieldPattern, this.ignoreCase) {
			names = append(names, name)
		}
		names = this.processValueForNames(names, name, val)
	}
	return names
}

func (this *pathFilter) processValueForNames(names []string, prefix string, val interface{}) []string {

	withAct, ok := val.(interface{ Actual() interface{} })
	if ok {
		val = withAct.Actual()
	}
	switch ov := val.(type) {
	case []interface{}:
		names = this.getNamesFromArray(names, prefix, ov)
	case map[string]interface{}:
		names = this.getNames(names, prefix, ov)
	default:
		if !this.comps && matchPattern(prefix, this.re, this.fieldPattern, this.ignoreCase) {
			names = append(names, prefix)
		}
	}
	return names
}

func matchPattern(s string, re *regexp.Regexp, fieldPattern bool, ignoreCase bool) bool {
	if re == nil {
		return true
	}
	if ignoreCase {
		// pattern will have already been forced to lower case
		s = strings.ToLower(s)
	}
	if !fieldPattern {
		return re.MatchString(s)
	}
	// match on individual fields
	q := false
	start := 0
	for i, c := range s {
		if q {
			if c == '`' {
				q = false
			}
		} else if c == '`' {
			q = true
		} else if c == '.' {
			if re.MatchString(s[start:i]) {
				return true
			}
			start = i + 1
		}
	}
	if start < len(s) && re.MatchString(s[start:]) {
		return true
	}
	return false
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
	FunctionBase
}

func NewObjectPairs(operands ...Expression) Function {
	rv := &ObjectPairs{
		*NewFunctionBase("object_pairs", operands...),
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

	types := false

	if len(this.operands) > 1 {
		options, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if options.Type() != value.OBJECT {
			return value.NULL_VALUE, nil
		}

		if v, ok := options.Field("types"); ok && v.Type() == value.BOOLEAN {
			types = v.Truth()
		}
	}

	oa := arg.Actual().(map[string]interface{})

	fields := make([]interface{}, 0, len(oa))
	if types {
		for n, v := range oa {
			fields = append(fields, map[string]interface{}{"name": n, "type": value.NewValue(v).Type().String()})
		}
	} else {
		for n, v := range oa {
			fields = append(fields, map[string]interface{}{"name": n, "val": v})
		}
	}

	sort.Sort(mapList(fields))

	return value.NewValue(fields), nil
}

func (this *ObjectPairs) Constructor() FunctionConstructor {
	return NewObjectPairs
}

func (this *ObjectPairs) MinArgs() int { return 1 }
func (this *ObjectPairs) MaxArgs() int { return 2 }

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
	re *regexp.Regexp
}

func NewObjectPairsNested(operands ...Expression) Function {
	rv := &ObjectPairsNested{
		*NewFunctionBase("object_pairs_nested", operands...),
		nil,
	}

	if 2 == len(operands) && operands[1].Type() == value.OBJECT {
		rv.re = precompilePattern(operands[1].Value())
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

	var pf pairFilter
	pf.comps = false
	pf.index = false
	pf.fieldPattern = false
	pf.ignoreCase = false
	nameOnly := false
	types := false
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
			pf.comps = c.Truth()
		}
		if r, ok := options.Field("report"); ok && r.Type() == value.STRING {
			switch r.ToString() {
			case "field":
				nameOnly = true
			case "path":
				nameOnly = false
			}
		}
		if ps, ok := options.Field("patternspace"); ok && ps.Type() == value.STRING {
			switch ps.ToString() {
			case "field":
				pf.fieldPattern = true
			case "path":
				pf.fieldPattern = false
			}
		}
		if i, ok := options.Field("ignorecase"); ok && i.Type() == value.BOOLEAN && i.Truth() {
			pf.ignoreCase = true
		}
		if p, ok := options.Field("pattern"); ok {
			pattern := p.ToString()
			if len(pattern) > 0 {
				pf.re = this.re
				if pf.re == nil {
					if rex, ok := options.Field("regex"); ok && rex.Type() == value.BOOLEAN {
						if !rex.Truth() {
							pattern = regexp.QuoteMeta(pattern)
						}
					}
					if e, ok := options.Field("exact"); ok && e.Type() == value.BOOLEAN {
						if e.Truth() {
							if pattern[0] != '^' {
								pattern = "^" + pattern
							}
							// doesn't matter if we double up on the end anchor
							pattern = pattern + "$"
						}
					}
					if pf.ignoreCase {
						pattern = strings.ToLower(pattern)
					}
					pf.re, err = regexp.Compile(pattern)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		if c, ok := options.Field("index"); ok && c.Type() == value.BOOLEAN {
			pf.index = c.Truth()
		}
		if c, ok := options.Field("types"); ok && c.Type() == value.BOOLEAN {
			types = c.Truth()
		}
	}

	// index==true forces no composites
	if pf.index == true {
		pf.comps = false
	}

	var pairs util.Pairs
	var l int

	if arg.Type() == value.OBJECT {
		o := arg.Actual().(map[string]interface{})
		l = len(o)
	} else { // value.ARRAY
		a := arg.Actual().([]interface{})
		l = len(a)
	}

	l *= 3
	pairs = make(util.Pairs, 0, l)
	pairs = pf.processPairValue(pairs, "", arg, "", nameOnly)
	sort.Sort(pairs)

	rv := make([]interface{}, len(pairs))
	if types && !pf.index {
		for i, m := range pairs {
			rv[i] = map[string]interface{}{"name": m.Name, "type": value.NewValue(m.Value).Type().String()}
		}
	} else {
		for i, m := range pairs {
			rv[i] = map[string]interface{}{"name": m.Name, "val": m.Value}
		}
	}

	return value.NewValue(rv), nil
}

type pairFilter struct {
	comps        bool
	re           *regexp.Regexp
	fieldPattern bool
	index        bool
	ignoreCase   bool
}

func (this *pairFilter) getPairsFromArray(pairs util.Pairs, prefix string, a []interface{}, basename string,
	nameOnly bool) util.Pairs {

	if this.index == true {
		// composites are forced to false with index set, so no need to check them here
		for _, val := range a {
			pairs = this.processPairValue(pairs, prefix+"[]", val, "", nameOnly)
		}
	} else {
		for i, val := range a {
			index := fmt.Sprintf("[%d]", i)
			if this.comps && matchPattern(prefix+index, this.re, this.fieldPattern, this.ignoreCase) {
				if nameOnly {
					pairs = addComposite(pairs, basename+index, val)
				} else {
					pairs = addComposite(pairs, prefix+index, val)
				}
			}
			pairs = this.processPairValue(pairs, prefix+index, val, "", nameOnly)
		}
	}
	return pairs
}

func (this *pairFilter) getPairs(pairs util.Pairs, prefix string, m map[string]interface{}, nameOnly bool) util.Pairs {
	if len(prefix) > 0 {
		prefix = prefix + "."
	}
	for name, val := range m {
		basename := name
		if strings.IndexAny(name, " \t.`") != -1 || this.index == true {
			basename = strings.Replace(name, "`", "\\u0060", -1)
			name = prefix + "`" + basename + "`"
		} else {
			name = prefix + name
		}
		if this.comps && matchPattern(name, this.re, this.fieldPattern, this.ignoreCase) {
			if nameOnly {
				pairs = addComposite(pairs, basename, val)
			} else {
				pairs = addComposite(pairs, name, val)
			}
		}
		pairs = this.processPairValue(pairs, name, val, basename, nameOnly)
	}
	return pairs
}

func addComposite(pairs util.Pairs, name string, val interface{}) util.Pairs {
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
	return pairs
}

func (this *pairFilter) processPairValue(pairs util.Pairs, prefix string, val interface{}, basename string,
	nameOnly bool) util.Pairs {

	withAct, ok := val.(interface{ Actual() interface{} })
	if ok {
		val = withAct.Actual()
	}
	switch ov := val.(type) {
	case []interface{}:
		pairs = this.getPairsFromArray(pairs, prefix, ov, basename, nameOnly)
	case map[string]interface{}:
		pairs = this.getPairs(pairs, prefix, ov, nameOnly)
	default:
		if matchPattern(prefix, this.re, this.fieldPattern, this.ignoreCase) {
			if nameOnly {
				pairs = append(pairs, util.Pair{Name: basename, Value: val})
			} else {
				pairs = append(pairs, util.Pair{Name: prefix, Value: val})
			}
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
// ObjectRemoveFields
//
///////////////////////////////////////////////////

type ObjectRemoveFields struct {
	FunctionBase
}

func NewObjectRemoveFields(operands ...Expression) Function {
	rv := &ObjectRemoveFields{
		*NewFunctionBase("object_remove_fields", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *ObjectRemoveFields) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectRemoveFields) Type() value.Type { return value.OBJECT }

func (this *ObjectRemoveFields) Evaluate(item value.Value, context Context) (value.Value, error) {

	if context == nil {
		return nil, errors.NewNilEvaluateParamError("context")
	}

	var rv interface{}
	var obj value.AnnotatedValue
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
					obj = value.NewAnnotatedValue(arg)
				}
			} else {
				if arg.Type() == value.STRING {
					n := arg.ToString()
					if len(n) > 0 {
						exp, err := context.Parse(n)
						if err != nil {
							return nil, errors.NewInvalidExpressionError(n, err.Error())
						}
						if e, ok := exp.(Expression); ok {
							ref, err := getReference(e, obj, context)
							if err != nil {
								return nil, err
							}
							rv = DeleteFromObject(rv, reverseReference(ref))
						} else {
							null = true
						}
					}
				} else if arg.Type() == value.ARRAY {
					act := arg.Actual().([]interface{})
				act_array:
					for j := range act {
						var n string
						switch t := act[j].(type) {
						case value.Value:
							if t.Type() != value.STRING {
								null = true
								break act_array
							}
							n = t.ToString()
						case string:
							n = t
						default:
							null = true
							break act_array
						}
						if len(n) > 0 {
							exp, err := context.Parse(n)
							if err != nil {
								return nil, errors.NewInvalidExpressionError(n, err.Error())
							}
							if e, ok := exp.(Expression); ok {
								ref, err := getReference(e, obj, context)
								if err != nil {
									return nil, err
								}
								rv = DeleteFromObject(rv, reverseReference(ref))
							} else {
								null = true
								break act_array
							}
						}
					}
				} else {
					null = true
				}
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(rv), nil
}

func (this *ObjectRemoveFields) MinArgs() int { return 2 }
func (this *ObjectRemoveFields) MaxArgs() int { return math.MaxInt16 }

func (this *ObjectRemoveFields) Constructor() FunctionConstructor {
	return NewObjectRemoveFields
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

	values := make([]interface{}, 0, len(oa))

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
		rv.re = precompilePattern(operands[1].Value())
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

		fields = make([]interface{}, 0, len(oa))

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
		r, e := context.Parse(fld.ToString())
		if e != nil || r == nil {
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

///////////////////////////////////////////////////
//
// ObjectFilter
//
///////////////////////////////////////////////////

type ObjectFilter struct {
	FunctionBase
	re *regexp.Regexp
}

func NewObjectFilter(operands ...Expression) Function {
	rv := &ObjectFilter{
		*NewFunctionBase("object_filter", operands...),
		nil,
	}

	if 2 == len(operands) && operands[1].Type() == value.OBJECT {
		rv.re = precompilePattern(operands[1].Value())
	}

	rv.expr = rv
	return rv
}

func (this *ObjectFilter) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectFilter) Type() value.Type { return value.ARRAY }

func (this *ObjectFilter) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT && arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	var ff fieldFilter
	ff.aNote = subscript
	ff.fieldPattern = false
	ff.comps = true
	ff.ignoreCase = false

	if len(this.operands) > 1 {
		options, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if options.Type() != value.OBJECT {
			return value.NULL_VALUE, nil
		}

		if as, ok := options.Field("arraysubscript"); ok && as.Type() == value.BOOLEAN {
			if !as.Truth() {
				ff.aNote = star
			}
		}
		if c, ok := options.Field("composites"); ok && c.Type() == value.BOOLEAN {
			ff.comps = c.Truth()
		}
		if ps, ok := options.Field("patternspace"); ok && ps.Type() == value.STRING {
			switch ps.ToString() {
			case "field":
				ff.fieldPattern = true
			case "path":
				ff.fieldPattern = false
			}
		}
		if i, ok := options.Field("ignorecase"); ok && i.Type() == value.BOOLEAN && i.Truth() {
			ff.ignoreCase = true
		}
		if p, ok := options.Field("pattern"); ok {
			pattern := p.ToString()
			if len(pattern) > 0 {
				ff.re = this.re
				if ff.re == nil {
					if rex, ok := options.Field("regex"); ok && rex.Type() == value.BOOLEAN {
						if !rex.Truth() {
							pattern = regexp.QuoteMeta(pattern)
						}
					}
					if e, ok := options.Field("exact"); ok && e.Type() == value.BOOLEAN {
						if e.Truth() {
							if pattern[0] != '^' {
								pattern = "^" + pattern
							}
							// doesn't matter if we double up on the end anchor
							pattern = pattern + "$"
						}
					}
					if ff.ignoreCase {
						pattern = strings.ToLower(pattern)
					}
					ff.re, err = regexp.Compile(pattern)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	obj := ff.processValueForFields("", arg)
	return value.NewValue(obj), nil
}

func (this *ObjectFilter) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectFilter(operands...)
	}
}

/*
Minimum input arguments required is 1.
*/
func (this *ObjectFilter) MinArgs() int { return 1 }

/*
Maximum input arguments allowed.
*/
func (this *ObjectFilter) MaxArgs() int { return 2 }

type fieldFilter struct {
	aNote        aNotation
	comps        bool
	re           *regexp.Regexp
	fieldPattern bool
	ignoreCase   bool
}

func (this *fieldFilter) getFieldsFromArray(prefix string, a []interface{}) []interface{} {

	var res []interface{}
	var keep aNotation
	for i, val := range a {
		var nv interface{}
		if this.aNote == subscript {
			nv = this.processValueForFields(prefix+fmt.Sprintf("[%d]", i), val)
		} else {
			keep, this.aNote = this.aNote, belowStar
			nv = this.processValueForFields(prefix, val)
			this.aNote = keep
		}
		if nv != nil {
			res = append(res, nv)
		}
	}
	return res
}

func (this *fieldFilter) getFields(prefix string, m map[string]interface{}) interface{} {

	if len(prefix) > 0 {
		if this.aNote == belowStar {
			prefix = prefix + "[*]."
		} else {
			prefix = prefix + "."
		}
	}
	res := make(map[string]interface{})
	for name, val := range m {
		var mname string
		if strings.IndexAny(name, " \t.`") != -1 {
			mname = strings.Replace(name, "`", "\\u0060", -1)
			mname = prefix + "`" + mname + "`"
		} else {
			mname = prefix + name
		}
		if this.comps && matchPattern(mname, this.re, this.fieldPattern, this.ignoreCase) {
			res[name] = val
		} else {
			// else check nested
			nv := this.processValueForFields(mname, val)
			if nv != nil {
				res[name] = nv
			}
		}
	}
	return res
}

func (this *fieldFilter) processValueForFields(prefix string, val interface{}) interface{} {

	var res interface{}
	var nv interface{}

	if v, ok := val.(value.Value); ok && v.Type() == value.NULL {
		// distinguish between a NULL value and nil
		nv = val
	}

	if withAct, ok := val.(interface{ Actual() interface{} }); ok {
		val = withAct.Actual()
	}

	switch ov := val.(type) {
	case []interface{}:
		res = this.getFieldsFromArray(prefix, ov)
	case map[string]interface{}:
		res = this.getFields(prefix, ov)
	default:
		if !this.comps && matchPattern(prefix, this.re, this.fieldPattern, this.ignoreCase) {
			if nv != nil {
				res = nv
			} else {
				res = val
			}
		}
	}
	if av, ok := res.([]interface{}); ok {
		if len(av) == 0 {
			res = nil
		}
	} else if ov, ok := res.(map[string]interface{}); ok {
		if len(ov) == 0 {
			res = nil
		}
	}
	return res
}

func precompilePattern(options value.Value) *regexp.Regexp {
	var re *regexp.Regexp
	if p, ok := options.Field("pattern"); ok {
		pattern := p.ToString()
		if rex, ok := options.Field("regex"); ok && rex.Type() == value.BOOLEAN {
			if !rex.Truth() {
				pattern = regexp.QuoteMeta(pattern)
			}
		}
		if e, ok := options.Field("exact"); ok && e.Type() == value.BOOLEAN {
			if e.Truth() {
				if pattern[0] != '^' {
					pattern = "^" + pattern
				}
				// doesn't matter if we double up on the end anchor
				pattern = pattern + "$"
			}
		}
		if i, ok := options.Field("ignorecase"); ok && i.Type() == value.BOOLEAN && i.Truth() {
			pattern = strings.ToLower(pattern)
		}
		re, _ = precompileRegexp(value.NewValue(pattern), false)
	}
	return re
}

/*
 * Deletes the field listed in parts in "ref" from the object "i"
 */
func DeleteFromObject(i interface{}, ref []string) interface{} {
	switch t := i.(type) {
	case value.Value:
		switch t := t.Actual().(type) {
		case map[string]interface{}:
			return DeleteFromObject(t, ref)
		case []interface{}:
			return DeleteFromObject(t, ref)
		default:
			return t
		}
	case map[string]interface{}:
		switch len(ref) {
		case 0:
			return t
		case 1:
			r := ref[0][1:]
			if ref[0][0] == 'i' {
				for k, _ := range t {
					if strings.ToLower(k) == strings.ToLower(r) {
						delete(t, k)
						break
					}
				}
			} else {
				delete(t, r)
			}
		default:
			r := ref[0][1:]
			if ref[0][0] == 'i' {
				for k, field := range t {
					if strings.ToLower(k) == strings.ToLower(r) {
						t[k] = DeleteFromObject(field, ref[1:])
						break
					}
				}
			} else {
				if field, ok := t[r]; ok {
					t[r] = DeleteFromObject(field, ref[1:])
				}
			}
		}
		return t
	case []interface{}:
		if len(ref) < 1 {
			return i
		} else if len(ref) == 1 {
			if ref[0] == "*" {
				t = t[:0]
			} else if strings.IndexRune(ref[0], ':') != -1 {
				parts := strings.Split(ref[0], ":")
				if len(parts) != 2 {
					return i
				}
				s64, err := strconv.ParseUint(parts[0], 10, 64)
				if err != nil || int(s64) >= len(t) {
					return i
				}
				s := int(s64)
				e64, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					return i
				}
				e := int(e64)
				if e < 0 {
					t = t[:s]
				} else if s < e {
					if e > len(t) {
						e = len(t)
					}
					if s == len(t)-1 {
						t = t[:len(t)-1]
					} else {
						copy(t[s:], t[e:])
						t = t[:len(t)-(e-s)]
					}
				}
			} else {
				n64, err := strconv.ParseUint(ref[0], 10, 64)
				if err != nil {
					return i
				}
				n := int(n64)
				if n >= len(t) {
					return i
				}
				if n >= 0 && n < len(t) {
					copy(t[n:], t[n+1:])
					t = t[:len(t)-1]
				}
			}
			return t
		}
		if ref[0] == "*" {
			for n := range t {
				t[n] = DeleteFromObject(t[n], ref[1:])
			}
		} else if strings.IndexRune(ref[0], ':') != -1 {
			parts := strings.Split(ref[0], ":")
			if len(parts) != 2 {
				return i
			}
			s64, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil || int(s64) >= len(t) {
				return i
			}
			s := int(s64)
			e64, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return i
			}
			e := int(e64)
			if e < 0 {
				e = len(t)
			} else if e > len(t) {
				e = len(t)
			}
			if s < len(t) && s < e {
				for i := s; i < e; i++ {
					t[i] = DeleteFromObject(t[i], ref[1:])
				}
			}
		} else {
			n, err := strconv.ParseUint(ref[0], 10, 64)
			if err != nil || int(n) >= len(t) {
				return i
			}
			t[n] = DeleteFromObject(t[n], ref[1:])
		}
		return t
	}
	return i
}

func splitStrings(src string) []string {
	var res []string
	quoted := false
	start := 0
	for i := range src {
		switch src[i] {
		case '`':
			quoted = !quoted
		case ',':
			if !quoted {
				if i-start > 0 {
					s := strings.TrimSpace(src[start:i])
					if len(s) > 0 {
						res = append(res, s)
					}
				}
				start = i + 1
			}
		}
	}
	if !quoted && start < len(src) {
		s := strings.TrimSpace(src[start:])
		if len(s) > 0 {
			res = append(res, s)
		}
	}
	return res
}

func getReference(ex Expression, item value.AnnotatedValue, context Context) ([]string, error) {
	var res []string
	switch e := ex.(type) {
	case *Field:
		if fn, ok := e.Second().(*FieldName); ok {
			if fn.CaseInsensitive() {
				res = append(res, "i"+fn.Alias())
			} else {
				res = append(res, " "+fn.Alias())
			}
		} else {
			r, err := e.Second().Evaluate(item, context)
			if err != nil {
				return nil, err
			}
			if r.Type() == value.NULL || r.Type() == value.MISSING {
				return nil, errors.NewInvalidExpressionError(e.String(), "NULL or missing field name")
			}
			res = append(res, " "+r.ToString())
		}
		ref, err := getReference(e.First(), item, context)
		if err != nil {
			return nil, err
		}
		res = append(res, ref...)
	case *Identifier:
		if e.CaseInsensitive() {
			res = append(res, "i"+e.Alias())
		} else {
			res = append(res, " "+e.Alias())
		}
	case *ArrayStar:
		res = append(res, "*")
		ref, err := getReference(e.Operand(), item, context)
		if err != nil {
			return nil, err
		}
		res = append(res, ref...)
	case *Slice:
		start := 0
		end := -1
		startEx := e.Start()
		if startEx != nil {
			v, err := startEx.Evaluate(item, context)
			if err != nil {
				return nil, err
			} else if v.Type() == value.NUMBER {
				start = int(value.AsNumberValue(v).Int64())
			} else {
				return nil, errors.NewInvalidExpressionError(e.String(), "Invalid slice specification")
			}
		}
		endEx := e.End()
		if endEx != nil {
			v, err := endEx.Evaluate(item, context)
			if err != nil {
				return nil, err
			} else if v.Type() == value.NUMBER {
				end = int(value.AsNumberValue(v).Int64())
				if end < 0 {
					return nil, errors.NewInvalidExpressionError(e.String(), "Invalid slice specification")
				}
			} else {
				return nil, errors.NewInvalidExpressionError(e.String(), "Invalid slice specification")
			}
		}
		if (end >= start || end == -1) && start >= 0 {
			ref, err := getReference(e.Operands()[0], item, context)
			if err != nil {
				return nil, err
			}
			res = append([]string{fmt.Sprintf("%d:%d", start, end)}, ref...)
		} else {
			return nil, errors.NewInvalidExpressionError(e.String(), "Invalid slice specification")
		}
	case *Element:
		second, err := e.Second().Evaluate(item, context)
		if err != nil {
			return nil, err
		} else {
			if second.Type() == value.STRING {
				if second.Actual().(string) == "*" {
					res = append(res, "*")
				} else {
					return nil, errors.NewInvalidExpressionError(e.String(), "Invalid array element")
				}
			} else if second.Type() == value.NUMBER {
				iv := int(second.Actual().(float64))
				if iv >= 0 {
					res = append(res, fmt.Sprintf("%d", iv))
				} else {
					return nil, errors.NewInvalidExpressionError(e.String(), "Invalid array element")
				}
			} else {
				return nil, errors.NewInvalidExpressionError(e.String(), "Invalid array element")
			}
			ref, err := getReference(e.First(), item, context)
			if err != nil {
				return nil, err
			}
			res = append(res, ref...)
		}
	default:
		context.Debugf("Unsupported expression: %v Type: %T", e.String(), e)
		return nil, errors.NewUnsupportedExpressionError(e.String(), "Invalid field reference")
	}
	return res, nil
}

func reverseReference(in []string) []string {
	l := len(in) - 1
	for i := 0; i < len(in)/2; i++ {
		in[i], in[l-i] = in[l-i], in[i]
	}
	return in
}

func GetReferences(exs Expressions, item value.AnnotatedValue, context Context, singleQualification bool) (
	[][]string, bool, error) {

	var references [][]string
	constant := true
	for _, ex := range exs {
		switch ex.(type) {
		case *Field:
			ref, err := getReference(ex, item, context)
			if err != nil {
				return nil, false, err
			}
			if singleQualification {
				ref = checkBinding(item, ref)
			}
			references = append(references, reverseReference(ref))
		case *Identifier:
			ref, err := getReference(ex, item, context)
			if err != nil {
				return nil, false, err
			}
			if singleQualification {
				ref = checkBinding(item, ref)
			}
			references = append(references, reverseReference(ref))
		case *Slice:
			ref, err := getReference(ex, item, context)
			if err != nil {
				return nil, false, err
			}
			if singleQualification {
				ref = checkBinding(item, ref)
			}
			references = append(references, reverseReference(ref))
		case *Element:
			ref, err := getReference(ex, item, context)
			if err != nil {
				return nil, false, err
			}
			if singleQualification {
				ref = checkBinding(item, ref)
			}
			references = append(references, reverseReference(ref))
		case *ArrayStar:
			ref, err := getReference(ex, item, context)
			if err != nil {
				return nil, false, err
			}
			if singleQualification {
				ref = checkBinding(item, ref)
			}
			references = append(references, reverseReference(ref))
		default:
			v, err := ex.Evaluate(item, context)
			if err != nil {
				return nil, false, err
			}
			constant = constant && ex.Static() != nil
			if v.Type() == value.STRING {
				strs := splitStrings(v.ToString())
				for _, str := range strs {
					exp, err := context.Parse(str)
					if err != nil {
						if strings.Index(err.Error(), "syntax error") != -1 {
							err = errors.NewParsingError(err, " "+str)
						}
						return nil, false, err
					}
					if e, ok := exp.(Expression); ok {
						ref, err := getReference(e, item, context)
						if err != nil {
							return nil, false, err
						}
						if singleQualification {
							ref = checkBinding(item, ref)
						}
						references = append(references, reverseReference(ref))
					} else {
						return nil, false, errors.NewInvalidExpressionError(str, nil)
					}
				}
			} else if v.Type() != value.MISSING && v.Type() != value.NULL {
				return nil, false, errors.NewUnsupportedExpressionError(ex.String(), "Does not evaluate to a string")
			}
		}
	}
	if len(references) > 0 {
		logging.Debugf("%v", references)
	}
	return references, constant, nil
}

func checkBinding(item value.AnnotatedValue, ref []string) []string {
	if len(ref) < 1 {
		return ref
	}
	v := item.GetValue().Actual()
	if m, ok := v.(map[string]interface{}); ok {
		if len(m) != 1 {
			return ref
		}
		i := len(ref) - 1
		if ref[i][0] == 'i' {
			for k, _ := range m {
				if strings.ToLower(k) == strings.ToLower(ref[i][1:]) {
					return ref
				}
			}
		} else if _, ok := m[ref[i][1:]]; ok {
			return ref
		}
		for k, _ := range m {
			ref = append(ref, " "+k)
		}
	} else {
		logging.Debugf("unknown type %T", v)
	}
	return ref
}

///////////////////////////////////////////////////
//
// ObjectTypes
//
///////////////////////////////////////////////////

type ObjectTypes struct {
	UnaryFunctionBase
}

func NewObjectTypes(operand Expression) Function {
	rv := &ObjectTypes{
		*NewUnaryFunctionBase("object_types", operand),
	}

	rv.expr = rv
	return rv
}

func (this *ObjectTypes) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectTypes) Type() value.Type { return value.OBJECT }

func (this *ObjectTypes) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	rv := make(map[string]interface{}, len(oa))

	for n, v := range oa {
		rv[n] = value.NewValue(v).Type().String()
	}

	return value.NewValue(rv), nil
}

func (this *ObjectTypes) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectTypes(operands[0])
	}
}

type ObjectTypesNested struct {
	UnaryFunctionBase
}

func NewObjectTypesNested(operand Expression) Function {
	rv := &ObjectTypesNested{
		*NewUnaryFunctionBase("object_types_nested", operand),
	}

	rv.expr = rv
	return rv
}

func (this *ObjectTypesNested) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectTypesNested) Type() value.Type { return value.OBJECT }

func (this *ObjectTypesNested) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(processTypes(arg.Actual())), nil
}

func processTypes(v interface{}) interface{} {
	switch v := v.(type) {
	case value.Value:
		return processTypes(v.Actual())
	case map[string]interface{}:
		nm := make(map[string]interface{}, len(v))
		for k, mv := range v {
			nm[k] = processTypes(mv)
		}
		return nm
	case []interface{}:
		na := make([]interface{}, len(v))
		for i, av := range v {
			na[i] = processTypes(av)
		}
		return na
	default:
		return value.NewValue(v).Type().String()
	}
}

func (this *ObjectTypesNested) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectTypesNested(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectConcat2
//
///////////////////////////////////////////////////

type ObjectConcat2 struct {
	FunctionBase
}

func NewObjectConcat2(operands ...Expression) Function {
	rv := &ObjectConcat2{
		*NewFunctionBase("object_concat2", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *ObjectConcat2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectConcat2) Type() value.Type { return value.OBJECT }

func (this *ObjectConcat2) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	null := false
	missing := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.OBJECT && !(arg.Type() == value.ARRAY && i != 0) {
			null = true
		} else if arg.Type() == value.ARRAY && !null && !missing {
			a := arg.Actual().([]interface{})
			for _, o := range a {
				av := value.NewValue(o)
				if av.Type() != value.OBJECT {
					null = true
					break
				}
				fields := av.Fields()
				for n, v := range fields {
					rv.SetField(n, v)
				}
			}
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

func (this *ObjectConcat2) MinArgs() int { return 2 }

func (this *ObjectConcat2) MaxArgs() int { return math.MaxInt16 }

func (this *ObjectConcat2) Constructor() FunctionConstructor {
	return NewObjectConcat2
}
