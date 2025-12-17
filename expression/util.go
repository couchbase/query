//  Copyright 2025-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/errors"
)

// Checks if the requested size to be allocated is within the available quota, if quota is set.
// Or within the fallback limit, if quota is not set
func checkSizeWithinLimit(termType string, context Context, elemSize uint64, nelem int, size uint64, fallbackLimit uint64) error {
	max := fallbackLimit
	if qc, ok := context.(QuotaContext); ok && qc.UseRequestQuota() && qc.MemoryQuota() > 0 {
		max = uint64(float64(qc.MemoryQuota()) * (1.0 - qc.CurrentQuotaUsage()))
	}

	if max < size {
		return errors.NewSizeError(termType, elemSize, nelem, size, max)
	}

	return nil
}
