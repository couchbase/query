//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/couchbase/query/errors"
)

/*
   Test the error handling methods
*/

func TestErrors(t *testing.T) {

	// Test some sample errors.
	s_err := HandleError(errors.E_SHELL_CONNECTION_REFUSED, "Random string")

	var b bytes.Buffer
	writetmp := bufio.NewWriter(&b)
	SetOutput(writetmp, true)
	PrintError(s_err)
	writetmp.Flush()

	t.Logf(" Printed Error : %s", b.String())

	s_err = HandleError(errors.E_SHELL_NO_SUCH_PARAM, "-r")

	b.Reset()
	writetmp = bufio.NewWriter(&b)
	SetOutput(writetmp, true)
	PrintError(s_err)
	writetmp.Flush()

	t.Logf(" Printed Error : %s", b.String())
}
