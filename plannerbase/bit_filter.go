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
)

type BFSource struct {
	joinAlias string
	bfInfos   map[string]*BFInfo
}

func newBFSource(alias string, bfInfos map[string]*BFInfo) *BFSource {
	return &BFSource{
		joinAlias: alias,
		bfInfos:   bfInfos,
	}
}

func (this *BFSource) JoinAlias() string {
	return this.joinAlias
}

func (this *BFSource) BFInfos() map[string]*BFInfo {
	return this.bfInfos
}

func (this *BFSource) GetIndexExprs(index datastore.Index, alias string,
	buildBFInfos map[datastore.Index]*BuildBFInfo) expression.Expressions {
	probeExprs := make(expression.Expressions, 0, len(this.bfInfos))
	for _, bfInfo := range this.bfInfos {
		if _, ok := bfInfo.indexes[index]; ok {
			if curBFInfo, ok := buildBFInfos[index]; ok && curBFInfo != nil {
				curBFInfo.exprs = append(curBFInfo.exprs, bfInfo.other)
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
}

func newBFInfo(index datastore.Index, self, other expression.Expression) *BFInfo {
	return &BFInfo{
		indexes: map[datastore.Index]bool{index: true},
		self:    self,
		other:   other,
	}
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

func GetBFInfoExprs(probeAlias string, buildBFInfos map[datastore.Index]*BuildBFInfo) map[datastore.Index]expression.Expressions {
	buildExprs := make(map[datastore.Index]expression.Expressions, len(buildBFInfos))
	for index, bfInfo := range buildBFInfos {
		if bfInfo.alias == probeAlias {
			buildExprs[index] = bfInfo.exprs
		}
	}
	return buildExprs
}
