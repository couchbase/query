//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"bytes"
	"math"

	"github.com/couchbase/query/value"
)

/*
This represents the concatenation operation for strings.
*/
type Concat struct {
	FunctionBase
}

func NewConcat(operands ...Expression) Function {
	// if the first operand is another Concat, combine the operands
	if len(operands) > 1 {
		if concat, ok := operands[0].(*Concat); ok {
			concat.operands = append(concat.operands, operands[1:]...)
			return concat
		}
	}

	rv := &Concat{}
	rv.Init("concat", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Concat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitConcat(this)
}

func (this *Concat) Type() value.Type { return value.STRING }

func (this *Concat) Evaluate(item value.Value, context Context) (value.Value, error) {
	var buf bytes.Buffer
	null := false

	for _, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		switch arg.Type() {
		case value.STRING:
			if !null {
				buf.WriteString(arg.ToString())
			}
		case value.MISSING:
			return value.MISSING_VALUE, nil
		default:
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(buf.String()), nil
}

/*
Minimum input arguments required for the concatenation
is 2.
*/
func (this *Concat) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for CONCAT is
MaxInt16 = 1<<15 - 1.
*/
func (this *Concat) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *Concat) Constructor() FunctionConstructor {
	return NewConcat
}

/*
This represents the concatenation operation for strings or array of strings with separator.
*/
type Concat2 struct {
	FunctionBase
}

func NewConcat2(operands ...Expression) Function {
	// if the second operand is another Concat2, and the separator match, combine the operands
	// (for Concat2 the first operand is the separator)
	if len(operands) > 2 {
		if concat2, ok := operands[1].(*Concat2); ok {
			if operands[0].EquivalentTo(concat2.operands[0]) {
				concat2.operands = append(concat2.operands, operands[2:]...)
				return concat2
			}
		}
	}

	rv := &Concat2{}
	rv.Init("concat2", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Concat2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Concat2) Type() value.Type { return value.STRING }

func (this *Concat2) Evaluate(item value.Value, context Context) (value.Value, error) {
	var buf bytes.Buffer
	var sp string
	null := false
	addSp := false

	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if !null {
			if (arg.Type() != value.ARRAY && arg.Type() != value.STRING) ||
				(i == 0 && arg.Type() == value.ARRAY) {
				null = true
			} else if arg.Type() == value.STRING {
				if i == 0 {
					sp = arg.ToString()
				} else if !null {
					if addSp && sp != "" {
						buf.WriteString(sp)
					}
					buf.WriteString(arg.ToString())
					addSp = true
				}
			} else if arg.Type() == value.ARRAY {
				for _, ae := range arg.Actual().([]interface{}) {
					ael := value.NewValue(ae)
					if ael.Type() == value.MISSING {
						return value.MISSING_VALUE, nil
					} else if ael.Type() != value.STRING {
						null = true
					} else if !null {
						if addSp && sp != "" {
							buf.WriteString(sp)
						}
						buf.WriteString(ael.ToString())
						addSp = true
					}
				}
			}
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(buf.String()), nil
}

/*
Minimum input arguments required for the concatenation
is 2.
*/
func (this *Concat2) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for CONCAT2 is
MaxInt16 = 1<<15 - 1.
*/
func (this *Concat2) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *Concat2) Constructor() FunctionConstructor {
	return NewConcat2
}
