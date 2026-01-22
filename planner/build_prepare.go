//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
	"github.com/couchbase/query/util"
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
			return plan.NewQueryPlan(plan.NewPrepare(val, prep, false)), nil
		}
	}

	dks := this.context.DeltaKeyspaces()
	this.context.SetDeltaKeyspaces(nil)
	prep, err, _ = BuildPrepared(stmt.Statement(), this.datastore, this.systemstore, this.namespace,
		false, true, this.context)
	this.context.SetDeltaKeyspaces(dks)

	if err != nil {
		return nil, err
	}

	persist := stmt.Save()
	planStabilityMode := settings.GetPlanStabilityMode()

	prep.SetName(name)
	prep.SetText(text)
	prep.SetType(stmt.Type())
	prep.SetIndexApiVersion(this.context.IndexApiVersion())
	prep.SetFeatureControls(this.context.FeatureControls())
	prep.SetNamespace(this.namespace)
	prep.SetQueryContext(this.context.QueryContext())
	prep.SetUseFts(this.context.UseFts())
	prep.SetUseCBO(this.context.UseCBO())
	prep.SetUserAgent(this.context.dsContext.UserAgent())
	prep.SetUsers(this.context.dsContext.Users())
	prep.SetRemoteAddr(this.context.dsContext.RemoteAddr())
	prep.SetPreparedTime(util.Now().ToTime())
	prep.SetPersist(persist)
	prep.SetAdHoc(!persist && planStabilityMode == settings.PS_MODE_AD_HOC)
	prep.KeyspaceReferences()

	json_bytes, err := prep.MarshalJSON()
	if err != nil {
		return nil, err
	}
	str, err := prep.BuildEncodedPlan()
	if err != nil {
		return nil, err
	}

	prep.SetEncodedPlan(str)

	if persist || (planStabilityMode == settings.PS_MODE_PREPARED_ONLY || planStabilityMode == settings.PS_MODE_AD_HOC) {
		// check and create (if not exists) QUERY_METADATA bucket
		hasMetadata, err1 := hasQueryMetadata(true, this.context.RequestId(), true)
		if err1 == nil && !hasMetadata {
			err1 = errors.NewMissingQueryMetadataError("SAVE option of PREPARE or Plan Stability")
		}
		if err1 != nil {
			return nil, err1
		}
	}

	val := value.NewValue(json_bytes)
	err = val.SetField("encoded_plan", value.NewValue(str))
	if err != nil {
		return nil, err
	}

	return plan.NewQueryPlan(plan.NewPrepare(val, prep, true)), nil
}
