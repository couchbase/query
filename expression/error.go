//  Copyright 2023-Present Couchbase, Inc.
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

type ErrorContext struct {
	line   int
	column int
}

func (this *ErrorContext) String() string {
	if this.line == 0 {
		return ""
	}
	return errors.NewErrorContext(this.line, this.column).Error()
}

func (this *ErrorContext) Set(line int, column int) {
	this.line = line
	this.column = column
}

func (this *ErrorContext) Get() (int, int) {
	return this.line, this.column
}
