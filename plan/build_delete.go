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
)

func (this *builder) VisitDelete(stmt *algebra.Delete) (interface{}, error) {
	this.where = stmt.Where()

	ksref := stmt.KeyspaceRef()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	err = this.beginMutate(keyspace, ksref, stmt.Keys(), stmt.Where())
	if err != nil {
		return nil, err
	}

	creds := this.Credentials()
	auth := NewAuthenticate(keyspace, creds, datastore.CAN_WRITE)
	this.subChildren = append(this.subChildren, auth)

	subChildren := this.subChildren
	subChildren = append(subChildren, NewSendDelete(keyspace, stmt.Limit()))

	if stmt.Returning() != nil {
		subChildren = append(subChildren, NewInitialProject(stmt.Returning()), NewFinalProject())
	}

	parallel := NewParallel(NewSequence(subChildren...))
	this.children = append(this.children, parallel)

	if stmt.Limit() != nil {
		this.children = append(this.children, NewLimit(stmt.Limit()))
	}

	if stmt.Returning() == nil {
		this.children = append(this.children, NewDiscard())
	}

	return NewSequence(this.children...), nil
}
