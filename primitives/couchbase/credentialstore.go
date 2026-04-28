//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

import (
	"fmt"

	"github.com/couchbase/cbauth"
)

const _CREDENTIAL_STORE_PATH = "/settings/credentials"

func (c *Client) CreateCredentialStore(cred cbauth.Creds, name string, params map[string]any) error {
	target := fmt.Sprintf("%s/%s", _CREDENTIAL_STORE_PATH, uriAdj(name))
	return c.parsePostURLResponseJSON(target, cred, params, nil)
}

func (c *Client) AlterCredentialStore(cred cbauth.Creds, name string, params map[string]any) error {
	target := fmt.Sprintf("%s/%s", _CREDENTIAL_STORE_PATH, uriAdj(name))
	return c.parsePostURLResponseJSON(target, cred, params, nil)
}

func (c *Client) DropCredentialStore(cred cbauth.Creds, name string) error {
	target := fmt.Sprintf("%s/%s", _CREDENTIAL_STORE_PATH, uriAdj(name))
	return c.parseDeleteURLResponseTerse(target, cred, nil, nil)
}

func (c *Client) GetCredentialStore(cred cbauth.Creds, name string) (rv map[string]any, err error) {
	target := fmt.Sprintf("%s/%s", _CREDENTIAL_STORE_PATH, uriAdj(name))
	err = c.parseURLResponse(target, cred, &rv)
	return
}

func (c *Client) ListCredentialStores(cred cbauth.Creds) ([]any, error) {
	rv := make([]any, 0, 1)
	err := c.parseURLResponse(_CREDENTIAL_STORE_PATH, cred, &rv)
	if err != nil {
		return nil, err
	}
	return rv, nil
}
