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
	"fmt"
	"strings"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/datastore"
)

func (this *builder) VisitCreatePrimaryIndex(node *algebra.CreatePrimaryIndex) (interface{}, error) {
	ksref := node.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	return NewCreatePrimaryIndex(keyspace), nil
}

func (this *builder) VisitCreateIndex(node *algebra.CreateIndex) (interface{}, error) {
	ksref := node.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	return NewCreateIndex(keyspace, node), nil
}

func (this *builder) VisitDropIndex(node *algebra.DropIndex) (interface{}, error) {
	ksref := node.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	index, er := keyspace.IndexByName(node.Name())
	if er != nil {
		return nil, er
	}

	return NewDropIndex(index, node), nil
}

func (this *builder) VisitAlterIndex(node *algebra.AlterIndex) (interface{}, error) {
	ksref := node.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	index, er := keyspace.IndexByName(node.Name())
	if er != nil {
		return nil, er
	}

	return NewAlterIndex(index, node), nil
}

func (this *builder) getNameKeyspace(ns, ks string) (datastore.Keyspace, error) {
	if ns == "" {
		ns = this.namespace
	}

	if strings.ToLower(ns) == "#system" {
		return nil, fmt.Errorf("Index operations not allowed on system namespace.")
	}

	datastore := this.datastore
	namespace, err := datastore.NamespaceByName(ns)
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(ks)
	if err != nil {
		return nil, err
	}

	return keyspace, nil
}
