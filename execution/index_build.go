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

	newRedirectBase(&rv.base)
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

		for _, name := range names {
			if _, err = indexer.IndexByName(name); err != nil {
				context.Error(errors.NewIndexNotFoundError(name, "execution.build_index.index_by_name", err))
				return
			}
		}

		err = indexer.BuildIndexes(context.RequestId(), names...)
		if err != nil {
			context.Error(err)
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
