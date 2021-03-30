//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

// noop is a placeholder operator that does nothing
// used to signify to sequences that we had received a plan from a node running an older version,
// which contains an operator we no longer use
package execution

import (
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Noop struct {
	base
}

var _noop = &Noop{}

func NewNoop() *Noop {
	return _noop
}

func (this *Noop) Accept(visitor Visitor) (interface{}, error) {
	return nil, nil
}

func (this *Noop) Copy() Operator {
	return _noop
}

func (this *Noop) PlanOp() plan.Operator {
	return nil
}

func (this *Noop) RunOnce(context *Context, parent value.Value) {
}

func (this *Noop) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}
