//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"strconv"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func groupKey(item value.Value, keys expression.Expressions, context *opContext) (string, error) {
	kvs := _GROUP_KEY_POOL.GetCapped(len(keys))
	defer _GROUP_KEY_POOL.Put(kvs)

	for i, key := range keys {
		k, e := key.Evaluate(item, context)
		if e != nil {
			return "", e
		}

		if k.Type() != value.MISSING {
			kvs[strconv.Itoa(i)] = k
		}
	}

	bytes, _ := value.NewValue(kvs).MarshalJSON()
	return string(bytes), nil
}

var _GROUP_KEY_POOL = util.NewStringInterfacePool(16)
