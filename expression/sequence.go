//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type SequenceOperation struct {
	NullaryFunctionBase
	next     bool
	defaults []string
	elems    []string
	fullName string
}

func NewSequenceNext(defs ...string) *SequenceOperation {
	rv := &SequenceOperation{
		*NewNullaryFunctionBase("next_sequence_value"),
		true,
		defs,
		nil,
		"",
	}
	rv.expr = rv
	rv.setVolatile()
	return rv
}

func NewSequencePrev(defs ...string) *SequenceOperation {
	rv := &SequenceOperation{
		*NewNullaryFunctionBase("prev_sequence_value"),
		false,
		defs,
		nil,
		"",
	}
	rv.expr = rv
	rv.setVolatile()
	return rv
}

func (this *SequenceOperation) Operator() string {
	if this.next {
		return "next value for " + this.FullName()
	} else {
		return "prev value for " + this.FullName()
	}
}

// Since the parser can't provide the full path in one go, this function is provided so it can be assembled from its components
// This also validates the addition; no more than 3 components may be added.
func (this *SequenceOperation) AddPart(s string) bool {
	if len(this.elems) >= 3 {
		return false
	}
	this.elems = append(this.elems, s)
	this.fullName = ""
	return true
}

// Tests if the available parts make for a valid sequence name
func (this *SequenceOperation) IsNameValid() bool {
	return len(this.defaults)+len(this.elems) >= 4
}

// Construct the full sequence name from the bits we've been provided, filling in the blanks from the defaults (from the
// query_context) provided on construction
func (this *SequenceOperation) FullName() string {
	if this.fullName == "" {
		var b strings.Builder
		b.WriteString(quoteIfNecessary(this.defaults[0]))
		b.WriteRune(':')
		d := util.MinInt(4-len(this.elems), len(this.defaults))
		for i := 1; i < d; i++ {
			b.WriteString(quoteIfNecessary(this.defaults[i]))
			b.WriteRune('.')
		}
		for i := range this.elems {
			if i > 0 {
				b.WriteRune('.')
			}
			b.WriteString(quoteIfNecessary(this.elems[i]))
		}
		this.fullName = b.String()
	}
	return this.fullName
}

func quoteIfNecessary(s string) string {
	if strings.IndexByte(s, '.') >= 0 {
		return "`" + s + "`"
	}
	return s
}

func (this *SequenceOperation) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *SequenceOperation) Type() value.Type { return value.NUMBER }

func (this *SequenceOperation) Evaluate(item value.Value, context Context) (value.Value, error) {
	if context.Readonly() {
		return nil, errors.NewSequenceError(errors.E_SEQUENCE_READ_ONLY_REQ)
	}
	var num int64
	var err errors.Error
	ctx := context.(SequenceContext)
	var seqs map[string]int64
	av, _ := item.(value.AnnotatedValue)
	if av != nil {
		seqs, _ = av.GetAttachment("sequences").(map[string]int64)
		if seqs == nil {
			seqs = make(map[string]int64)
		}
		if num, ok := seqs[this.FullName()]; ok {
			return value.NewValue(num), nil
		}
	}
	if this.next {
		num, err = ctx.NextSequenceValue(this.FullName())
		if err == nil && av != nil {
			seqs[this.FullName()] = num
			av.SetAttachment("sequences", seqs)
		}
	} else {
		num, err = ctx.PrevSequenceValue(this.FullName())
		if err != nil && err.Code() == errors.W_SEQUENCE_NO_PREV_VALUE {
			return value.MISSING_VALUE, err
		}
	}
	if err != nil {
		return nil, err
	}
	return value.NewValue(num), nil
}

func (this *SequenceOperation) Static() Expression {
	return this
}

func (this *SequenceOperation) Copy() Expression {
	rv := &SequenceOperation{
		*NewNullaryFunctionBase(this.name),
		this.next,
		this.defaults,
		this.elems,
		this.fullName,
	}
	rv.expr = rv
	rv.setVolatile()
	return rv
}

func (this *SequenceOperation) Constructor() FunctionConstructor {
	// not a regular function
	return nil
}

func (this *SequenceOperation) Privileges() *auth.Privileges {
	privs := this.ExpressionBase.Privileges()
	privs.Add(this.FullName(), auth.PRIV_QUERY_USE_SEQUENCES, auth.PRIV_PROPS_NONE)
	return privs
}

func (this *SequenceOperation) Indexable() bool {
	return false
}
