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
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// KeyScan is used for KEYS clauses (except after JOIN / NEST).
type KeyScan struct {
	base
	plan *plan.KeyScan
	keys map[string]bool
}

var _KEYSCAN_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_KEYSCAN_OP_POOL, func() interface{} {
		return &KeyScan{}
	})
}

func NewKeyScan(plan *plan.KeyScan, context *Context) *KeyScan {
	rv := _KEYSCAN_OP_POOL.Get().(*KeyScan)
	rv.plan = plan
	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
}

func (this *KeyScan) Copy() Operator {
	rv := _KEYSCAN_OP_POOL.Get().(*KeyScan)
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *KeyScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *KeyScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		defer this.cleanup(context)
		if !active {
			return
		}

		// Distinct keys then create map for keys
		if this.plan.Distinct() {
			defer func() {
				this.keys = nil
			}()

			pipelineCap := int(context.GetPipelineCap())
			if pipelineCap <= _STRING_BOOL_POOL.Size() {
				this.keys = _STRING_BOOL_POOL.Get()
				defer func() {
					_STRING_BOOL_POOL.Put(this.keys)
				}()
			} else {
				this.keys = make(map[string]bool, pipelineCap)
			}
		}

		keys, e := this.plan.Keys().Evaluate(parent, &this.operatorCtx)
		if e != nil {
			context.Error(errors.NewEvaluationError(e, "KEYS"))
			return
		}

		actuals := keys.Actual()
		switch actuals := actuals.(type) {
		case []interface{}:
			for _, key := range actuals {
				k := value.NewValue(key).Actual()
				if k, ok := k.(string); ok {
					// Distinct keys
					if this.keys != nil {
						if _, ok1 := this.keys[k]; ok1 {
							continue
						}
						this.keys[k] = true
					}

					av := this.newEmptyDocumentWithKey(key, parent, context)
					if !this.sendItem(av) {
						break
					}
				} else {
					context.Warning(errors.NewWarning(fmt.Sprintf("Document key must be string: %v", k)))
				}
			}
		case string:
			av := this.newEmptyDocumentWithKey(actuals, parent, context)
			if !this.sendItem(av) {
				break
			}
		default:
			context.Warning(errors.NewWarning(fmt.Sprintf("Document key must be string: %v", actuals)))
		}
	})
}

func (this *KeyScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *KeyScan) Done() {
	this.baseDone()
	this.keys = nil
	if this.isComplete() {
		_KEYSCAN_OP_POOL.Put(this)
	}
}
