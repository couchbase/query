//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

import (
	"bytes"
	"fmt"
)

type User struct {
	Name     string `json:"name"`
	Id       string `json:"id"`
	Domain   string `json:"domain"`
	Roles    []Role `json:"roles"`
	Password string
	Groups   []string `json:"groups"`
}

type Role struct {
	Role           string
	BucketName     string `json:"bucket_name"`
	ScopeName      string `json:"scope_name"`
	CollectionName string `json:"collection_name"`
}

type Group struct {
	Id    string `json:"id"`
	Desc  string `json:"description"`
	Roles []Role `json:"roles"`
}

// Sample:
// {"role":"admin","name":"Admin","desc":"Can manage ALL cluster features including security.","ce":true}
// {"role":"query_select","bucket_name":"*","name":"Query Select","desc":"Can execute SELECT statement on bucket to retrieve data"}
type RoleDescription struct {
	Role           string
	Name           string
	Desc           string
	Ce             bool
	BucketName     string `json:"bucket_name"`
	ScopeName      string `json:"scope_name"`
	CollectionName string `json:"collection_name"`
}

// Return user-role data, as parsed JSON.
// Sample:
//
//	[{"id":"ivanivanov","name":"Ivan Ivanov","roles":[{"role":"cluster_admin"},{"bucket_name":"default","role":"bucket_admin"}]},
//	 {"id":"petrpetrov","name":"Petr Petrov","roles":[{"role":"replication_admin"}]}]
func (c *Client) GetUserRoles() ([]interface{}, error) {
	ret := make([]interface{}, 0, 1)
	err := c.parseURLResponse("/settings/rbac/users", &ret)
	if err != nil {
		return nil, err
	}

	// Get the configured administrator.
	// Expected result: {"port":8091,"username":"Administrator"}
	adminInfo := make(map[string]interface{}, 2)
	err = c.parseURLResponse("/settings/web", &adminInfo)
	if err != nil {
		return nil, err
	}

	// Create a special entry for the configured administrator.
	adminResult := map[string]interface{}{
		"name":   adminInfo["username"],
		"id":     adminInfo["username"],
		"domain": "builtin",
		"roles": []interface{}{
			map[string]interface{}{
				"role": "admin",
			},
		},
	}

	// Add the configured administrator to the list of results.
	ret = append(ret, adminResult)

	return ret, nil
}

func (c *Client) GetUserInfoAll() ([]User, error) {
	ret := make([]User, 0, 16)
	err := c.parseURLResponse("/settings/rbac/users", &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func rolesToParamFormat(roles []Role) string {
	var buffer bytes.Buffer
	for i, role := range roles {
		if i > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(role.Role)
		if role.BucketName != "" {
			buffer.WriteString("[")
			buffer.WriteString(role.BucketName)
			buffer.WriteString("]")
		}
	}
	return buffer.String()
}

func (c *Client) PutUserInfo(u *User) error {
	params := make(map[string]interface{})
	if u.Name != string([]byte{0}) {
		params["name"] = u.Name
	}
	if len(u.Roles) > 0 {
		params["roles"] = rolesToParamFormat(u.Roles)
	}
	if u.Password != string([]byte{0}) {
		params["password"] = u.Password
	}
	if len(u.Groups) > 0 {
		first := false
		var s string
		for i := 0; i < len(u.Groups); i++ {
			if u.Groups[i] != "" {
				if !first {
					s += ","
				}
				s += u.Groups[i]
				first = false
			}
		}
		params["groups"] = s
	}
	var target string
	switch u.Domain {
	case "external":
		target = "/settings/rbac/users/external/" + u.Id
	case "local":
		target = "/settings/rbac/users/local/" + u.Id
	default:
		return fmt.Errorf("Unknown user type: %s", u.Domain)
	}
	var ret string // PUT returns an empty string. We ignore it.
	err := c.parsePutURLResponseTerse(target, params, &ret)
	return err
}

func (c *Client) GetRolesAll() ([]RoleDescription, error) {
	ret := make([]RoleDescription, 0, 32)
	err := c.parseURLResponse("/settings/rbac/roles", &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Client) DeleteUser(u *User) error {
	var target string
	switch u.Domain {
	case "external":
		target = "/settings/rbac/users/external/" + u.Id
	case "local":
		target = "/settings/rbac/users/local/" + u.Id
	default:
		return fmt.Errorf("Unknown user type: %s", u.Domain)
	}
	var ret string // PUT returns an empty string. We ignore it.
	err := c.parseDeleteURLResponseTerse(target, nil, &ret)
	return err
}

func (c *Client) GetUserInfo(u *User) error {
	var target string
	switch u.Domain {
	case "external":
		target = "/settings/rbac/users/external/" + u.Id
	case "local":
		target = "/settings/rbac/users/local/" + u.Id
	default:
		return fmt.Errorf("Unknown user type: %s", u.Domain)
	}
	err := c.parseURLResponse(target, u)
	return err
}

func (c *Client) GetGroupInfo(g *Group) error {
	target := fmt.Sprintf("/settings/rbac/groups/%s", g.Id)
	err := c.parseURLResponse(target, g)
	return err
}

func (c *Client) PutGroupInfo(g *Group) error {
	params := make(map[string]interface{})
	if g.Desc != string([]byte{0}) {
		params["description"] = g.Desc
	}
	var s string
	first := true
	for i := 0; i < len(g.Roles); i++ {
		if !first {
			s += ","
		}
		s += g.Roles[i].Role
		if g.Roles[i].BucketName != "" {
			s += "[" + g.Roles[i].BucketName
			if g.Roles[i].ScopeName != "" && g.Roles[i].ScopeName != "*" {
				s += ":" + g.Roles[i].ScopeName
				if g.Roles[i].CollectionName != "" && g.Roles[i].CollectionName != "*" {
					s += ":" + g.Roles[i].CollectionName
				}
			}
			s += "]"
		}
		first = false
	}
	params["roles"] = s
	target := fmt.Sprintf("/settings/rbac/groups/%s", g.Id)
	var ret string // PUT returns an empty string. We ignore it.
	err := c.parsePutURLResponseTerse(target, params, &ret)
	return err
}

func (c *Client) DeleteGroup(g *Group) error {
	target := fmt.Sprintf("/settings/rbac/groups/%s", g.Id)
	var ret string // PUT returns an empty string. We ignore it.
	err := c.parseDeleteURLResponseTerse(target, nil, &ret)
	return err
}

// The subtle different between GroupInfo() and GetGroupInfoAll() is the returned maps from GroupInfo() may be missing fields
// which are always present in the Group type.
func (c *Client) GroupInfo() ([]interface{}, error) {
	ret := make([]interface{}, 0, 1)
	err := c.parseURLResponse("/settings/rbac/groups", &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Client) GetGroupInfoAll() ([]Group, error) {
	ret := make([]Group, 0, 16)
	err := c.parseURLResponse("/settings/rbac/groups", &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
