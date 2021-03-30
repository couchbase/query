//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

// A copiedObjectValue is like an objectValue, except it shares its elements with
// at least one other object. Accordingly, when it is recycled, to prevent double recycling of
// maps, the recycling algorithm does not recurse down into the elements.

package value

type copiedObjectValue struct {
	objectValue
}

func (this copiedObjectValue) Track() {
}

func (this copiedObjectValue) Recycle() {
	// Do nothing.
}
