//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/expression"
)

func (this *builder) beginMutate(keyspace datastore.Keyspace,
	ksref *algebra.KeyspaceRef, keys, where expression.Expression) error {
	ksref.SetDefaultNamespace(this.namespace)
	term := algebra.NewKeyspaceTerm(ksref.Namespace(), ksref.Keyspace(), nil, ksref.As(), nil)

	this.children = make([]Operator, 0, 8)
	this.subChildren = make([]Operator, 0, 8)

	if keys != nil {
		scan := NewKeyScan(keys)
		this.children = append(this.children, scan)
	} else {
		scan, err := this.selectScan(keyspace, term)
		if err != nil {
			return err
		}

		this.children = append(this.children, scan)
	}

	fetch := NewFetch(keyspace, term)
	this.subChildren = append(this.subChildren, fetch)

	if where != nil {
		this.subChildren = append(this.subChildren, NewFilter(where))
	}

	return nil
}
