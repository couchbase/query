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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type SendUpsert struct {
	base
	plan     *plan.SendUpsert
	keyspace datastore.Keyspace
}

func NewSendUpsert(plan *plan.SendUpsert, context *Context) *SendUpsert {
	rv := &SendUpsert{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.execPhase = UPSERT
	rv.output = rv
	return rv
}

func (this *SendUpsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpsert(this)
}

func (this *SendUpsert) Copy() Operator {
	rv := &SendUpsert{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *SendUpsert) PlanOp() plan.Operator {
	return this.plan
}

func (this *SendUpsert) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SendUpsert) beforeItems(context *Context, parent value.Value) bool {
	this.keyspace = getKeyspace(this.plan.Keyspace(), this.plan.Term().ExpressionTerm(), context)
	return this.keyspace != nil
}

func (this *SendUpsert) processItem(item value.AnnotatedValue, context *Context) bool {
	return this.enbatch(item, this, context)
}

func (this *SendUpsert) afterItems(context *Context) {
	this.flushBatch(context)
	context.ReleaseSkipKeys()
}

func (this *SendUpsert) flushBatch(context *Context) bool {
	defer this.releaseBatch(context)

	if len(this.batch) == 0 || !this.isRunning() {
		return true
	}

	var dpairs []value.Pair
	if _UPSERT_POOL.Size() >= len(this.batch) {
		dpairs = _UPSERT_POOL.Get()
		defer _UPSERT_POOL.Put(dpairs)
	} else {
		dpairs = make([]value.Pair, 0, len(this.batch))
	}

	keyExpr := this.plan.Key()
	valExpr := this.plan.Value()
	optionsExpr := this.plan.Options()
	var err error
	var ok, copyOptions bool
	i := 0

	for _, av := range this.batch {
		copyOptions = true
		dpairs = dpairs[0 : i+1]
		dpair := &dpairs[i]
		var key, val, options value.Value

		if keyExpr != nil {
			// UPSERT ... SELECT
			key, err = keyExpr.Evaluate(av, context)
			if err != nil {
				context.Error(errors.NewEvaluationError(err,
					fmt.Sprintf("UPSERT key for %v", av.GetValue())))
				continue
			}

			if valExpr != nil {
				val, err = valExpr.Evaluate(av, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err,
						fmt.Sprintf("UPSERT value for %v", av.GetValue())))
					continue
				}
			} else {
				val = av
			}

			if context.UseRequestQuota() {
				context.ReleaseValueSize(av.Size())
			}
			if optionsExpr != nil {
				options, err = optionsExpr.Evaluate(av, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err,
						fmt.Sprintf("UPSERT value for %v", av.GetValue())))
					continue
				}
				if optionsExpr.Value() == nil || options.Equals(optionsExpr.Value()) != value.TRUE_VALUE {
					copyOptions = false
				}
			}
		} else {
			// UPSERT ... VALUES
			key, ok = av.GetAttachment("key").(value.Value)
			if !ok {
				context.Error(errors.NewUpsertKeyError(av.GetValue()))
				continue
			}

			val, ok = av.GetAttachment("value").(value.Value)
			if !ok {
				context.Error(errors.NewUpsertValueError(av.GetValue()))
				continue
			}
			if context.UseRequestQuota() {
				context.ReleaseValueSize(key.Size() + val.Size())
			}

			options, _ = av.GetAttachment("options").(value.Value)
		}

		dpair.Name, ok = key.Actual().(string)
		if !ok {
			context.Error(errors.NewUpsertKeyTypeError(key))
			continue
		}
		if context.SkipKey(dpair.Name) {
			context.Error(errors.NewUpsertKeyAlreadyMutatedError(this.keyspace.QualifiedName(), dpair.Name))
			return false // halt mutations
		}

		if options != nil && options.Type() != value.OBJECT {
			context.Error(errors.NewInsertOptionsTypeError(options))
			continue
		}

		dpair.Options = adjustExpiration(options, copyOptions)
		expiration, _ := getExpiration(dpair.Options)
		// UPSERT can preserve expiration, but we can't get old value without read for RETURNING clause.
		dpair.Value = this.setDocumentKey(dpair.Name, value.NewAnnotatedValue(val), expiration, context)
		i++
	}

	dpairs = dpairs[0:i]

	this.switchPhase(_SERVTIME)

	// Perform the actual UPSERT
	var errs errors.Errors
	dpairs, errs = this.keyspace.Upsert(dpairs, context)

	this.switchPhase(_EXECTIME)

	// Update mutation count with number of upserted docs
	context.AddMutationCount(uint64(len(dpairs)))

	mutationOk := true
	if len(errs) > 0 {
		context.Errors(errs)
		if context.txContext != nil {
			mutationOk = false
		}
	}

	// Capture the upserted keys in case there is a RETURNING clause
	for _, dp := range dpairs {
		if !context.AddKeyToSkip(dp.Name) {
			return false
		}
		dv := value.NewAnnotatedValue(dp.Value)
		av := value.NewAnnotatedValue(make(map[string]interface{}, 1))
		av.CopyAnnotations(dv)
		av.SetField(this.plan.Alias(), dv)
		if context.UseRequestQuota() {
			err := context.TrackValueSize(av.Size() + uint64(len(dp.Name)))
			if err != nil {
				context.Error(err)
				av.Recycle()
				return false
			}
		}
		if !this.sendItem(av) {
			return false
		}
	}

	return mutationOk
}

func (this *SendUpsert) readonly() bool {
	return false
}

func (this *SendUpsert) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

var _UPSERT_POOL = value.NewPairPool(_BATCH_SIZE)
