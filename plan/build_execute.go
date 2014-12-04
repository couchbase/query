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
	"encoding/json"

	"github.com/couchbaselabs/query/algebra"
)

func (this *builder) VisitExecute(stmt *algebra.Execute) (interface{}, error) {
	var prepared Prepared

	// stmt contains a JSON representation of a plan.Prepared
	prepared_object := stmt.Prepared()

	// convert the JSON representation to a []bytes
	prepared_bytes, err := prepared_object.Value().MarshalJSON()

	if err != nil {
		return nil, err
	}

	// convert the []bytes to an actual plan.Prepared
	err = json.Unmarshal(prepared_bytes, &prepared)
	if err != nil {
		return nil, err
	}

	return prepared, nil
}
