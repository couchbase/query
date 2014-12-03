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
	"strings"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/expression"
)

func Build(stmt algebra.Statement, datastore, systemstore datastore.Datastore,
	creds datastore.Credentials, namespace string, subquery bool) (Operator, error) {
	builder := newBuilder(datastore, systemstore, creds, namespace, subquery)
	o, err := stmt.Accept(builder)

	if err != nil {
		return nil, err
	}

	_, is_prepared := o.(*Prepared)
	op := o.(Operator)
	if !subquery && !is_prepared {
		return NewSequence(op, NewStream()), nil
	} else {
		return op, nil
	}
}

type builder struct {
	datastore       datastore.Datastore
	systemstore     datastore.Datastore
	credentials     datastore.Credentials
	namespace       string
	subquery        bool
	delayProjection bool                  // Used to allow ORDER BY non-projected expressions
	where           expression.Expression // Used for index selection
	order           *algebra.Order        // Used to collect aggregates from ORDER BY
	distinct        bool
	children        []Operator
	subChildren     []Operator
}

func newBuilder(datastore, systemstore datastore.Datastore, creds datastore.Credentials, namespace string, subquery bool) *builder {
	return &builder{
		datastore:       datastore,
		systemstore:     systemstore,
		credentials:     creds,
		namespace:       namespace,
		subquery:        subquery,
		delayProjection: false,
	}
}

func (this *builder) getTermKeyspace(node *algebra.KeyspaceTerm) (datastore.Keyspace, error) {
	node.SetDefaultNamespace(this.namespace)
	ns := node.Namespace()

	datastore := this.datastore
	if strings.ToLower(ns) == "#system" {
		datastore = this.systemstore
	}

	namespace, err := datastore.NamespaceByName(ns)
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(node.Keyspace())
	if err != nil {
		return nil, err
	}

	return keyspace, nil
}

func (this *builder) Credentials() datastore.Credentials {
	return this.credentials
}
