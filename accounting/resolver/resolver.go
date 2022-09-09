//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package resolver

import (
	"strings"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/accounting/gometrics"
	"github.com/couchbase/query/accounting/stub"

	"github.com/couchbase/query/errors"
)

func NewAcctstore(uri string) (accounting.AccountingStore, errors.Error) {
	if strings.HasPrefix(uri, "stub:") {
		return accounting_stub.NewAccountingStore(uri)
	}

	if strings.HasPrefix(uri, "gometrics:") {
		return accounting_gm.NewAccountingStore()
	}

	return nil, errors.NewAdminInvalidURL("AccountingStore", uri)
}
