//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.
//

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

type Optimizer interface {
	Initialize(builder Builder)
	OptimizeQueryBlock(node algebra.Node) (plan.Operator, error)
}

type Builder interface {
	GetBaseKeyspaces() map[string]*base.BaseKeyspace
	GetPrepareContext() *PrepareContext
	BuildScan(node algebra.SimpleFromTerm) ([]plan.Operator, []plan.CoveringOperator, error)
	BuildJoin(node *algebra.AnsiJoin) (plan.Operator, error)
}

func (this *builder) GetBaseKeyspaces() map[string]*base.BaseKeyspace {
	return this.baseKeyspaces
}

func (this *builder) GetPrepareContext() *PrepareContext {
	return this.context
}

func (this *builder) BuildScan(node algebra.SimpleFromTerm) ([]plan.Operator, []plan.CoveringOperator, error) {
	// Test code to check if simple scans can flow through the system completely
	// Needs a lot more work, will not be invoked for now.
	children := this.children
	subChildren := this.subChildren
	coveringScans := this.coveringScans
	countScan := this.countScan
	orderScan := this.orderScan
	lastOp := this.lastOp
	indexPushDowns := this.storeIndexPushDowns()
	defer func() {
		this.children = children
		this.subChildren = subChildren
		this.coveringScans = coveringScans
		this.countScan = countScan
		this.orderScan = orderScan
		this.lastOp = lastOp
		this.restoreIndexPushDowns(indexPushDowns, true)
	}()

	this.children = make([]plan.Operator, 0, 16)
	this.subChildren = make([]plan.Operator, 0, 16)
	this.coveringScans = nil
	this.countScan = nil
	this.order = nil
	this.orderScan = nil
	this.limit = nil
	this.offset = nil
	this.lastOp = nil

	_, err := node.Accept(this)
	if err != nil {
		return nil, nil, err
	}
	return this.children, this.coveringScans, nil

	//return nil, nil, nil
}

func (this *builder) BuildJoin(node *algebra.AnsiJoin) (plan.Operator, error) {
	return nil, nil
}
