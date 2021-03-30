//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package main

// #include <stdlib.h>
import "C"
import "errors"

func promptPassword(prompt string) ([]byte, error) {
	password := C.getpass(C.CString(prompt))
	if password == nil {
		return nil, errors.New("getpass: failed to get password")
	}
	return []byte(C.GoString(password)), nil
}
