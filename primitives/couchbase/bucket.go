//  Copyright 2023-Present Couchbase, Inc.
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

func (c *Client) CreateBucket(cred cbauth.Creds, params map[string]interface{}) error {
	var ret interface{}
	err := c.parsePostURLResponseTerse("/pools/default/buckets", cred, params, &ret)
	return err
}

func (c *Client) AlterBucket(cred cbauth.Creds, name string, params map[string]interface{}) error {
	var ret interface{}
	target := fmt.Sprintf("/pools/default/buckets/%s", name)
	err := c.parsePostURLResponseTerse(target, cred, params, &ret)
	return err
}

func (c *Client) DropBucket(cred cbauth.Creds, name string) error {
	var ret interface{}
	target := fmt.Sprintf("/pools/default/buckets/%s", name)
	err := c.parseDeleteURLResponseTerse(target, cred, nil, &ret)
	return err
}

func (c *Client) BucketInfo(cred cbauth.Creds) ([]interface{}, error) {
	ret := make([]interface{}, 0, 1)
	err := c.parseURLResponse("/pools/default/buckets", cred, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
