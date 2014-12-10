//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import "github.com/couchbaselabs/query/algebra"

func (this *builder) VisitExecute(stmt *algebra.Execute) (interface{}, error) {

	// stmt contains a JSON representation of a plan.Prepared
	prepared_object := stmt.Prepared()

	// check if there is a plan.Prepared already in the cache
	prepared, err := PreparedCache().GetPrepared(prepared_object)
	if err != nil {
		return nil, err
	}
	if prepared != nil {
		return prepared, nil
	} else {
		prepared = &Prepared{}
	}

	// no cached plan.Prepared => create it
	op_bytes, err := prepared_object.MarshalJSON()
	if err != nil {
		return nil, err
	}
	err = prepared.UnmarshalJSON(op_bytes)
	if err == nil {
		PreparedCache().AddPrepared(prepared)
	}

	return prepared, err
}
