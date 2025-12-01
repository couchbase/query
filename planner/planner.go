//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
)

// plan, planner and prepareds naturally have circular references, which golang abhors
// we define an interface that prepareds can implement so that we can fix things at runtime

type PlanCache interface {

	// return the statement text to be cached
	GetText(text string, offset int) string

	// return the expected statement name generated from the text and options
	GetName(text, namespace string, context *PrepareContext) (string, errors.Error)

	// return the encoded name generated from name and query context
	EncodeName(name, queryContext string) string

	// check if plan already exists for name / text / options combo
	GetPlan(name, text, namespace string, context *PrepareContext) (*plan.Prepared, errors.Error)

	// Predefined prepare name
	IsPredefinedPrepareName(name string) bool
}

var planCache PlanCache

func SetPlanCache(pc PlanCache) {
	planCache = pc
}
