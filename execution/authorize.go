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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Authorize struct {
	base
	dynamicAuthorize bool
	plan             *plan.Authorize
	child            Operator
}

var _AUTH_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_AUTH_OP_POOL, func() interface{} {
		return &Authorize{}
	})
}

func NewAuthorize(plan *plan.Authorize, context *Context, child Operator, dynamicAuthorize bool) *Authorize {
	rv := _AUTH_OP_POOL.Get().(*Authorize)
	rv.plan = plan
	rv.child = child
	rv.dynamicAuthorize = dynamicAuthorize
	newRedirectBase(&rv.base)
	rv.base.setInline()
	rv.output = rv
	return rv
}

func (this *Authorize) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAuthorize(this)
}

func (this *Authorize) Copy() Operator {
	rv := _AUTH_OP_POOL.Get().(*Authorize)
	rv.plan = this.plan
	rv.child = this.child.Copy()
	this.base.copy(&rv.base)
	return rv
}

func (this *Authorize) PlanOp() plan.Operator {
	return this.plan
}

func (this *Authorize) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.SetKeepAlive(1, context) // terminate early
		this.switchPhase(_EXECTIME)
		this.setExecPhase(AUTHORIZE, context)
		defer this.switchPhase(_NOTIME) // accrue current phase's time
		if !active || !context.assert(this.child != nil, "Authorize has no child") {
			this.notify()
			this.fail(context)
			return
		}

		this.switchPhase(_SERVTIME)
		ds := datastore.GetDatastore()
		if ds != nil {
			var perr error
			privs := this.plan.Privileges()
			if privs != nil && this.plan.Dynamic() {
				privs, perr = this.getPrivileges(privs, context, this.dynamicAuthorize)
				if perr != nil {
					context.Fatal(errors.NewError(perr, "dynamic authorization"))
					this.fail(context)
					return
				}
			}

			privs, perr = this.addTxPrivileges(privs, context)
			if perr != nil {
				context.Fatal(errors.NewError(perr, "transactional authorization"))
				this.fail(context)
				return
			}

			authenticatedUsers, err := ds.Authorize(privs, context.Credentials())
			if err != nil {
				context.Fatal(err)
				this.fail(context)
				return
			}
			context.authenticatedUsers = authenticatedUsers
		}

		this.switchPhase(_EXECTIME)
		this.child.SetInput(this.input)
		this.child.SetOutput(this.output)
		this.child.SetStop(nil)
		this.child.SetParent(this)

		this.fork(this.child, context, parent)
	})
}

func (this *Authorize) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	r["~child"] = this.child
	return json.Marshal(r)
}

func (this *Authorize) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*Authorize)
	this.child.accrueTimes(copy.child)
}

func (this *Authorize) SendAction(action opAction) {
	rv := this.baseSendAction(action)
	child := this.child
	if rv && child != nil {
		child.SendAction(action)
	}
}

func (this *Authorize) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.child != nil {
		rv = this.child.reopen(context)
	}
	return rv
}

func (this *Authorize) Done() {
	this.baseDone()
	if this.child != nil {
		child := this.child
		this.child = nil
		child.Done()
	}
	if this.isComplete() {
		_AUTH_OP_POOL.Put(this)
	}
}

func (this *Authorize) getPrivileges(privs *auth.Privileges, context *Context, inclDynamic bool) (*auth.Privileges, error) {
	var rerr error
	nprivs := auth.NewPrivileges()
	privs.ForEach(func(pp auth.PrivilegePair) {
		if (pp.Props & auth.PRIV_PROPS_DYNAMIC_TARGET) != 0 {
			if inclDynamic {
				expr, err := parser.Parse(pp.Target)
				if err == nil {
					var path *algebra.Path
					path, err = getKeyspacePath(expr, context)
					if err == nil && path != nil {
						rprivs, _ := algebra.PrivilegesFromPath(pp.Priv, path)
						nprivs.AddAll(rprivs)
					}
				}
				if err != nil {
					rerr = err
				}
			}
		} else {
			nprivs.AddPair(pp)
		}
	})

	return nprivs, rerr
}
