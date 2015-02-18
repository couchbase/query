//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

func groupKey(item value.Value, keys expression.Expressions, context *Context) (string, error) {
	kvs := make(map[string]interface{}, len(keys))
	for i, key := range keys {
		k, e := key.Evaluate(item, context)
		if e != nil {
			return "", e
		}

		if k.Type() != value.MISSING {
			kvs[string(i)] = k
		}
	}

	bytes, _ := value.NewValue(kvs).MarshalJSON()
	return string(bytes), nil
}
