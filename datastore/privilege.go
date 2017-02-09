//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package datastore

import ()

type Privilege int

const (
	PRIV_READ           Privilege = 1
	PRIV_WRITE          Privilege = 2
	PRIV_DDL            Privilege = 3
	PRIV_SYSTEM_READ    Privilege = 4  // Access to tables in the system namespace, such as system:keyspaces.
	PRIV_SECURITY_READ  Privilege = 5  // Reading user information.
	PRIV_SECURITY_WRITE Privilege = 6  // Updating user information.
	PRIV_QUERY_SELECT   Privilege = 7  // Ability to run SELECT statements.
	PRIV_QUERY_UPDATE   Privilege = 8  // Ability to run UPDATE statements.
	PRIV_QUERY_INSERT   Privilege = 9  // Ability to run INSERT statements.
	PRIV_QUERY_DELETE   Privilege = 10 // Ability to run DELETE statements.
)

type PrivilegePair struct {
	Target string // For what resource is the privilege requested. Typically a string of
	// the form "namespace:keyspace". Could be blank, for system-wide
	// privileges
	Priv Privilege // The level of privilege requested. Note there could be multiple
	// privileges against the same target.
}

// A set of permissions required, typically to run a specific query.
type Privileges struct {
	List []PrivilegePair
}

func NewPrivileges() *Privileges {
	return &Privileges{List: make([]PrivilegePair, 0, 16)}
}

func (this *Privileges) AddAll(other *Privileges) {
	if other == nil {
		return
	}
	for _, pair := range other.List {
		this.Add(pair.Target, pair.Priv)
	}
}

func (this *Privileges) Add(target string, priv Privilege) {
	for _, pair := range this.List {
		if pair.Target == target && pair.Priv == priv {
			// already present
			return
		}
	}
	this.List = append(this.List, PrivilegePair{Target: target, Priv: priv})
}

/*
Type Credentials maps users to passwords.
*/
type Credentials map[string]string

/*
Type AuthenticatedUsers is a list of users whose credentials checked out.
*/
type AuthenticatedUsers []string
