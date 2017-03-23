//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package auth

type Privilege int

const (
	PRIV_READ                  Privilege = 1
	PRIV_WRITE                 Privilege = 2
	PRIV_SYSTEM_READ           Privilege = 4  // Access to tables in the system namespace, such as system:keyspaces.
	PRIV_SECURITY_READ         Privilege = 5  // Reading user information.
	PRIV_SECURITY_WRITE        Privilege = 6  // Updating user information.
	PRIV_QUERY_SELECT          Privilege = 7  // Ability to run SELECT statements.
	PRIV_QUERY_UPDATE          Privilege = 8  // Ability to run UPDATE statements.
	PRIV_QUERY_INSERT          Privilege = 9  // Ability to run INSERT statements.
	PRIV_QUERY_DELETE          Privilege = 10 // Ability to run DELETE statements.
	PRIV_QUERY_BUILD_INDEX     Privilege = 11 // Ability to run BUILD INDEX statements.
	PRIV_QUERY_CREATE_INDEX    Privilege = 12 // Ability to run CREATE INDEX statements.
	PRIV_QUERY_ALTER_INDEX     Privilege = 13 // Ability to run ALTER INDEX statements.
	PRIV_QUERY_DROP_INDEX      Privilege = 14 // Ability to run DROP INDEX statements.
	PRIV_QUERY_LIST_INDEX      Privilege = 15 // Ability to list indexes of a keyspace.
	PRIV_QUERY_EXTERNAL_ACCESS Privilege = 16 // Ability to access the web from a N1QL query.
)

func IsStatementTypePrivilege(priv Privilege) bool {
	return priv == PRIV_QUERY_SELECT || priv == PRIV_QUERY_UPDATE ||
		priv == PRIV_QUERY_INSERT || priv == PRIV_QUERY_DELETE
}

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

var NO_PRIVILEGES = NewPrivileges()

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

func (this *Privileges) ForEach(f func(PrivilegePair)) {
	for _, pair := range this.List {
		f(pair)
	}
}

func (this *Privileges) AddPair(pp PrivilegePair) {
	for _, pair := range this.List {
		if pair.Target == pp.Target && pair.Priv == pp.Priv {
			// already present
			return
		}
	}
	this.List = append(this.List, pp)
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
