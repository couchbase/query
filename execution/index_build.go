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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type BuildIndexes struct {
	base
	plan *plan.BuildIndexes
}

func NewBuildIndexes(plan *plan.BuildIndexes, context *Context) *BuildIndexes {
	rv := &BuildIndexes{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *BuildIndexes) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBuildIndexes(this)
}

func (this *BuildIndexes) Copy() Operator {
	rv := &BuildIndexes{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *BuildIndexes) PlanOp() plan.Operator {
	return this.plan
}

func (this *BuildIndexes) RunOnce(context *Context, parent value.Value) {
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

		// Actually build indexes
		this.switchPhase(_SERVTIME)
		node := this.plan.Node()

		indexer, err := this.plan.Keyspace().Indexer(node.Using())
		if err != nil {
			context.Error(err)
			return
		}

		err = indexer.Refresh()
		if err != nil {
			context.Error(err)
			return
		}

		names, err1 := getIndexNames(context, parent, node.Names(), "build_index")
		if err1 != nil {
			context.Error(err1)
			return
		}

		idxNames := make([]string, 0, len(names))
		var index datastore.Index
		for _, name := range names {
			index, err = indexer.IndexByName(name)
			if err != nil {
				context.Error(errors.NewIndexNotFoundError(name, "execution.build_index.index_by_name", err))
				// skip this index in BUILD command, but continue with other indexes
				continue
			}
			state, _, err1 := index.State()
			if err1 != nil {
				// skip this index in BUILD command, but continue with other indexes
				context.Error(err1)
				continue
			}
			if state != datastore.ONLINE {
				idxNames = append(idxNames, name)
			}
		}

		if len(idxNames) == 0 {
			return
		}

		err = indexer.BuildIndexes(context.RequestId(), idxNames...)
		if err != nil {
			context.Error(err)
			return
		}

		if node.Using() == datastore.GSI || node.Using() == datastore.DEFAULT {
			err = updateStats(idxNames, "build_index", this.plan.Keyspace(), context)
			if err != nil {
				context.Error(err)
				return
			}
		}
	})
}

func getIndexNames(context *Context, av value.Value, exprs expression.Expressions, err_key string) ([]string, errors.Error) {
	ikey := "execution." + err_key + ".get_index_name"
	rv := make([]string, 0, len(exprs))
	for _, expr := range exprs {
		val, err := expr.Evaluate(av, context)
		if err != nil {
			return nil, errors.NewEvaluationError(err, "index name expression")
		}

		actual := val.Actual()

		if actuals, ok := actual.([]interface{}); ok {
			for _, actual := range actuals {
				ac := value.NewValue(actual).Actual()
				if s, ok := ac.(string); ok {
					rv = append(rv, s)
				} else {
					return nil, errors.NewInvalidIndexNameError(ac, ikey)
				}
			}
		} else if s, ok := actual.(string); ok {
			rv = append(rv, s)
		} else {
			return nil, errors.NewInvalidIndexNameError(actual, ikey)
		}
	}
	return rv, nil

}

func (this *BuildIndexes) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
