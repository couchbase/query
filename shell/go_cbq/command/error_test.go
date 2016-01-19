//  Copyright (c) 2015-2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
	s_err := HandleError(errors.CONNECTION_REFUSED, "Random string")

	var b bytes.Buffer
	writetmp := bufio.NewWriter(&b)
	SetWriter(writetmp)
	PrintError(s_err)
	writetmp.Flush()

	t.Logf(" Printed Error : %s", b.String())

	s_err = HandleError(errors.NO_SUCH_PARAM, "-r")

	b.Reset()
	writetmp = bufio.NewWriter(&b)
	SetWriter(writetmp)
	PrintError(s_err)
	writetmp.Flush()

	t.Logf(" Printed Error : %s", b.String())
}
