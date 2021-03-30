//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

import (
	"github.com/couchbase/query/value"
)

/*
Expression path. The type Paths is a slice of Path.
*/
type Paths []Path

/*
Path is of type interface that inherits Expression.
It also contains 2 methods Set and Unset. They take
as input an item, value or type Value, and a context
and return a boolean value that depicts if the path
was set or unset.
*/
type Path interface {
	Expression
	Set(item, val value.Value, context Context) bool
	Unset(item value.Value, context Context) bool
}
