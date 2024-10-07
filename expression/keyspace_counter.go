//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

// check whether expr has a single keyspace reference to 'alias'
func HasSingleKeyspaceReference(expr Expression, alias string, keyspaces map[string]string) bool {
	refs, err := CountKeySpaces(expr, keyspaces)
	if err == nil && len(refs) == 1 {
		if _, ok := refs[alias]; ok {
			return true
		}
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
	if err != nil {
		return nil, err
	}
	if _, ok := expr.Second().(*FieldName); !ok {
		err = this.Traverse(expr.Second())
	}
	return nil, err
}

func (this *keyspaceCounter) VisitIdentifier(expr *Identifier) (interface{}, error) {
	keyspace := expr.Identifier()
	if len(keyspace) == 0 {
		return nil, errors.NewPlanInternalError("keyspaceCounter.VisitIdentifier: empty keyspace name")
	}

	if expr.CaseInsensitive() {
		keyspace = strings.ToLower(keyspace)
		for k, _ := range this.baseKeyspaces {
			if keyspace == strings.ToLower(k) {
				this.keyspaces[k] = this.baseKeyspaces[k]
				break
			}
		}
	} else {
		if _, ok := this.baseKeyspaces[keyspace]; ok {
			if _, ok = this.keyspaces[keyspace]; !ok {
				this.keyspaces[keyspace] = this.baseKeyspaces[keyspace]
			}
		}
	}

	return nil, nil
}
