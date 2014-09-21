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

	"github.com/couchbaselabs/query/algebra"
)

func (this *builder) VisitUpsert(node *algebra.Upsert) (interface{}, error) {
	ksref := node.KeyspaceRef()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	children := make([]Operator, 0, 2)

	if node.Values() != nil {
		children = append(children, NewValueScan(node.Values()))
	} else if node.Select() != nil {
		sel, err := node.Select().Accept(this)
		if err != nil {
			return nil, err
		}

		children = append(children, sel.(Operator))
	} else {
		return nil, fmt.Errorf("UPSERT missing both VALUES and SELECT.")
	}

	subChildren := make([]Operator, 0, 3)
	subChildren = append(subChildren, NewSendUpsert(keyspace, node.Key()))
	if node.Returning() != nil {
		subChildren = append(subChildren, NewInitialProject(node.Returning()), NewFinalProject())
	}

	parallel := NewParallel(NewSequence(subChildren...))
	children = append(children, parallel)
	return NewSequence(children...), nil
}
