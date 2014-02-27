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
	"github.com/couchbaselabs/query/value"
)

type All struct {
	base
	children     []Operator
	childChannel StopChannel
}

func NewAll(children []Operator) *All {
	rv := &All{
		base:         newBase(),
		children:     children,
		childChannel: make(StopChannel, len(children)),
	}

	rv.output = rv
	return rv
}

func (this *All) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAll(this)
}

func (this *All) Copy() Operator {
	rv := &All{
		base:         this.base.copy(),
		childChannel: make(StopChannel, len(this.children)),
	}

	children := make([]Operator, len(this.children))
	for i, c := range this.children {
		children[i] = c.Copy()
	}

	rv.children = children
	return rv
}

func (this *All) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		n := len(this.children)

		// Run children in parallel
		for _, child := range this.children {
			child.SetOutput(this.output)
			go child.RunOnce(context, parent)
		}

		for n > 0 {
			select {
			case <-this.childChannel: // Never closed
				// Wait for all children
				n--
			case <-this.stopChannel: // Never closed
				this.notifyStop()
				for _, child := range this.children {
					select {
					case child.StopChannel() <- false:
					default:
					}
				}
			}
		}
	})
}

func (this *All) ChildChannel() StopChannel {
	return this.childChannel
}
