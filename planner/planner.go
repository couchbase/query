//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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

	// check if plan already exists for name / text / options combo
	GetPlan(name, text, namespace string, context *PrepareContext) (*plan.Prepared, errors.Error)

	// Predefined prepare name
	IsPredefinedPrepareName(name string) bool
}

var planCache PlanCache

func SetPlanCache(pc PlanCache) {
	planCache = pc
}
