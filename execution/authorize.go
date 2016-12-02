//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"encoding/json"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Authorize struct {
	base
	plan         *plan.Authorize
	child        Operator
	childChannel StopChannel
}

func NewAuthorize(plan *plan.Authorize, child Operator) *Authorize {
	rv := &Authorize{
		base:         newBase(),
		plan:         plan,
		child:        child,
		childChannel: make(StopChannel, 1),
	}

	rv.output = rv
	return rv
}

func (this *Authorize) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAuthorize(this)
}

func (this *Authorize) Copy() Operator {
	return &Authorize{
		base:         this.base.copy(),
		plan:         this.plan,
		child:        this.child.Copy(),
		childChannel: make(StopChannel, 1),
	}
}

func (this *Authorize) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		timer := time.Now()

		ds := datastore.GetDatastore()
		if ds != nil {
			authenticatedUsers, err := ds.Authorize(this.plan.Privileges(), context.Credentials(), context.OriginalHttpRequest())
			if err != nil {
				context.Fatal(err)
				return
			}
			context.authenticatedUsers = authenticatedUsers
		}

		t := time.Since(timer)
		context.AddPhaseTime(AUTHORIZE, t)
		this.addTime(t)

		this.child.SetInput(this.input)
		this.child.SetOutput(this.output)
		this.child.SetStop(nil)
		this.child.SetParent(this)

		go this.child.RunOnce(context, parent)

		for {
			select {
			case <-this.childChannel: // Never closed
				// Wait for child
				return
			case <-this.stopChannel: // Never closed
				this.notifyStop()
				notifyChildren(this.child)
			}
		}
	})
}

func (this *Authorize) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *Authorize) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	r["child"] = this.child
	return json.Marshal(r)
}
