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
)

func (c *Client) CreateBucket(params map[string]interface{}) error {
	var ret interface{}
	err := c.parsePostURLResponseTerse("/pools/default/buckets", params, &ret)
	return err
}

func (c *Client) AlterBucket(name string, params map[string]interface{}) error {
	var ret interface{}
	target := fmt.Sprintf("/pools/default/buckets/%s", name)
	err := c.parsePostURLResponseTerse(target, params, &ret)
	return err
}

func (c *Client) DropBucket(name string) error {
	var ret interface{}
	target := fmt.Sprintf("/pools/default/buckets/%s", name)
	err := c.parseDeleteURLResponseTerse(target, nil, &ret)
	return err
}

func (c *Client) BucketInfo() ([]interface{}, error) {
	ret := make([]interface{}, 0, 1)
	err := c.parseURLResponse("/pools/default/buckets", &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
