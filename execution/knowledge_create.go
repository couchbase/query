//  Copyright 2026-Present Couchbase, Inc.
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
	"github.com/couchbase/query/natural/knowledge"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type CreateKnowledge struct {
	base
	plan *plan.CreateKnowledge
}

func NewCreateKnowledge(plan *plan.CreateKnowledge, context *Context) *CreateKnowledge {
	rv := &CreateKnowledge{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *CreateKnowledge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateKnowledge(this)
}

func (this *CreateKnowledge) Copy() Operator {
	rv := &CreateKnowledge{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CreateKnowledge) PlanOp() plan.Operator {
	return this.plan
}

func (this *CreateKnowledge) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		// semantics restricts this to a shape that's knowable independent of any query context, so
		// a plain (item-independent) Evaluate against the request's arguments is all that's needed
		val, verr := this.plan.Node().Value().Evaluate(nil, &this.operatorCtx)
		if verr != nil {
			context.Error(errors.NewKnowledgeError(errors.E_KNOWLEDGE_INVALID_DATA, verr))
			return
		}

		if this.plan.Node().Bulk() {
			if val.Type() != value.OBJECT {
				context.Error(errors.NewKnowledgeError(errors.E_KNOWLEDGE_INVALID_DATA,
					"value must be an object, not "+val.Type().String()))
				return
			}
			fields := val.Fields()
			entries := make(map[string]string, len(fields))
			for name, v := range fields {
				fv := value.NewValue(v)
				if fv.Type() != value.STRING {
					context.Error(errors.NewKnowledgeError(errors.E_KNOWLEDGE_INVALID_DATA,
						"value for '"+name+"' must be a string, not "+fv.Type().String()))
					return
				}
				sv := fv.ToString()
				if len(sv) > knowledge.MaxHintValueSize {
					context.Error(errors.NewKnowledgeError(errors.E_KNOWLEDGE_LIMIT_EXCEEDED,
						fmt.Sprintf("value for '%s' exceeds the maximum size of %d bytes", name, knowledge.MaxHintValueSize)))
					return
				}
				entries[name] = sv
			}
			if len(entries) == 0 {
				context.Error(errors.NewKnowledgeError(errors.E_KNOWLEDGE_INVALID_DATA, "object must have at least one entry"))
				return
			}
			this.switchPhase(_SERVTIME)
			err := knowledge.CreateKnowledgeBulk(this.plan.Node().Keyspace(), entries, this.plan.Node().Replace())
			this.switchPhase(_EXECTIME)
			if err != nil {
				context.Error(err)
			}
			return
		}

		if val.Type() != value.STRING {
			context.Error(errors.NewKnowledgeError(errors.E_KNOWLEDGE_INVALID_DATA,
				"value must be a string, not "+val.Type().String()))
			return
		}
		sv := val.ToString()
		if len(sv) > knowledge.MaxHintValueSize {
			context.Error(errors.NewKnowledgeError(errors.E_KNOWLEDGE_LIMIT_EXCEEDED,
				fmt.Sprintf("value exceeds the maximum size of %d bytes", knowledge.MaxHintValueSize)))
			return
		}

		name := this.plan.Node().Name()
		if name == "" {
			// implicit name, mirroring PREPARE's opt_name: generated per execution (not once at
			// plan-build time - this.plan can be shared and re-run many times, e.g. a prepared
			// statement re-EXECUTEd or auto-prepare, and every unnamed CREATE KNOWLEDGE should
			// mint a fresh entry rather than every execution after the first colliding on the
			// same baked-in name), so this is a random id rather than PREPARE's content-derived
			// UUIDv5 (which is deliberately deterministic, so repeated PREPAREs of the same text
			// reuse the same cache slot -- the opposite of what we want here).
			var uerr error
			name, uerr = util.UUIDV4()
			if uerr != nil {
				context.Error(errors.NewKnowledgeError(errors.E_KNOWLEDGE, uerr))
				return
			}
		}

		this.switchPhase(_SERVTIME)
		err := knowledge.CreateKnowledge(this.plan.Node().Keyspace(), name, sv, this.plan.Node().Replace())
		this.switchPhase(_EXECTIME)
		if err != nil {
			context.Error(err)
		}
	})
}

func (this *CreateKnowledge) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
