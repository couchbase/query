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

	"github.com/couchbase/query/datastore"
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

// the path is expected to be syntactically correct - ie one produced with FullName
func ParsePath(path string) []string {
	return parsePathOrContext(path)
}

func IsSystem(namespaceOrPath string) bool {
	l := len(datastore.SYSTEM_NAMESPACE)
	return len(namespaceOrPath) >= l && namespaceOrPath[0:l] == datastore.SYSTEM_NAMESPACE
}

// Create a path from a namespace:keyspace combination.
func NewPathShort(namespace, keyspace string) *Path {
	return &Path{
		elements: []string{namespace, keyspace},
	}
}

func SetPathShort(namespace, keyspace string, path *Path) {
	path.elements = []string{namespace, keyspace}
}

// Create a path from a namespace:bucket.scope.keyspace combination
func NewPathLong(namespace, bucket, scope, keyspace string) *Path {
	return &Path{
		elements: []string{namespace, bucket, scope, keyspace},
	}
}

func SetPathLong(namespace, bucket, scope, keyspace string, path *Path) {
	path.elements = []string{namespace, bucket, scope, keyspace}
}

// Create a scope path from a namespace:bucket.scope combination
func NewPathScope(namespace, bucket, scope string) *Path {
	return &Path{
		elements: []string{namespace, bucket, scope},
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
		return &Path{elements: []string{namespace, keyspace}}
	}

	elems := ParseQueryContext(queryContext)

	if elems[0] == "" {
		elems[0] = namespace
	}
	return &Path{
		elements: append(elems, keyspace),
	}
}

// For use with dynamic keyspaces, creates a full path from a full path string or a keyspace and a context
func NewVariablePathWithContext(keyspace, namespace, queryContext string) (*Path, errors.Error) {
	res, parts := validatePathOrContext(keyspace)
	if res != "" || parts == 0 {
		return nil, errors.NewDatastoreInvalidPathError(keyspace)
	}
	switch parts {
	case 1:
		return NewPathWithContext(keyspace, namespace, queryContext), nil
	case 3:
		return nil, errors.NewDatastoreInvalidPathError(keyspace)
	default:
		elems := parsePathOrContext(keyspace)
		if elems[0] == "" {
			elems[0] = namespace
		}
		return &Path{elements: elems}, nil
	}
}

func NewPathFromElements(elems []string) *Path {
	return &Path{elements: elems}
}

// These two are used to generate partial paths for RBAC roles
func (path *Path) BucketPath() *Path {
	return &Path{elements: path.elements[:2]}
}

func (path *Path) ScopePath() *Path {
	if len(path.elements) == 2 {
		return nil
	}
	return &Path{elements: path.elements[:3]}
}

func (path *Path) Namespace() string {
	return path.elements[0]
}

func (path *Path) IsSystem() bool {
	return len(path.elements) > 0 && path.elements[0] == datastore.SYSTEM_NAMESPACE
}

// the next three methods are currently unused but left for completeness
func (path *Path) Bucket() string {
	if len(path.elements) > 1 {
		return path.elements[1]
	} else {
		return ""
	}
}

func (path *Path) Scope() string {
	if len(path.elements) > 2 {
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

func (path *Path) FullName() string {
	return path.string(false, false)
}

func (path *Path) QueryContext() string {
	return path.string(false, true)
}

func (path *Path) SimpleString() string {
	return path.string(false, false)
}

func (path *Path) ProtectedString() string {
	return path.string(true, false)
}

func (path *Path) string(forceBackticks bool, isContext bool) string {
	acc := ""
	lastIndex := len(path.elements) - 1
	if isContext {
		lastIndex--
	}
	for i := 0; i <= lastIndex; i++ {
		s := path.elements[i]

		// The first element, i.e. the namespace, may be an empty string.
		// That means we can omit it, and the separator after it.
		if i == 0 && s == "" {
			continue
		}
		// Wrap any element that contains "." in back-ticks.
		if forceBackticks || strings.IndexByte(s, '.') >= 0 {
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
		} else if isContext && i == 0 {

			// always terminate namespaces with ':' for scopes
			acc += ":"
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
func ParseQueryContext(queryContext string) []string {
	if queryContext == "" || queryContext == ":" {
		return []string{""}
	}
	return parsePathOrContext(queryContext)
}

func parsePathOrContext(queryContext string) []string {
	elements := []string{}
	hasNamespace := false
	start := 0
	end := 0
	inBackTicks := false
	for i, c := range queryContext {
		switch c {
		case '`':
			inBackTicks = !inBackTicks
			if inBackTicks {
				start = i + 1
			} else {
				end = i
			}
		case ':':
			if inBackTicks {
				continue
			}
			if end != i-1 {
				end = i
			}
			elements = append(elements, queryContext[start:end])
			start = i + 1
			hasNamespace = true
		case '.':
			if inBackTicks {
				continue
			}
			if !hasNamespace {
				elements = append(elements, "")
				hasNamespace = true
			}
			if end != i-1 {
				end = i
			}
			elements = append(elements, queryContext[start:end])
			start = i + 1
		}
	}
	if !hasNamespace {
		elements = append(elements, "")
	}
	if start < len(queryContext) {
		if start < end {
			elements = append(elements, queryContext[start:end])
		} else {
			elements = append(elements, queryContext[start:])
		}
	}
	return elements
}

// for now the only formats we support are
// blank
// namespace:
// namespace:bucket.scope
// [:]bucket.scope
func ValidateQueryContext(queryContext string) errors.Error {
	res, parts := validatePathOrContext(queryContext)
	if res != "" {
		return errors.NewQueryContextError(res)
	}
	if parts == 2 {
		return errors.NewQueryContextError("missing scope")
	}
	if parts > 3 {
		return errors.NewQueryContextError("too many context elements")
	}
	return nil
}

func validatePathOrContext(queryContext string) (string, int) {
	hasNamespace := false
	parts := 0
	countPart := true
	lastTerminator := -1
	lastBackTick := -1
	inBackTick := false
	for i, c := range queryContext {
		switch c {
		case '`':
			inBackTick = !inBackTick
			if inBackTick {
				if lastTerminator >= 0 && lastTerminator != i-1 {
					return "invalid use of back ticks", 0
				}
			} else {
				lastBackTick = i
			}
		case ':':
			if inBackTick {
				continue
			}
			if hasNamespace {
				return "repeated namespace", 0
			}
			if lastBackTick >= 0 && lastBackTick != i-1 {
				return "invalid use of back ticks", 0
			}
			if parts == 0 {
				parts++ // namespace is implied
			}
			hasNamespace = true
			countPart = true
			lastTerminator = i
		case '.':
			if inBackTick {
				continue
			}
			if countPart {
				return "missing bucket", 0
			}
			if lastBackTick >= 0 && lastBackTick != i-1 {
				return "invalid use of back ticks", 0
			}
			if !hasNamespace {
				parts++ // namespace is implied
				hasNamespace = true
			}
			countPart = true
			lastTerminator = i
		default:
			if countPart {
				parts++
				countPart = false
			}
		}
	}
	if inBackTick {
		return "back tick not terminated", 0
	}
	return "", parts
}
