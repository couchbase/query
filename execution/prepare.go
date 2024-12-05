//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"

	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Prepare struct {
	base
	plan     *plan.Prepare
	prepared value.Value
}

func NewPrepare(plan *plan.Prepare, context *Context, prepared value.Value) *Prepare {
	rv := &Prepare{
		plan:     plan,
		prepared: prepared,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Prepare) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrepare(this)
}

func (this *Prepare) Copy() Operator {
	rv := &Prepare{
		plan:     this.plan,
		prepared: this.prepared,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Prepare) PlanOp() plan.Operator {
	return this.plan
}

func (this *Prepare) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped
		if !active {
			return
		}

		if this.plan.Force() {
			plan := this.plan.Plan()
			err := prepareds.AddPrepared(plan)
			if err != nil {
				context.Fatal(err)
				return
			}
		}

		// We are going to amend the prepared name, so make a copy not
		// to affect the cache
		val := value.NewValue(this.prepared).Copy()
		host := tenant.EncodeNodeName(distributed.RemoteAccess().WhoAmI())
		name, ok := val.Actual().(map[string]interface{})["name"].(string)
		if host != "" && ok {
			name = distributed.RemoteAccess().MakeKey(host, name)
			val.Actual().(map[string]interface{})["name"] = name
		}

		// encoded_plans are not enabled by default, so we are asked by the SDK team
		// not to send the plan to save on network traffic, so that older SDKs don't
		// cache and send long and useless stuff
		// However, some older SDKs don't like empty encoded plans and treat them as
		// an error, so we send a short plan that decodes correctly but to an empty string
		// To make it more complicated, older engines will probably fail to decode the
		// plan which in turn will confuse the older SDKs even more
		// So, on mixed versions clusters, we still send proper encoded plans
		// This will eventually go away when SDKs older than 3.0 will stop being supported
		if !util.IsFeatureEnabled(context.featureControls, util.N1QL_ENCODED_PLAN) &&
			distributed.RemoteAccess().Enabled(distributed.NEW_PREPAREDS) {
			val.Actual().(map[string]interface{})["encoded_plan"] = prepareds.EmptyPlan
		}
		this.sendItem(value.NewAnnotatedValue(val))
	})
}

func (this *Prepare) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
