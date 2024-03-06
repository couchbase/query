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

type Merge struct {
	base
	plan     *plan.Merge
	update   Operator
	delete   Operator
	insert   Operator
	matched  map[string]bool
	inserted map[string]bool
	children []Operator
	inputs   []*Channel
}

func NewMerge(plan *plan.Merge, context *Context, update, delete, insert Operator) *Merge {
	rv := &Merge{
		plan:   plan,
		update: update,
		delete: delete,
		insert: insert,
	}

	newBase(&rv.base, context)
	rv.trackChildren(3)
	rv.output = rv
	return rv
}

func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
}

func (this *Merge) Copy() Operator {
	rv := &Merge{
		plan:   this.plan,
		update: copyOperator(this.update),
		delete: copyOperator(this.delete),
		insert: copyOperator(this.insert),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Merge) PlanOp() plan.Operator {
	return this.plan
}

func (this *Merge) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		this.setExecPhase(MERGE, context)
		defer this.switchPhase(_NOTIME) // accrue current phase's time
		defer this.notify()             // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		this.fork(this.input, context, parent)

		update, updateInput := this.wrapChild(this.update, context)
		delete, deleteInput := this.wrapChild(this.delete, context)
		insert, insertInput := this.wrapChild(this.insert, context)

		this.children = _MERGE_OPERATOR_POOL.Get()
		this.inputs = _MERGE_CHANNEL_POOL.Get()
		if update != nil || delete != nil {
			this.matched = _MERGE_KEY_POOL.Get()
		}
		if insert != nil {
			this.inserted = _MERGE_KEY_POOL.Get()
		}
		defer func() {
			if this.matched != nil {
				_MERGE_KEY_POOL.Put(this.matched)
				this.matched = nil
			}
			if this.inserted != nil {
				_MERGE_KEY_POOL.Put(this.inserted)
				this.inserted = nil
			}
		}()

		if update != nil {
			this.children = append(this.children, update)
			this.inputs = append(this.inputs, updateInput)
			update.SetStop(newActionStopNotifier(this))
		}

		if delete != nil {
			this.children = append(this.children, delete)
			this.inputs = append(this.inputs, deleteInput)
			delete.SetStop(newActionStopNotifier(this))
		}

		if insert != nil {
			this.children = append(this.children, insert)
			this.inputs = append(this.inputs, insertInput)
			insert.SetStop(newActionStopNotifier(this))
		}

		for _, child := range this.children {
			this.fork(child, context, parent)
		}

		var item value.AnnotatedValue
		ok := true

		for ok {
			item, ok = this.getItem()
			if !ok || item == nil {
				break
			}
			this.addInDocs(1)
			if this.ValueExchange().stoppedChildren() > 0 {
				break
			}
			if this.plan.IsOnKey() {
				ok = this.processKeyMatch(item, context, update, delete, insert)
			} else {
				ok = this.processAction(item, context, "", update, delete, insert)
			}
		}

		// Close child input Channels, which will signal children
		for _, input := range this.inputs {
			input.close(context)
		}

		// Wait for all children
		this.childrenWaitNoStop(this.children...)
	})
}

func (this *Merge) processKeyMatch(item value.AnnotatedValue,
	context *Context, update, delete, insert Operator) bool {
	kv, e := this.plan.Key().Evaluate(item, &this.operatorCtx)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, "MERGE key"))
		return false
	}

	ka := kv.Actual()
	k, ok := ka.(string)
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid MERGE key %v of type %T.", ka, ka)))
		return false
	}

	this.switchPhase(_SERVTIME)

	ok = true
	bvs := make(map[string]value.AnnotatedValue, 1)
	errs := this.plan.Keyspace().Fetch([]string{k}, bvs, context, nil, nil, false)

	this.switchPhase(_EXECTIME)

	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			ok = false
		}
	}

	if !ok {
		return false
	}

	if len(bvs) > 0 {
		item.SetField(this.plan.KeyspaceRef().Alias(), bvs[k])
	}

	return this.processAction(item, context, k, update, delete, insert)
}

func (this *Merge) processAction(item value.AnnotatedValue, context *Context,
	insertKey string, update, delete, insert Operator) bool {

	var tv value.Value
	var tav value.AnnotatedValue
	var key string
	match := false
	ok1 := true

	tv, ok1 = item.Field(this.plan.KeyspaceRef().Alias())
	if ok1 {
		tav, ok1 = tv.(value.AnnotatedValue)
		if !ok1 {
			context.Error(errors.NewExecutionInternalError("Merge.processAction: Not an annotated value"))
			return false
		}

		key, ok1 = this.getDocumentKey(tav, context)
		if !ok1 {
			return false
		}

		// check whether the matched document was inserted as part of
		// INSERT action of this MERGE statement, if so, treat it as unmatched
		if insert == nil {
			match = true
		} else if _, ok1 = this.inserted[key]; !ok1 {
			match = true
		}
	}

	ok := true
	if match {
		// Perform UPDATE and/or DELETE
		check := true
		if update != nil {
			matched := true
			if this.plan.UpdateFilter() != nil {
				val, err := this.plan.UpdateFilter().Evaluate(item, &this.operatorCtx)
				matched = err == nil && val.Truth()
			}
			if matched {
				// make sure document is not updated multiple times
				if this.matched[key] {
					context.Error(errors.NewMergeMultiUpdateError(key))
					return false
				}
				item1 := item
				if delete != nil {
					item1 = item.CopyForUpdate().(value.AnnotatedValue)
					if context.UseRequestQuota() {
						err := context.TrackValueSize(item1.Size())
						if err != nil {
							context.Error(err)
							item1.Recycle()
							item.Recycle()
							return false
						}
					}
				}
				this.matched[key] = true
				check = false
				ok = this.sendItemOp(update.Input(), item1)
			} else if delete == nil {
				item.Recycle()
			}
		}
		if delete != nil {
			if ok {
				matched := true
				if this.plan.DeleteFilter() != nil {
					val, err := this.plan.DeleteFilter().Evaluate(item, &this.operatorCtx)
					matched = err == nil && val.Truth()
				}
				if matched {
					// make sure document is not updated multiple times
					if check && this.matched[key] {
						context.Error(errors.NewMergeMultiUpdateError(key))
						return false
					}
					this.matched[key] = true
					ok = this.sendItemOp(delete.Input(), item)
				} else {
					item.Recycle()
				}
			} else {
				item.Recycle()
			}
		}
	} else {
		// Not matched; INSERT
		if insert != nil {
			if insertKey != "" {
				key = insertKey
			} else {
				ins, ok1 := insert.(*SendInsert)
				if !ok1 {
					context.Error(errors.NewExecutionInternalError("Merge.processAction: incorrect type for insert operator"))
					return false
				}
				kv, e := ins.plan.Key().Evaluate(item, &this.operatorCtx)
				if e != nil {
					context.Error(errors.NewEvaluationError(e, "MERGE INSERT key"))
					return false
				}
				key, ok1 = kv.Actual().(string)
				if !ok1 {
					context.Error(errors.NewInsertKeyTypeError(kv))
					return false
				}
			}
			matched := true
			if this.plan.InsertFilter() != nil {
				val, err := this.plan.InsertFilter().Evaluate(item, &this.operatorCtx)
				matched = err == nil && val.Truth()
			}
			if matched {
				if this.inserted[key] {
					context.Error(errors.NewMergeMultiInsertError(key))
					return false
				}
				ok = this.sendItemOp(insert.Input(), item)
				this.inserted[key] = true
			} else {
				item.Recycle()
			}
		}
	}

	return ok
}

func (this *Merge) wrapChild(op Operator, context *Context) (Operator, *Channel) {
	if op == nil {
		return nil, nil
	}

	ch := NewChannel(context)
	op.SetInput(ch)
	op.SetOutput(this.output)
	op.SetParent(this)
	return op, ch
}

func (this *Merge) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		if this.update != nil {
			r["update"] = this.update
		}
		if this.delete != nil {
			r["delete"] = this.delete
		}
		if this.insert != nil {
			r["insert"] = this.insert
		}
	})
	return json.Marshal(r)
}

func (this *Merge) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*Merge)
	if this.update != nil {
		this.insert.accrueTimes(copy.insert)
	}
	if this.delete != nil {
		this.update.accrueTimes(copy.update)
	}
	if this.insert != nil {
		this.insert.accrueTimes(copy.insert)
	}
}

func (this *Merge) SendAction(action opAction) {
	this.baseSendAction(action)
	update := this.update
	delete := this.delete
	insert := this.insert
	if update != nil {
		update.SendAction(action)
	}
	if delete != nil {
		delete.SendAction(action)
	}
	if insert != nil {
		insert.SendAction(action)
	}
}

func (this *Merge) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.update != nil {
		rv = this.update.reopen(context)
	}
	if rv && this.delete != nil {
		rv = this.delete.reopen(context)
	}
	if rv && this.insert != nil {
		rv = this.insert.reopen(context)
	}
	return rv
}

func (this *Merge) Done() {
	this.baseDone()
	if this.update != nil {
		update := this.update
		this.update = nil
		update.Done()
	}
	if this.delete != nil {
		delete := this.delete
		this.delete = nil
		delete.Done()
	}
	if this.insert != nil {
		insert := this.insert
		this.insert = nil
		insert.Done()
	}
	_MERGE_OPERATOR_POOL.Put(this.children)
	this.children = nil

	inputs := this.inputs
	this.inputs = nil
	for _, input := range inputs {
		input.Done()
	}
	_MERGE_CHANNEL_POOL.Put(inputs)
}

var _MERGE_OPERATOR_POOL = NewOperatorPool(3)
var _MERGE_CHANNEL_POOL = NewChannelPool(3)
var _MERGE_KEY_POOL = util.NewStringBoolPool(1024)

// The purpose of this type is to accept a stop notification from an action and relay that to the merge by stopping the merge's
// valueExchange.  This will wake it if it is waiting along with preventing further exchange actions, allowing the operator to end
// cleanly.
type actionStopNotifier struct {
	exchange *valueExchange
}

func (this *actionStopNotifier) SendAction(action opAction) {
	this.exchange.sendStop()
}

func newActionStopNotifier(op Operator) *actionStopNotifier {
	return &actionStopNotifier{exchange: &op.getBase().valueExchange}
}
