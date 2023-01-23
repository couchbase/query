//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

// Sample data:
// {"disabled":["12333", "22244"],"uid":"132492431","auditdEnabled":true,
//
//	"disabledUsers":[{"name":"bill","domain":"local"},{"name":"bob","domain":"local"}],
//	"logPath":"/Users/johanlarson/Library/Application Support/Couchbase/var/lib/couchbase/logs",
//	"rotateInterval":86400,"rotateSize":20971520}
type AuditSpec struct {
	Disabled       []uint32    `json:"disabled"`
	Uid            string      `json:"uid"`
	AuditdEnabled  bool        `json:"auditdEnabled`
	DisabledUsers  []AuditUser `json:"disabledUsers"`
	LogPath        string      `json:"logPath"`
	RotateInterval int64       `json:"rotateInterval"`
	RotateSize     int64       `json:"rotateSize"`
}

type AuditUser struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

func (c *Client) GetAuditSpec() (*AuditSpec, error) {
	ret := &AuditSpec{}
	err := c.parseURLResponse("/settings/audit", ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
