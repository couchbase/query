//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"strings"

	"github.com/couchbase/query/errors"
)

// gather (and count) keyspace references for an expression
func CountKeySpaces(expr Expression, baseKeyspaces map[string]string) (map[string]string, error) {

	counter := newKeyspaceCounter(baseKeyspaces)
	_, err := expr.Accept(counter)
	if err != nil {
		return nil, err
	}

	return counter.keyspaces, nil
}

// check whether expr has references to any of the keyspaces
func HasKeyspaceReferences(expr Expression, keyspaces map[string]string) bool {
	refs, err := CountKeySpaces(expr, keyspaces)
	if err == nil && len(refs) > 0 {
		return true
	}
	return false
}

type keyspaceCounter struct {
	TraverserBase

	baseKeyspaces map[string]string
	keyspaces     map[string]string
}

func newKeyspaceCounter(baseKeyspaces map[string]string) *keyspaceCounter {
	rv := &keyspaceCounter{
		baseKeyspaces: baseKeyspaces,
		keyspaces:     make(map[string]string, len(baseKeyspaces)),
	}

	rv.traverser = rv
	return rv
}

func (this *keyspaceCounter) VisitField(expr *Field) (interface{}, error) {
	err := this.Traverse(expr.First())
	return nil, err
}

func (this *keyspaceCounter) VisitIdentifier(expr *Identifier) (interface{}, error) {
	keyspace := expr.String()
	keyspace = strings.Trim(keyspace, "`")
	if len(keyspace) == 0 {
		return nil, errors.NewPlanInternalError("keyspaceCounter.VisitIdentifier: empty keyspace name")
	}

	if _, ok := this.baseKeyspaces[keyspace]; ok {
		if _, ok = this.keyspaces[keyspace]; !ok {
			this.keyspaces[keyspace] = this.baseKeyspaces[keyspace]
		}
	}

	return nil, nil
}
