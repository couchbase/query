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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
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
func NewPath(namespace, keyspace string) *Path {
	return &Path{
		elements:                []string{namespace, keyspace},
		firstElementIsNamespace: true,
	}
}

func (path *Path) GetNamespaceKeyspace() (namespace string, keyspace string, err error) {
	if len(path.elements) != 2 {
		return "", "", fmt.Errorf("GetKeyspaceNamespace() not supported for paths of length > 2: %v", path.elements)
	}
	return path.elements[0], path.elements[1], nil
}

// This isn't quite right, but it will do for now.
func (path *Path) Namespace() string {
	if path.firstElementIsNamespace {
		return path.elements[0]
	} else {
		return ""
	}
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
		// Do not include empty namespace.
		if path.firstElementIsNamespace && i == 0 {
			continue
		}
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

/*
Represents the keyspace-ref used in DML statements. It
contains three fields namespace, keyspace (bucket) and
an alias (as).
*/
type KeyspaceRef struct {
	path *Path  `json:"path"`
	as   string `json:"as"`
}

/*
The function NewKeyspaceRef returns a pointer to the
KeyspaceRef struct by assigning the input attributes
to the fields of the struct.
*/
func NewKeyspaceRef(namespace, keyspace, as string) *KeyspaceRef {
	return &KeyspaceRef{NewPath(namespace, keyspace), as}
}

/*
Qualify identifiers for the keyspace. It also makes sure that the
keyspace term contains a name or alias.
*/
func (this *KeyspaceRef) Formalize() (f *expression.Formalizer, err error) {
	keyspace := this.Alias()
	if keyspace == "" {
		err = errors.NewNoTermNameError("Keyspace", "semantics.keyspace.reference_requires_name_or_alias")
		return
	}

	f = expression.NewFormalizer(keyspace, nil)
	return
}

/*
Returns the namespace string.
*/
func (this *KeyspaceRef) Namespace() string {
	return this.path.Namespace()

}

/*
Set the default namespace.
*/
func (this *KeyspaceRef) SetDefaultNamespace(namespace string) {
	this.path.SetDefaultNamespace(namespace)
}

/*
Returns the keyspace string.
*/
func (this *KeyspaceRef) Keyspace() string {
	return this.path.Keyspace()
}

/*
Returns the AS alias string.
*/
func (this *KeyspaceRef) As() string {
	return this.as
}

/*
Returns the alias as the keyspace or the as string
based on if as is empty.
*/
func (this *KeyspaceRef) Alias() string {
	if this.as != "" {
		return this.as
	} else {
		return this.path.Alias()
	}
}

/*
Marshals input into byte array.
*/
func (this *KeyspaceRef) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 3)
	r["path"] = this.path
	if this.as != "" {
		r["as"] = this.as
	}

	return json.Marshal(r)
}

/*
Returns the full keyspace name, including the namespace.
*/
func (this *KeyspaceRef) FullName() string {
	return this.path.String()
}
