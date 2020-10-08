//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

func (this *builder) VisitPrepare(stmt *algebra.Prepare) (interface{}, error) {
	var prep *plan.Prepared
	var err error

	name := stmt.Name()
	force := stmt.Force()
	text := planCache.GetText(stmt.Text(), stmt.Offset())
	if name == "" {
		var err errors.Error

		name, err = planCache.GetName(text, this.namespace, this.context)
		if err != nil {
			return nil, err
		}
	} else if planCache.IsPredefinedPrepareName(name) {
		return nil, errors.NewPredefinedPreparedNameError(name)
	}

	if !force {
		var gpErr errors.Error

		prep, gpErr = planCache.GetPlan(name, text, this.namespace, this.context)
		if gpErr != nil {
			return nil, gpErr
		}

		if prep != nil {
			json_bytes, err := prep.MarshalJSON()
			if err != nil {
				return nil, err
			}
			val := value.NewValue(json_bytes)
			err = val.SetField("encoded_plan", value.NewValue(prep.EncodedPlan()))
			if err != nil {
				return nil, err
			}
			return plan.NewPrepare(val, prep, false), nil
		}
	}

	dks := this.context.DeltaKeyspaces()
	this.context.SetDeltaKeyspaces(nil)
	prep, err = BuildPrepared(stmt.Statement(), this.datastore, this.systemstore, this.namespace, false, true, this.context)
	this.context.SetDeltaKeyspaces(dks)

	if err != nil {
		return nil, err
	}

	prep.SetName(name)
	prep.SetText(text)
	prep.SetType(stmt.Type())
	prep.SetIndexApiVersion(this.context.IndexApiVersion())
	prep.SetFeatureControls(this.context.FeatureControls())
	prep.SetNamespace(this.namespace)
	prep.SetQueryContext(this.context.QueryContext())
	prep.SetUseFts(this.context.UseFts())
	prep.SetUseCBO(this.context.UseCBO())

	json_bytes, err := prep.MarshalJSON()
	if err != nil {
		return nil, err
	}
	str := prep.BuildEncodedPlan(json_bytes)

	prep.SetEncodedPlan(str)
	val := value.NewValue(json_bytes)
	err = val.SetField("encoded_plan", value.NewValue(str))
	if err != nil {
		return nil, err
	}

	return plan.NewPrepare(val, prep, true), nil
}
