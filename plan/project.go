//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"github.com/couchbaselabs/query/algebra"
)

type InitialProject struct {
	readonly
	projection *algebra.Projection
	terms      ProjectTerms
}

func NewInitialProject(projection *algebra.Projection) *InitialProject {
	results := projection.Terms()
	terms := make(ProjectTerms, len(results))

	for i, res := range results {
		terms[i] = &ProjectTerm{
			result: res,
		}
	}

	return &InitialProject{
		projection: projection,
		terms:      terms,
	}
}

func (this *InitialProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialProject(this)
}

func (this *InitialProject) Projection() *algebra.Projection {
	return this.projection
}

func (this *InitialProject) Terms() ProjectTerms {
	return this.terms
}

type FinalProject struct {
	readonly
}

func NewFinalProject() *FinalProject {
	return &FinalProject{}
}

func (this *FinalProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalProject(this)
}

type ProjectTerms []*ProjectTerm

type ProjectTerm struct {
	result *algebra.ResultTerm
}

func (this *ProjectTerm) Result() *algebra.ResultTerm {
	return this.result
}
