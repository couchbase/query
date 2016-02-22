//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
)

func (this *builder) visitFrom(node *algebra.Subselect, group *algebra.Group) error {
	count, err := this.fastCount(node)
	if err != nil {
		return err
	}

	if count {
		this.maxParallelism = 1
		this.resetOrderLimit()
	} else if node.From() != nil {
		if group != nil {
			this.resetOrderLimit()
		}

		_, err := node.From().Accept(this)
		if err != nil {
			return err
		}
	} else {
		// No FROM clause
		this.resetOrderLimit()
		scan := plan.NewDummyScan()
		this.children = append(this.children, scan)
		this.maxParallelism = 1
	}

	return nil
}

func (this *builder) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
	node.SetDefaultNamespace(this.namespace)
	keyspace, err := this.getTermKeyspace(node)
	if err != nil {
		return nil, err
	}

	if this.subquery && this.correlated && node.Keys() == nil {
		return nil, errors.NewSubqueryMissingKeysError(node.Keyspace())
	}

	scan, err := this.selectScan(keyspace, node, this.limit)
	if err != nil {
		return nil, err
	}

	this.children = append(this.children, scan)

	if this.coveringScan == nil {
		fetch := plan.NewFetch(keyspace, node)
		this.subChildren = append(this.subChildren, fetch)
	}

	return nil, nil
}

func (this *builder) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	sel, err := node.Subquery().Accept(this)
	if err != nil {
		return nil, err
	}

	this.children = make([]plan.Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]plan.Operator, 0, 16) // sub-children, executed across data-parallel streams
	this.children = append(this.children, sel.(plan.Operator), plan.NewAlias(node.Alias()))
	return nil, nil
}

func (this *builder) VisitJoin(node *algebra.Join) (interface{}, error) {
	this.resetOrderLimit()

	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	join := plan.NewJoin(keyspace, node)
	this.subChildren = append(this.subChildren, join)
	return nil, nil
}

func (this *builder) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	this.resetOrderLimit()

	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	join, err := this.buildIndexJoin(keyspace, node)
	if err != nil {
		return nil, err
	}

	this.subChildren = append(this.subChildren, join)
	return nil, nil
}

func (this *builder) VisitNest(node *algebra.Nest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	nest := plan.NewNest(keyspace, node)
	this.subChildren = append(this.subChildren, nest)
	return nil, nil
}

func (this *builder) VisitIndexNest(node *algebra.IndexNest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	nest, err := this.buildIndexNest(keyspace, node)
	if err != nil {
		return nil, err
	}

	this.subChildren = append(this.subChildren, nest)
	return nil, nil
}

func (this *builder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	this.resetOrderLimit()

	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	unnest := plan.NewUnnest(node)
	this.subChildren = append(this.subChildren, unnest)
	return nil, nil
}

func (this *builder) fastCount(node *algebra.Subselect) (bool, error) {
	if node.From() == nil ||
		node.Where() != nil ||
		node.Group() != nil {
		return false, nil
	}

	from, ok := node.From().(*algebra.KeyspaceTerm)
	if !ok ||
		from.Projection() != nil ||
		from.Keys() != nil {
		return false, nil
	}

	from.SetDefaultNamespace(this.namespace)
	keyspace, err := this.getTermKeyspace(from)
	if err != nil {
		return false, err
	}

	for _, term := range node.Projection().Terms() {
		count, ok := term.Expression().(*algebra.Count)
		if !ok || count.Operand() != nil {
			return false, nil
		}
	}

	scan := plan.NewCountScan(keyspace, from)
	this.children = append(this.children, scan)
	return true, nil
}

func (this *builder) resetOrderLimit() {
	this.order = nil
	this.limit = nil
}
