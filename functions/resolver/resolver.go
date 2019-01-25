//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package resolver

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/inline"
)

func MakeBody(name functions.FunctionName, bytes []byte) (functions.FunctionBody, errors.Error) {
	var language_type struct {
		Language string `json:"#language"`
	}

	err := json.Unmarshal(bytes, &language_type)
	if err != nil {
		return nil, errors.NewFunctionEncodingError("decode body", name.Name(), err)
	}
	switch language_type.Language {
	case "inline":
		return inline.MakeInline(name, bytes)
	default:
		return nil, errors.NewFunctionEncodingError("decode body", name.Name(), fmt.Errorf("unknown language %v", language_type.Language))
	}
}
