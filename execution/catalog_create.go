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
	"strings"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type CreateCatalog struct {
	base
	plan *plan.CreateCatalog
}

func NewCreateCatalog(plan *plan.CreateCatalog, context *Context) *CreateCatalog {
	rv := &CreateCatalog{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *CreateCatalog) Accept(visitor Visitor) (any, error) {
	return visitor.VisitCreateCatalog(this)
}

func (this *CreateCatalog) Copy() Operator {
	rv := &CreateCatalog{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CreateCatalog) PlanOp() plan.Operator {
	return this.plan
}

func (this *CreateCatalog) RunOnce(context *Context, parent value.Value) {
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

		this.switchPhase(_SERVTIME)
		node := this.plan.Node()

		cbDs, ok := context.datastore.(datastore.CouchbaseDatastore)
		if !ok {
			context.Error(errors.NewOtherNotImplementedError(nil, strings.ReplaceAll(node.Type(), "_", " ")))
			return
		}

		err := cbDs.CreateCatalog(context, node.Name(), node.CatalogType(), node.Source(), node.Credential(), node.With())
		if err != nil {
			ae := errors.IsExistsError("Catalog", err)
			if !ae || node.FailIfExists() {
				if ae {
					err = errors.NewCbCatalogExistsError(node.Name())
				}
				context.Error(err)
			}
		}
	})
}

func (this *CreateCatalog) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
