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
func CountKeySpaces(expr Expression, baseKeyspaces map[string]bool) (map[string]bool, error) {

	counter := newKeyspaceCounter(baseKeyspaces)
	_, err := expr.Accept(counter)
	if err != nil {
		return nil, err
	}

	return counter.keyspaces, nil
}

type keyspaceCounter struct {
	TraverserBase

	baseKeyspaces map[string]bool
	keyspaces     map[string]bool
	withinField   bool
}

func newKeyspaceCounter(baseKeyspaces map[string]bool) *keyspaceCounter {
	rv := &keyspaceCounter{
		baseKeyspaces: baseKeyspaces,
		keyspaces:     make(map[string]bool, len(baseKeyspaces)),
	}

	rv.traverser = rv
	return rv
}

func (this *keyspaceCounter) VisitField(expr *Field) (interface{}, error) {
	withinField := this.withinField
	defer func() { this.withinField = withinField }()
	this.withinField = true

	err := this.Traverse(expr.First())
	return nil, err
}

func (this *keyspaceCounter) VisitIdentifier(expr *Identifier) (interface{}, error) {
	if !this.withinField {
		return nil, nil
	}

	keyspace := expr.String()
	keyspace = strings.Trim(keyspace, "`")
	if len(keyspace) == 0 {
		return nil, errors.NewPlanInternalError("keyspaceCounter.VisitIdentifier: empty keyspace name")
	}

	if _, ok := this.baseKeyspaces[keyspace]; ok {
		if _, ok = this.keyspaces[keyspace]; !ok {
			this.keyspaces[keyspace] = true
		}
	}

	return nil, nil
}
