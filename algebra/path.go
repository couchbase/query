//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func IsSystemId(namespaceOrPath string) bool {
	l := len(datastore.SYSTEM_NAMESPACE)
	return len(namespaceOrPath) >= l && namespaceOrPath[0:l] == datastore.SYSTEM_NAMESPACE
}

func IsSystemName(namespaceOrPath string) bool {
	l := len(datastore.SYSTEM_NAMESPACE_NAME)
	return len(namespaceOrPath) >= l && namespaceOrPath[0:l] == datastore.SYSTEM_NAMESPACE_NAME
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

// Create a scope path from an identifier and query context
func NewPathScopeWithContext(namespace, scope, queryContext string) (*Path, errors.Error) {
	elems := ParseQueryContext(queryContext)
	if elems[0] == "" {
		elems[0] = namespace
	}
	elems = append(elems, scope)
	if len(elems) != 3 {
		return nil, errors.NewDatastoreInvalidScopePartsError(elems...)
	}
	return &Path{
		elements: elems,
	}, nil
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
// :bucket (produces a long path with the scope set to _default)
// namespace:bucket (as above)
func NewPathWithContext(keyspace, namespace, queryContext string) *Path {
	if queryContext == "" {
		return &Path{elements: []string{namespace, keyspace}}
	}

	elems := ParseQueryContext(queryContext)

	if elems[0] == "" {
		elems[0] = namespace
	}

	// if the query context scope isn't specified, use the default scope
	if len(elems) == 2 {
		elems = append(elems, "_default")
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
		// 3 means either [namespace]:bucket.scope or ident.ident
		// if no : then try to see if it's a valid scope + keyspace
		if strings.IndexByte(keyspace, ':') >= 0 {
			return nil, errors.NewDatastoreInvalidPathError(keyspace)
		}
		elems := parsePathOrContext(keyspace)
		return NewPathFromElementsWithContext(elems[1:2], namespace, queryContext)
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

// Creates a full path from a slice of identifiers and a query context
// if the slice contains a single keyspace, it behaves like NewPathWithContext()
// two elements are taken to mean scope.collection, and work with a queryContext
// of the form
// :bucket (namespace is either specified by the namespace parameter or set later)
// namespace:bucket (produces a long path)
// any other path length or query context yields an error
func NewPathFromElementsWithContext(parts []string, namespace, queryContext string) (*Path, errors.Error) {
	if len(parts) == 1 {
		return NewPathWithContext(parts[0], namespace, queryContext), nil
	}
	elems := ParseQueryContext(queryContext)

	if elems[0] == "" {
		elems[0] = namespace
	}

	elems = append(elems, parts...)
	if len(elems) != 4 {
		return nil, errors.NewDatastoreInvalidKeyspacePartsError(elems...)
	}
	return &Path{elements: elems}, nil
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
	return pathFromParts(forceBackticks, isContext, path.elements...)
}

func PathFromParts(elements ...string) string {
	return pathFromParts(false, false, elements...)
}

func pathFromParts(forceBackticks bool, isContext bool, elements ...string) string {
	acc := ""
	lastIndex := len(elements) - 1
	if isContext {
		lastIndex--
	}
	for i := 0; i <= lastIndex; i++ {
		s := elements[i]

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
// namespace:bucket
// namespace:bucket.scope
// [:]bucket
// [:]bucket.scope
func ValidateQueryContext(queryContext string) errors.Error {
	res, parts := validatePathOrContext(queryContext)
	if res != "" {
		return errors.NewQueryContextError(res)
	}
	if parts > 3 {
		return errors.NewQueryContextError("too many context elements")
	}
	return nil
}

func PartsFromPath(path string) int {
	res, parts := validatePathOrContext(path)
	if res != "" {
		return -1
	}
	return parts
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
