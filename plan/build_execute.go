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
	"github.com/couchbaselabs/query/errors"
)

func (this *builder) VisitExecute(stmt *algebra.Execute) (interface{}, error) {

	// stmt contains a JSON representation of a plan.Prepared
	prepared_object := stmt.Prepared().Value()

	sig, ok := prepared_object.Field("signature")

	if !ok {
		return nil, errors.NewError(nil, "prepared is missing signature")
	}

	operator, ok := prepared_object.Field("operator")

	if !ok {
		return nil, errors.NewError(nil, "prepared is missing operator")
	}

	op_bytes, err := operator.MarshalJSON()

	if err != nil {
		return nil, err
	}

	var prepared Prepared

	err = prepared.UnmarshalJSON(op_bytes)

	prepared.signature = sig

	return &prepared, err
}
