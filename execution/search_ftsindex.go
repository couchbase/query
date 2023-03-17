//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"
	"fmt"
	"math"

	ftsverify "github.com/couchbase/n1fty/verify"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/search"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type IndexFtsSearch struct {
	base
	conn     *datastore.IndexConnection
	plan     *plan.IndexFtsSearch
	children []Operator
	keys     map[string]bool
	pool     bool
}

func NewIndexFtsSearch(plan *plan.IndexFtsSearch, context *Context) *IndexFtsSearch {
	rv := &IndexFtsSearch{}
	rv.plan = plan

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *IndexFtsSearch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexFtsSearch(this)
}

func (this *IndexFtsSearch) Copy() Operator {
	rv := &IndexFtsSearch{}
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexFtsSearch) PlanOp() plan.Operator {
	return this.plan
}

func (this *IndexFtsSearch) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		this.setExecPhase(FTS_SEARCH, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer this.notify()                          // Notify that I have stopped
		if !active {
			return
		}

		if this.plan.HasDeltaKeyspace() {
			defer func() {
				this.keys, this.pool = this.deltaKeyspaceDone(this.keys, this.pool)
			}()
			this.keys, this.pool = this.scanDeltaKeyspace(this.plan.Keyspace(), parent,
				FTS_SEARCH, context, this.plan.Covers())
		}

		this.conn = datastore.NewIndexConnection(context)
		defer this.conn.Dispose()  // Dispose of the connection
		defer this.conn.SendStop() // Notify index that I have stopped

		util.Fork(func(interface{}) {
			this.search(context, this.conn, parent)
		}, nil)

		ok := true
		var docs uint64 = 0

		var countDocs = func() {
			if docs > 0 {
				context.AddPhaseCount(FTS_SEARCH, docs)
			}
		}
		defer countDocs()

		// for right hand side of nested-loop join we don't want to include parent values
		// in the returned scope value
		scope_value := parent
		if this.plan.IsUnderNL() {
			scope_value = nil
		}

		outName := this.plan.SearchInfo().OutName()
		covers := this.plan.Covers()
		lc := len(covers)
		fc := this.plan.FilterCovers()

		for ok {
			entry, cont := this.getItemEntry(this.conn)
			if cont {
				if entry != nil {
					if _, sok := this.keys[entry.PrimaryKey]; !sok {
						av := this.newEmptyDocumentWithKey(entry.PrimaryKey, scope_value, context)
						if lc > 0 {
							for c, v := range fc {
								av.SetCover(c.Text(), v)
							}

							av.SetCover(covers[0].Text(), value.NewValue(true))
							av.SetCover(covers[1].Text(), value.NewValue(entry.PrimaryKey))
							smeta := entry.MetaData
							var score value.Value
							if smeta != nil {
								score, _ = smeta.Field("score")
							}

							if lc > 2 {
								av.SetCover(covers[2].Text(), score)
							}
							if lc > 3 {
								av.SetCover(covers[3].Text(), smeta)
							}
							av.SetField(this.plan.Term().Alias(), av)

							if context.UseRequestQuota() {
								err := context.TrackValueSize(av.Size())
								if err != nil {
									context.Error(err)
									av.Recycle()
									break
								}
							}
						}
						av.SetAttachment("smeta", map[string]interface{}{outName: entry.MetaData})
						av.SetBit(this.bit)
						ok = this.sendItem(av)
						docs++
						if docs > _PHASE_UPDATE_COUNT {
							context.AddPhaseCount(FTS_SEARCH, docs)
							docs = 0
						}
					}
				} else {
					ok = false
				}
			} else {
				return
			}
		}
	})
}

func (this *IndexFtsSearch) search(context *Context, conn *datastore.IndexConnection, parent value.Value) {
	defer context.Recover(nil) // Recover from any panic

	scanVector := context.ScanVectorSource().ScanVector(this.plan.Term().Namespace(), this.plan.Term().Path().Bucket())

	indexSearchInfo, err := this.planToSearchMapping(context, parent)
	index, ok := this.plan.Index().(datastore.FTSIndex)
	if err != nil || !ok {
		context.Error(errors.NewEvaluationError(err, "searchinfo"))
		conn.Sender().Close()
		return
	}

	consistency := context.ScanConsistency()
	if consistency == datastore.SCAN_PLUS && context.txContext != nil {
		consistency = datastore.UNBOUNDED
	}

	index.Search(context.RequestId(), indexSearchInfo, consistency, scanVector, conn)
}

func (this *IndexFtsSearch) planToSearchMapping(context *Context,
	parent value.Value) (indexSearchInfo *datastore.FTSSearchInfo, err error) {
	indexSearchInfo = &datastore.FTSSearchInfo{}

	psearchInfo := this.plan.SearchInfo()
	indexSearchInfo.Field, _, err = evalOne(psearchInfo.FieldName(), &this.operatorCtx, parent)
	if err != nil {
		return nil, err
	}

	indexSearchInfo.Query, _, err = evalOne(psearchInfo.Query(), &this.operatorCtx, parent)
	if err != nil {
		return nil, err
	}

	indexSearchInfo.Options, _, err = evalOne(psearchInfo.Options(), &this.operatorCtx, parent)
	if err != nil {
		return nil, err
	}

	if indexSearchInfo.Query == nil || (indexSearchInfo.Query.Type() != value.STRING &&
		indexSearchInfo.Query.Type() != value.OBJECT) {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Search() function Query parameter must be string or object.")))
	}

	if indexSearchInfo.Options != nil && indexSearchInfo.Options.Type() != value.OBJECT {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Search() function Options parameter must be object.")))
	}

	indexSearchInfo.Offset = evalLimitOffset(psearchInfo.Offset(), parent, int64(0), false, &this.operatorCtx)
	indexSearchInfo.Limit = evalLimitOffset(psearchInfo.Limit(), parent, math.MaxInt64, false, &this.operatorCtx)
	indexSearchInfo.Order = psearchInfo.Order()

	return
}

func (this *IndexFtsSearch) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

// send a stop/pause
func (this *IndexFtsSearch) SendAction(action opAction) {
	this.connSendAction(this.conn, action)
}

func (this *IndexFtsSearch) Done() {
	this.baseDone()
	this.keys, this.pool = this.deltaKeyspaceDone(this.keys, this.pool)
}

func SetSearchInfo(aliasMap map[string]string, item value.Value,
	context *opContext, exprs ...expression.Expression) error {
	var q, o value.Value
	var err error

	for _, expr := range exprs {
		if expr == nil {
			continue
		} else if sfn, ok := expr.(*search.Search); ok {
			if path, ok := aliasMap[sfn.KeyspaceAlias()]; ok {
				sfn.SetKeyspacePath(path)
			}

			// record error as part of search function so that we can raise error
			// only if search function is invoked
			var v datastore.Verify
			q, _, err = evalOne(sfn.Query(), context, item)
			if err != nil || q == nil || (q.Type() != value.STRING && q.Type() != value.OBJECT) {
				err = fmt.Errorf("%v function Query parameter must be string or object.", sfn)
			}

			if err == nil {
				o, _, err = evalOne(sfn.Options(), context, item)
				if err != nil || (o != nil && o.Type() != value.OBJECT) {
					err = fmt.Errorf("%v function Options parameter must be object.", sfn)
				}
			}

			if err == nil {
				v, err = ftsverify.NewVerify(sfn.KeyspacePath(), sfn.FieldName(),
					q, o, context.MaxParallelism())
			}

			sfn.SetVerify(v, err)
		} else if _, ok := expr.(expression.Subquery); !ok {
			SetSearchInfo(aliasMap, item, context, expr.Children()...)
		}
	}

	return nil
}
