//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"strings"
)

// A keyspace path. Supported forms:
//    customers
//    system:prepareds
//    default:prepareds
//    default:myBucket.myScope.myCollection
type Path struct {
	elements                []string `json:"elements"`
	firstElementIsNamespace bool     `json:"firstElementIsNamespace"`
}

// Create a path from a namespace:keyspace combination.
func NewPathShort(namespace, keyspace string) *Path {
	return &Path{
		elements:                []string{namespace, keyspace},
		firstElementIsNamespace: true,
	}
}

func NewPathLong(namespace, bucket, scope, keyspace string) *Path {
	return &Path{
		elements:                []string{namespace, bucket, scope, keyspace},
		firstElementIsNamespace: true,
	}
}

// This isn't quite right, but it will do for now.
func (path *Path) Namespace() string {
	if path.firstElementIsNamespace {
		return path.elements[0]
	} else {
		return ""
	}
}

func (path *Path) Bucket() string {
	if len(path.elements) == 4 {
		return path.elements[1]
	} else {
		return ""
	}
}

func (path *Path) Scope() string {
	if len(path.elements) == 4 {
		return path.elements[2]
	} else {
		return ""
	}
}

func (path *Path) IsCollection() bool {
	return len(path.elements) == 4
}

// Also, not quite right. But temporary.
func (path *Path) Keyspace() string {
	return path.elements[len(path.elements)-1]
}

func (path *Path) SetDefaultNamespace(namespace string) {
	if path.firstElementIsNamespace && path.elements[0] == "" {
		path.elements[0] = namespace
	}
}

func (path *Path) Alias() string {
	return path.elements[len(path.elements)-1]
}

func (path *Path) String() string {
	acc := ""
	lastIndex := len(path.elements) - 1
	for i, s := range path.elements {
		// Wrap any element that contains "." in back-ticks.
		if strings.Contains(s, ".") {
			acc += "`"
			acc += s
			acc += "`"
		} else {
			acc += s
		}
		// Add a separator. ":" after a namespace, else "."
		if i < lastIndex {
			// Need a separator.
			if i == 0 && path.firstElementIsNamespace {
				acc += ":"
			} else {
				acc += "."
			}
		}
	}
	return acc
}
