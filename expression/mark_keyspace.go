//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

func MarkKeyspace(keyspace string, expr Expression) {
	km := newKeyspaceMarker(keyspace)
	_, _ = km.Map(expr)
	return
}

// keyspaceMarker is used to mark an identifier as keyspace identifier
type keyspaceMarker struct {
	MapperBase

	keyspace string
}

func newKeyspaceMarker(keyspace string) *keyspaceMarker {
	rv := &keyspaceMarker{
		keyspace: keyspace,
	}

	rv.mapFunc = func(expr Expression) (Expression, error) {
		if keyspace == "" {
			return expr, nil
		} else {
			return expr, expr.MapChildren(rv)
		}
	}

	rv.mapper = rv
	return rv
}

func (this *keyspaceMarker) VisitIdentifier(expr *Identifier) (interface{}, error) {
	if expr.identifier == this.keyspace {
		expr.SetKeyspaceAlias(true)
	}
	return expr, nil
}
