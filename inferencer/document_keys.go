// Copyright 2021-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included
// in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
// in that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.

package inferencer

import (
	"math"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

func GetAvgDocKeyLen(retriever DocumentRetriever, conn *datastore.ValueConnection,
	context datastore.QueryContext, timeout int32) (int64, errors.Error) {

	start := time.Now()
	count := 0
	totalLength := 0

	for {
		if conn != nil {
			select {
			case <-conn.StopChannel():
				return -1, nil
			default:
			}
		}

		key, doc, err := retriever.GetNextDoc(context)
		if err != nil {
			return -1, err
		}

		if doc == nil {
			// no more docs
			break
		}
		count += 1
		totalLength += len(key)

		if int32(time.Now().Sub(start)/time.Second) > timeout {
			break
		}
	}

	if count > 0 {
		return int64(math.Round(float64(totalLength) / float64(count))), nil
	}

	return -1, nil
}
