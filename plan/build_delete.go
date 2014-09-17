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

func (this *builder) VisitDelete(node *algebra.Delete) (interface{}, error) {
	err := node.Formalize()
	if err != nil {
		return nil, err
	}

	ksref := node.KeyspaceRef()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	err = this.beginMutate(keyspace, ksref.Alias(), node.Keys(), node.Where())
	if err != nil {
		return nil, err
	}

	subChildren := this.subChildren
	subChildren = append(subChildren, NewSendDelete(keyspace))

	if node.Returning() != nil {
		subChildren = append(subChildren, NewInitialProject(node.Returning()), NewFinalProject())
	}

	parallel := NewParallel(NewSequence(subChildren...))
	this.children = append(this.children, parallel)

	if node.Limit() != nil {
		this.children = append(this.children, NewLimit(node.Limit()))
	}

	return NewSequence(this.children...), nil
}

func (this *builder) beginMutate(keyspace datastore.Keyspace,
	alias string, keys, where expression.Expression) error {
	this.children = make([]Operator, 0, 4)
	this.subChildren = make([]Operator, 0, 8)

	if keys != nil {
		scan := NewKeyScan(keys)
		this.children = append(this.children, scan)
	} else {
		index, err := keyspace.IndexByPrimary()
		if err != nil {
			return err
		}

		scan := NewPrimaryScan(index)
		this.children = append(this.children, scan)
	}

	fetch := NewFetch(keyspace, nil, alias)
	this.subChildren = append(this.subChildren, fetch)

	if where != nil {
		this.subChildren = append(this.subChildren, NewFilter(where))
	}

	return nil
}
