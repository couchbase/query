//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbase/query/distributed"
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
	rv := &NodeName{
		*NewNullaryFunctionBase("node_name"),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NodeName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NodeName) Type() value.Type { return value.STRING }

/*
Wrap the local node name in a value and return it.
WhoAmI() returns no error, but an empty string if
the service is not part of a cluster
*/
func (this *NodeName) Evaluate(item value.Value, context Context) (value.Value, error) {
	return value.NewValue(distributed.RemoteAccess().WhoAmI()), nil
}

/*
Factory method pattern.
*/
func (this *NodeName) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNodeName()
	}
}
