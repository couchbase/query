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

type Sequence struct {
	base
	children     []Operator
	childChannel StopChannel
}

func NewSequence(children ...Operator) *Sequence {
	rv := &Sequence{
		base:         newBase(),
		children:     children,
		childChannel: make(StopChannel, 1),
	}

	rv.output = rv
	return rv
}

func (this *Sequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSequence(this)
}

func (this *Sequence) Copy() Operator {
	children := make([]Operator, len(this.children))

	for i, child := range this.children {
		children[i] = child.Copy()
	}

	return &Sequence{
		base:         this.base.copy(),
		children:     children,
		childChannel: make(StopChannel, 1),
	}
}

func (this *Sequence) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		first_child := this.children[0]
		first_child.SetInput(this.input)
		first_child.SetStop(this.stop)

		n := len(this.children)

		for i := 1; i < n; i++ {
			this.children[i].SetInput(this.children[i-1].Output())
			this.children[i].SetStop(this.children[i-1])
		}

		last_child := this.children[n-1]
		last_child.SetOutput(this.output)
		last_child.SetParent(this)

		// Run last child
		go last_child.RunOnce(context, parent)

		for {
			select {
			case <-this.childChannel: // Never closed
				// Wait for last child
				return
			case <-this.stopChannel: // Never closed
				this.notifyStop()
				select {
				case last_child.StopChannel() <- false:
				default:
				}
			}
		}
	})
}

func (this *Sequence) ChildChannel() StopChannel {
	return this.childChannel
}
