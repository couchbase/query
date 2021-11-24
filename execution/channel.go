//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

// Dummy operator that simply wraps an item channel.
type Channel struct {
	base
}

func NewChannel(context *Context) *Channel {
	rv := &Channel{}
	newBase(&rv.base, context)
	rv.dormant()
	rv.output = rv
	return rv
}

func (this *Channel) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitChannel(this)
}

func (this *Channel) Copy() Operator {
	rv := &Channel{}
	this.base.copy(&rv.base)
	return rv
}

func (this *Channel) PlanOp() plan.Operator {
	return nil
}

// This operator is a no-op. It simply provides a shared itemChannel.
func (this *Channel) RunOnce(context *Context, parent value.Value) {
}

func (this *Channel) MarshalJSON() ([]byte, error) {

	// there's no corresponding plan.Channel, so we have a dummy
	return nil, nil
}
