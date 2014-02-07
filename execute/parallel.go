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
	"runtime"

	"github.com/couchbaselabs/query/value"
)

type Parallel struct {
	base
	child        Operator
	childChannel StopChannel
}

func NewParallel(child Operator) *Parallel {
	rv := &Parallel{
		base:         newBase(),
		child:        child,
		childChannel: make(StopChannel, runtime.NumCPU()),
	}

	rv.output = rv
	return rv
}

func (this *Parallel) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParallel(this)
}

func (this *Parallel) Copy() Operator {
	return &Parallel{
		base:         this.base.copy(),
		child:        this.child.Copy(),
		childChannel: make(StopChannel, runtime.NumCPU()),
	}
}

func (this *Parallel) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		this.child.SetInput(this.input)
		this.child.SetOutput(this.output)
		this.child.SetStop(nil)
		this.child.SetParent(this)

		n := runtime.NumCPU()

		children := make([]Operator, n)
		children[0] = this.child
		for i := 1; i < n; i++ {
			children[i] = this.child.Copy()
		}

		// Run children in parallel
		for i := 0; i < n; i++ {
			go children[i].RunOnce(context, parent)
		}

		for {
			select {
			// Wait for all children
			case <-this.childChannel: // Never closed
				if n--; n <= 0 {
					return
				}
			case <-this.stopChannel: // Never closed
				for _, child := range children {
					child.StopChannel() <- false
				}
			}
		}
	})
}

func (this *Parallel) ChildChannel() StopChannel {
	return this.childChannel
}
