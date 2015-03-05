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

	"github.com/couchbase/query/algebra"
)

func (this *builder) VisitMerge(stmt *algebra.Merge) (interface{}, error) {
	children := make([]Operator, 0, 8)
	subChildren := make([]Operator, 0, 8)
	source := stmt.Source()

	if source.Select() != nil {
		sel, err := source.Select().Accept(this)
		if err != nil {
			return nil, err
		}

		children = append(children, sel.(Operator))
	} else {
		if source.From() == nil {
			return nil, fmt.Errorf("MERGE missing source.")
		}

		_, err := source.From().Accept(this)
		if err != nil {
			return nil, err
		}

		// Update local operator slices with results of building From:
		children = append(children, this.children...)
		subChildren = append(subChildren, this.subChildren...)

	}

	if source.As() != "" {
		subChildren = append(subChildren, NewAlias(source.As()))
	}

	ksref := stmt.KeyspaceRef()
	ksref.SetDefaultNamespace(this.namespace)

	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	actions := stmt.Actions()
	var update, delete, insert Operator

	if actions.Update() != nil {
		act := actions.Update()
		ops := make([]Operator, 0, 5)

		if act.Where() != nil {
			ops = append(ops, NewFilter(act.Where()))
		}

		ops = append(ops, NewClone(ksref.Alias()))

		if act.Set() != nil {
			ops = append(ops, NewSet(act.Set()))
		}

		if act.Unset() != nil {
			ops = append(ops, NewUnset(act.Unset()))
		}

		ops = append(ops, NewSendUpdate(keyspace, ksref.Alias(), stmt.Limit()))
		update = NewSequence(ops...)
	}

	if actions.Delete() != nil {
		act := actions.Delete()
		ops := make([]Operator, 0, 4)

		if act.Where() != nil {
			ops = append(ops, NewFilter(act.Where()))
		}

		ops = append(ops, NewSendDelete(keyspace, ksref.Alias(), stmt.Limit()))
		delete = NewSequence(ops...)
	}

	if actions.Insert() != nil {
		act := actions.Insert()
		ops := make([]Operator, 0, 4)

		if act.Where() != nil {
			ops = append(ops, NewFilter(act.Where()))
		}

		ops = append(ops, NewSendInsert(keyspace, ksref.Alias(), stmt.Key(), nil, stmt.Limit()))
		insert = NewSequence(ops...)
	}

	merge := NewMerge(keyspace, ksref, stmt.Key(), update, delete, insert)
	subChildren = append(subChildren, merge)

	if stmt.Returning() != nil {
		subChildren = append(subChildren, NewInitialProject(stmt.Returning()), NewFinalProject())
	}

	parallel := NewParallel(NewSequence(subChildren...))
	children = append(children, parallel)

	if stmt.Limit() != nil {
		children = append(children, NewLimit(stmt.Limit()))
	}

	if stmt.Returning() == nil {
		children = append(children, NewDiscard())
	}

	return NewSequence(children...), nil
}
