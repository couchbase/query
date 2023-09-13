//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// NodeName
//
///////////////////////////////////////////////////

/*
This represents the distributed function node_name().
It returns the name of the local node, if part of a
cluster
*/
type NodeName struct {
	NullaryFunctionBase
}

func NewNodeName() Function {
	rv := &NodeName{}
	rv.Init("node_name")
	// technically there's the possibility that names change
	// for nodes that start their lives as "127.0.0.1:8091"
	// but that's a once in a lifetime event, and is going
	// to happen pre production work, so...
	rv.unsetVolatile()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NodeName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NodeName) Type() value.Type {
	return value.STRING
}

/*
Wraps the local node name (in on-prem) or local nodeUUID (in serverless) in a value and return it.
WhoAmI() returns no error, but an empty string if
the service is not part of a cluster
*/
func (this *NodeName) Evaluate(item value.Value, context Context) (value.Value, error) {
	return value.NewValue(tenant.EncodeNodeName(distributed.RemoteAccess().WhoAmI())), nil
}

/*
static value (for pushdowns)
*/
func (this *NodeName) Static() Expression {
	return this.expr.(Function)
}

/*
Factory method pattern.
*/
func (this *NodeName) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNodeName()
	}
}

func (this *NodeName) Indexable() bool {
	return false
}

///////////////////////////////////////////////////
//
// NodeUUID
//
///////////////////////////////////////////////////

type NodeUUID struct {
	UnaryFunctionBase
}

func NewNodeUUID(operand Expression) Function {
	rv := &NodeUUID{}
	rv.Init("node_uuid", operand)

	rv.unsetVolatile()
	rv.expr = rv
	return rv
}

func (this *NodeUUID) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NodeUUID) Type() value.Type {
	return value.STRING
}

func (this *NodeUUID) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}
	host := arg.ToString()
	if host == "" {
		host = distributed.RemoteAccess().WhoAmI()
	}

	return value.NewValue(distributed.RemoteAccess().NodeUUID(host)), nil
}

func (this *NodeUUID) Static() Expression {
	return this.expr.(Function)
}

func (this *NodeUUID) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNodeUUID(operands[0])
	}
}

func (this *NodeUUID) Indexable() bool {
	return false
}
