//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package semantics

import (
	"github.com/couchbase/query/algebra"
)

func (this *SemChecker) VisitCreateFunction(stmt *algebra.CreateFunction) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitDropFunction(stmt *algebra.DropFunction) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitExecuteFunction(stmt *algebra.ExecuteFunction) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}
