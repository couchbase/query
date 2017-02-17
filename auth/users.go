//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package auth

// These structures are generic representations of users and their roles.
// Very similar structures exist in go-couchbase, but to keep open the
// possibility of connecting to other back ends, the query engine
// uses its own representation.
type User struct {
	Name  string
	Id    string
	Roles []Role
}

type Role struct {
	Name     string
	Keyspace string
}
