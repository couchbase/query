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
	PRIV_READ          Privilege = 1
	PRIV_WRITE         Privilege = 2
	PRIV_DDL           Privilege = 3
	PRIV_SYSTEM_READ   Privilege = 4 // Access to tables in the system namespace, such as system:keyspaces.
	PRIV_SECURITY_READ Privilege = 5 // Access to system:user_info, specifically.
)

/*
Type Privileges maps string of the form "namespace:keyspace" to
privileges.
*/
type Privileges map[string]Privilege

func NewPrivileges() Privileges {
	return make(Privileges, 16)
}

func (this Privileges) Add(other Privileges) {
	for k, p := range other {
		tp, ok := this[k]
		if !ok || tp < p {
			this[k] = p
		}
	}
}

/*
Type Credentials maps users to passwords.
*/
type Credentials map[string]string

/*
Type AuthenticatedUsers is a list of users whose credentials checked out.
*/
type AuthenticatedUsers []string
