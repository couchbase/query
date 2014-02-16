//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	_ "fmt"

	"github.com/couchbaselabs/query/value"
)

type ParentScan struct {
	base
}

func NewParentScan() *ParentScan {
	rv := &ParentScan{
		base: newBase(),
	}

	rv.output = rv
	return rv
}

func (this *ParentScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParentScan(this)
}

func (this *ParentScan) Copy() Operator {
	return &ParentScan{this.base.copy()}
}

func (this *ParentScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		// Shallow copy of the parent includes
		// correlated and annotated aspects
		this.sendItem(parent.Copy().(value.AnnotatedValue))
	})
}
