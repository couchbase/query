//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package algebra

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type Context interface {
	expression.Context
	Datastore() datastore.Datastore
	NamedArg(name string) (value.Value, bool)
	PositionalArg(position int) (value.Value, bool)
	EvaluateSubquery(query *Select, parent value.Value) (value.Value, error)
}
