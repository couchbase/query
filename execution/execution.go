//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package execution provides query execution. The execution is
data-parallel to the extent possible.

*/
package execution

import (
	"encoding/json"

	"github.com/couchbase/query/value"
)

type stopChannel chan int

type Operator interface {
	json.Marshaler // used for profiling

	Accept(visitor Visitor) (interface{}, error)
	ItemChannel() annotatedChannel                // Closed by this operator
	Input() Operator                              // Read by this operator
	SetInput(op Operator)                         // Can be set
	Output() Operator                             // Written by this operator
	SetOutput(op Operator)                        // Can be set
	Stop() Operator                               // Notified when this operator stops
	SetStop(op Operator)                          // Can be set
	Parent() Operator                             // Notified when this operator stops
	SetParent(parent Operator)                    // Can be set
	Bit() uint8                                   // Child bit
	SetBit(b uint8)                               // Child bit
	SetRoot()                                     // Let the root operator know that it is, in fact, root
	SetKeepAlive(children int, context *Context)  // Sets keep alive
	Copy() Operator                               // Keep input/output/parent; make new channels
	RunOnce(context *Context, parent value.Value) // Uses Once.Do() to run exactly once; never panics
	SendStop()                                    // Stops the operator
	Done()                                        // Frees and pools resources

	reopen(context *Context)    // resets operator to initial state
	close(context *Context)     // the operator is no longer operating!
	keepAlive(op Operator) bool // operator was set to terminate early
	stopCh() stopChannel        // Never closed, just garbage-collected
	childCh() stopChannel       // Never closed, just garbage-collected

	// local infrastructure to add up times of children of the parallel operator
	accrueTimes(o Operator)
	time() *base
	accrueTime(b *base)
}
