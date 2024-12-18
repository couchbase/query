//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

type BFSource struct {
	joinKeyspace *BaseKeyspace
	selecs       map[datastore.Index]float64
	bfInfos      map[string]*BFInfo
}

func newBFSource(joinKeyspace *BaseKeyspace, bfInfos map[string]*BFInfo) *BFSource {
	return &BFSource{
		joinKeyspace: joinKeyspace,
		bfInfos:      bfInfos,
	}
}

func (this *BFSource) JoinKeyspace() *BaseKeyspace {
	return this.joinKeyspace
}

func (this *BFSource) BFInfos() map[string]*BFInfo {
	return this.bfInfos
}

func (this *BFSource) DropBFInfo(exp string) {
	delete(this.bfInfos, exp)
}

func (this *BFSource) Empty() bool {
	return len(this.bfInfos) == 0
}

func (this *BFSource) SetSelec(index datastore.Index, selec float64) {
	if this.selecs == nil {
		this.selecs = make(map[datastore.Index]float64)
	}
	this.selecs[index] = selec
}

func (this *BFSource) Selec() float64 {
	selec := 1.0
	for _, sel := range this.selecs {
		if sel <= 0.0 {
			return -1.0
		}
		selec *= sel
	}
	return selec
}

func (this *BFSource) GetIndexExprs(index datastore.Index, alias string,
	buildBFInfos map[datastore.Index]*BuildBFInfo) expression.Expressions {
	probeExprs := make(expression.Expressions, 0, len(this.bfInfos))
	for _, bfInfo := range this.bfInfos {
		if _, ok := bfInfo.indexes[index]; ok {
			if curBFInfo, ok := buildBFInfos[index]; ok && curBFInfo != nil {
				add := true
				for _, exp := range curBFInfo.exprs {
					if exp.EquivalentTo(bfInfo.other) {
						add = false
						break
					}
				}
				if add {
					curBFInfo.exprs = append(curBFInfo.exprs, bfInfo.other)
				}
			} else {
				buildBFInfos[index] = newBuildBFInfo(alias, expression.Expressions{bfInfo.other})
			}
			probeExprs = append(probeExprs, bfInfo.self)
		}
	}
	return probeExprs
}

type BFInfo struct {
	indexes map[datastore.Index]bool
	self    expression.Expression
	other   expression.Expression
	fltr    *Filter
}

func newBFInfo(index datastore.Index, self, other expression.Expression, fltr *Filter) *BFInfo {
	return &BFInfo{
		indexes: map[datastore.Index]bool{index: true},
		self:    self,
		other:   other,
		fltr:    fltr,
	}
}

func (this *BFInfo) Selec() float64 {
	return this.fltr.selec
}

func (this *BFInfo) HasIndex(index datastore.Index) bool {
	_, ok := this.indexes[index]
	return ok
}

func (this *BFInfo) DropIndex(index datastore.Index) {
	delete(this.indexes, index)
}

func (this *BFInfo) Empty() bool {
	return len(this.indexes) == 0
}

func (this *BFInfo) Filter() *Filter {
	return this.fltr
}

type BuildBFInfo struct {
	alias string
	exprs expression.Expressions
}

func newBuildBFInfo(alias string, exprs expression.Expressions) *BuildBFInfo {
	return &BuildBFInfo{
		alias: alias,
		exprs: exprs,
	}
}

func (this *BuildBFInfo) Alias() string {
	return this.alias
}

func (this *BuildBFInfo) Expressions() expression.Expressions {
	return this.exprs
}

func GetBFInfoExprs(probeAlias string, buildBFInfos map[datastore.Index]*BuildBFInfo) []*plan.BitFilterIndex {
	buildExprs := make([]*plan.BitFilterIndex, 0, len(buildBFInfos))
	for index, bfInfo := range buildBFInfos {
		if bfInfo.alias == probeAlias {
			buildExprs = append(buildExprs, plan.NewBitFilterIndex(index, bfInfo.exprs))
		}
	}
	return buildExprs
}

// information on each build side term
type BuildInfo struct {
	skip    bool
	bfInfos map[datastore.Index]*BuildBFInfo
}

func NewBuildInfo() *BuildInfo {
	return &BuildInfo{}
}

func (this *BuildInfo) Skip() bool {
	return this.skip
}

func (this *BuildInfo) SetSkip() {
	this.skip = true
}

func (this *BuildInfo) BFInfos() map[datastore.Index]*BuildBFInfo {
	return this.bfInfos
}

func (this *BuildInfo) NewBFInfos() {
	this.bfInfos = make(map[datastore.Index]*BuildBFInfo, 4)
}

// this function is called on a plan constructed for a keyspace and identifies all indexes
// used for this keyspace, it should not be called with join/nest/unnest etc.
func gatherIndexes(indexes map[datastore.Index]bool, ops ...plan.Operator) {
	for _, op := range ops {
		switch op := op.(type) {
		case *plan.IndexScan3:
			indexes[op.Index()] = true
		case *plan.IntersectScan:
			gatherScanIndexes(indexes, op.Scans()...)
		case *plan.OrderedIntersectScan:
			gatherScanIndexes(indexes, op.Scans()...)
		case *plan.UnionScan:
			gatherScanIndexes(indexes, op.Scans()...)
		case *plan.DistinctScan:
			gatherScanIndexes(indexes, op.Scan())
		default:
			// skip Fetch, Filter
		}
	}
}

func gatherScanIndexes(indexes map[datastore.Index]bool, scans ...plan.SecondaryScan) {
	for _, scan := range scans {
		gatherIndexes(indexes, scan)
	}
}
