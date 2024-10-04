//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package aus

const (
	AUS_LOG_PREFIX              = "AUS: "
	AUS_DOC_PREFIX              = "aus::"
	AUS_COORDINATION_DOC_PREFIX = "aus_coord::"
)

type MutateOp int

const (
	MOP_NONE MutateOp = iota
	MOP_INSERT
	MOP_UPSERT
	MOP_UPDATE
	MOP_DELETE
)
