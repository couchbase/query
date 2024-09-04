//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package execution provides query execution. The execution is
data-parallel to the extent possible.
*/
package execution

import (
	"encoding/json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type stopChannel chan int

type OpSendAction interface {
	SendAction(action opAction) // Stop or Pause the operator
}

type Operator interface {
	json.Marshaler // used for profiling

	consumer

	OpSendAction

	Accept(visitor Visitor) (interface{}, error)
	ValueExchange() *valueExchange                 // Closed by this operator
	Input() Operator                               // Read by this operator
	SetInput(op Operator)                          // Can be set
	Output() Operator                              // Written by this operator
	SetOutput(op Operator)                         // Can be set
	Stop() OpSendAction                            // Notified when this operator stops
	SetStop(op OpSendAction)                       // Can be set
	Parent() Operator                              // Notified when this operator stops
	SetParent(parent Operator)                     // Can be set
	Bit() uint8                                    // Child bit
	SetBit(b uint8)                                // Child bit
	SetRoot(context *Context)                      // Let the root operator know that it is, in fact, root
	SetKeepAlive(children int, context *Context)   // Sets keep alive
	IsSerializable() bool                          // The operator supports being serialized
	IsParallel() bool                              // The operator has more than one producer
	SerializeOutput(op Operator, context *Context) // Has the producer run the consumer inline
	Copy() Operator                                // Keep input/output/parent; make new channels
	RunOnce(context *Context, parent value.Value)  // Uses Once.Do() to run exactly once; never panics
	Done()                                         // Frees and pools resources

	reopen(context *Context) bool // resets operator to initial state
	close(context *Context)       // the operator is no longer operating!
	keepAlive(op Operator) bool   // operator was set to terminate early
	stopCh() stopChannel          // Never closed, just garbage-collected

	getBase() *base

	PlanOp() plan.Operator

	// local infrastructure to add up times of children of the parallel operator
	accrueTimes(o Operator)
	time() *base
	accrueTime(b *base)

	stashOutput()
}

func hasSystemXattrs(options value.Value) bool {
	if options != nil && options.Type() == value.OBJECT {
		if v, ok := options.Field("xattrs"); ok && v.Type() == value.OBJECT {
			for k, _ := range v.Fields() {
				if k[0] == '_' {
					return true
				}
			}
		}
	}
	return false
}

func authForSysXattrs(ks datastore.Keyspace, context *Context) errors.Error {
	if ds := datastore.GetDatastore(); ds != nil {
		privs := auth.NewPrivileges()
		privs.Add(ks.QualifiedName(), auth.PRIV_XATTRS_WRITE, auth.PRIV_PROPS_NONE)
		return ds.Authorize(privs, context.Credentials())
	}
	return nil
}
