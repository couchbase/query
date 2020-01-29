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

	"github.com/couchbase/query/errors"
)

// A keyspace path. Supported forms:
//    customers (needs queryContext)
//    system:prepareds
//    default:customers
//    default:myBucket.myScope.myCollection
type Path struct {
	elements []string `json:"elements"`
}

// Create a path from a namespace:keyspace combination.
func NewPathShort(namespace, keyspace string) *Path {
	return &Path{
		elements: []string{namespace, keyspace},
	}
}

// Create a path from a namespace:bucket.scope.keyspace combination
func NewPathLong(namespace, bucket, scope, keyspace string) *Path {
	return &Path{
		elements: []string{namespace, bucket, scope, keyspace},
	}
}

// Create a path from three possible combinations:
// namespace:bucket
// namespace:keyspace (for backwards compatibility)
// namespace:bucket.scope.keyspace
// used for plan unmarshalling
func NewPathShortOrLong(namespace, bucket, scope, keyspace string) *Path {
	if keyspace == "" {
		return &Path{
			elements: []string{namespace, bucket},
		}
	} else if bucket == "" {
		return &Path{
			elements: []string{namespace, keyspace},
		}
	}
	return &Path{
		elements: []string{namespace, bucket, scope, keyspace},
	}
}

// Creates a full path from a single identifier and a query context
// the query context is expected to be of the form
// blank (namespace is either specified by the namespace parameter or set later)
// :bucket.scope (namespace is either specified by the namespace parameter or set later)
// namespace: (produces a short path)
// namespace:bucket.scope (produces a long path)
func NewPathWithContext(keyspace, namespace, queryContext string) *Path {
	if queryContext == "" {
		return &Path{

			// FIXME this has to be amended once collection privileges are defined
			// ideally we want the path fully qualified here, as we have been missing
			// several SetDefaultNamespace calls in planner
			// elements: []string{namespace, keyspace},
			elements: []string{"", keyspace},
		}
	}

	elems := parseQueryContext(queryContext)

	// FIXME ditto
	//	if elems[0] == "" {
	//		elems[0] = namespace
	//	}
	return &Path{
		elements: append(elems, keyspace),
	}
}

func (path *Path) Namespace() string {
	return path.elements[0]
}

// the next three methods are currently unused but left for completeness
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

// the keyspace is always the last element of the path
func (path *Path) Keyspace() string {
	return path.elements[len(path.elements)-1]
}

func (path *Path) Parts() []string {
	return path.elements
}

// FIXME ideally this should go
func (path *Path) SetDefaultNamespace(namespace string) {
	if path.elements[0] == "" {
		path.elements[0] = namespace
	}
}

func (path *Path) Alias() string {
	return path.elements[len(path.elements)-1]
}

func (path *Path) SimpleString() string {
	forceBackticks := false
	return path.string(forceBackticks)
}

func (path *Path) ProtectedString() string {
	forceBackticks := true
	return path.string(forceBackticks)
}

func (path *Path) string(forceBackticks bool) string {
	acc := ""
	lastIndex := len(path.elements) - 1
	for i, s := range path.elements {
		// The first element, i.e. the namespace, may be an empty string.
		// That means we can omit it, and the separator after it.
		if i == 0 && s == "" {
			continue
		}
		// Wrap any element that contains "." in back-ticks.
		if forceBackticks || strings.Contains(s, ".") {
			acc += "`"
			acc += s
			acc += "`"
		} else {
			acc += s
		}
		// Add a separator. ":" after a namespace, else "."
		if i < lastIndex {
			// Need a separator.
			if i == 0 {
				acc += ":"
			} else {
				acc += "."
			}
		}
	}
	return acc
}

// this is used for operator marshalling
// by the time it's called, the namespace is expected to be set
func (this *Path) marshalKeyspace(m map[string]interface{}) {
	l := len(this.elements)
	m["namespace"] = this.elements[0]
	if l > 2 {
		m["bucket"] = this.elements[1]
		m["scope"] = this.elements[2]
	}
	m["keyspace"] = this.elements[l-1]
}

// the queryContext is expected to be syntactically correct as per function below
// no checks are made, and undesired behaviour will ensue if it isn't
func parseQueryContext(queryContext string) []string {
	if queryContext == "" || queryContext == ":" {
		return []string{""}
	}
	elements := []string{}
	hasNamespace := false
	start := 0
	for i, c := range queryContext {
		switch c {
		case ':':
			elements = append(elements, queryContext[0:i])
			start = i + 1
			hasNamespace = true
		case '.':
			if !hasNamespace {
				elements = append(elements, "")
			}
			elements = append(elements, queryContext[start:i])
			start = i + 1
		}
	}
	if len(elements) == 0 {
		elements = append(elements, "")
	}
	return elements
}

// for now the only formats we support are
// blank
// namespace:
// namespace:bucket.scope
// [:]bucket.scope
func ValidateQueryContext(queryContext string) errors.Error {
	hasNamespace := false
	parts := 0
	countPart := true
	for _, c := range queryContext {
		switch c {
		case ':':
			if hasNamespace {
				return errors.NewQueryContextError("repeated namespace")
			} else {
				hasNamespace = true
				countPart = true
			}
		case '.':
			if countPart {
				return errors.NewQueryContextError("missing bucket")
			}
			if !hasNamespace {
				parts++ // namespace is implied
				hasNamespace = true
				countPart = true
			}
		default:
			if countPart {
				parts++
				countPart = false
				if parts > 3 {
					return errors.NewQueryContextError("too many context elements")
				}
			}
		}
		if parts == 2 {
			return errors.NewQueryContextError("missing scope")
		}
	}
	return nil
}
