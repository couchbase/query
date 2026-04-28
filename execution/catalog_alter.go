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

type AlterCatalog struct {
	base
	plan *plan.AlterCatalog
}

func NewAlterCatalog(plan *plan.AlterCatalog, context *Context) *AlterCatalog {
	rv := &AlterCatalog{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *AlterCatalog) Accept(visitor Visitor) (any, error) {
	return visitor.VisitAlterCatalog(this)
}

func (this *AlterCatalog) Copy() Operator {
	rv := &AlterCatalog{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *AlterCatalog) PlanOp() plan.Operator {
	return this.plan
}

func (this *AlterCatalog) RunOnce(context *Context, parent value.Value) {
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

		err := cbDs.AlterCatalog(context, node.Name(), node.With())
		if err != nil {
			if errors.IsNotFoundError("Catalog", err) {
				err = errors.NewCbCatalogNotFoundError(err, node.Name())
			}
			context.Error(err)
		}
	})
}

func (this *AlterCatalog) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
