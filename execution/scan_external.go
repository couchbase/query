//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

var _EXTERNALSCAN_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_EXTERNALSCAN_OP_POOL, func() interface{} {
		return &ExternalScan{}
	})
}

const _DEF_RESULT_CACHE_SIZE = 64

// ExternalScan scans external collections (e.g., Iceberg tables).
type ExternalScan struct {
	base
	plan       *plan.ExternalScan
	conn       *datastore.IndexConnection
	scanReport *datastore.IndexScanReport
	params     *datastore.ExternalScanParams // nil until first scan; shared via Copy()
	results    []interface{}
}

type externalScanDesc struct {
	scan         *ExternalScan
	context      *Context
	inlineFilter expression.Expression
	parent       value.Value
}

func scanExternalFork(p interface{}) {
	d := p.(externalScanDesc)
	d.scan.scan(d.inlineFilter, d.context, d.parent, d.scan.conn)
}

func NewExternalScan(plan *plan.ExternalScan, context *Context) *ExternalScan {
	rv := _EXTERNALSCAN_OP_POOL.Get().(*ExternalScan)
	rv.plan = plan
	newRedirectBase(&rv.base, context)
	rv.phase = EXTERNAL_SCAN
	rv.output = rv
	rv.scanReport = datastore.NewIndexScanReport()
	return rv
}

func (this *ExternalScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExternalScan(this)
}

func (this *ExternalScan) Copy() Operator {
	rv := _EXTERNALSCAN_OP_POOL.Get().(*ExternalScan)
	rv.plan = this.plan
	rv.params = this.params
	rv.scanReport = this.scanReport
	this.base.copy(&rv.base)
	return rv
}

func (this *ExternalScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *ExternalScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		this.setExecPhaseWithAgg(this.Phase(), context)
		defer this.cleanup(context)
		if !active {
			return
		}
		this.conn = datastore.NewIndexConnection(context)
		this.conn.SetIndexScanReport(this.scanReport)
		defer this.conn.Dispose() // Dispose of the connection
		defer this.conn.WaitScanReport(context.ScanReportWait())
		defer this.conn.SendStop() // Notify index that I have stopped

		useCache := this.plan.IsUnderNL()
		alias := this.plan.Term().Alias()

		if this.plan.CountOnly() {
			util.Fork(scanExternalFork, externalScanDesc{this, context, nil, parent})

			// The datastore sends exactly one entry carrying the total count
			// (see ExternalScanParams.CountOnly); there are no per-row results
			// to iterate or wrap with the keyspace alias.
			var count value.Value = value.ZERO_VALUE
			if entry, cont := this.getItemEntry(this.conn); cont && entry != nil {
				count = entry.EntryKey[0]
			}

			cv := value.NewScopeValue(nil, parent)
			av := value.NewAnnotatedValue(cv)
			av.SetAttachment(value.ATT_COUNT, count)
			if context.UseRequestQuota() {
				if err := context.TrackValueSize(av.Size()); err != nil {
					context.Error(err)
					av.Recycle()
					return
				}
			}
			this.sendItem(av)
			return
		}

		// use cached results if available
		if useCache && this.results != nil {
			for _, act := range this.results {
				actv := value.NewScopeValue(make(map[string]interface{}), parent)
				actv.SetField(alias, act)
				av := value.NewAnnotatedValue(actv)
				av.SetId("")

				if context.UseRequestQuota() {
					err := context.TrackValueSize(av.Size())
					if err != nil {
						context.Error(err)
						av.Recycle()
						return
					}
				}
				if !this.sendItem(av) {
					av.Recycle()
					break
				}
			}
			return
		}

		// Replace named/positional parameters in filter if present
		var filter, externalFilter expression.Expression
		if this.plan.Filter() != nil {
			filter = this.plan.Filter()
			if len(context.namedArgs) > 0 || len(context.positionalArgs) > 0 {
				var err error
				filter, err = plannerbase.ReplaceParameters(filter,
					context.namedArgs, context.positionalArgs)
				if err != nil {
					context.Error(errors.NewEvaluationWithCauseError(err, "replace query parameters"))
					return
				}
			}
			filter.EnableInlistHash(&this.operatorCtx)
			defer filter.ResetMemory(&this.operatorCtx)
		}
		externalFilter = context.getExternalFilters(alias)
		if externalFilter != nil {
			defer context.clearExternalFilters(alias)
			if filter == nil {
				filter = externalFilter
			} else {
				filter = expression.NewAnd(filter, externalFilter)
			}
		}
		util.Fork(scanExternalFork, externalScanDesc{this, context, filter, parent})

		ok := true
		var docs uint64 = 0

		var countDocs = func() {
			if docs > 0 {
				context.AddPhaseCountWithAgg(this.Phase(), docs)
			}
		}
		defer countDocs()

		var results []interface{}
		if useCache {
			this.results = nil
			results = make([]interface{}, 0, _DEF_RESULT_CACHE_SIZE)
		}
		for ok {
			entry, cont := this.getItemEntry(this.conn)
			if cont {
				if entry != nil {
					this.addInDocs(1)
					act := entry.EntryKey[0]
					actv := value.NewScopeValue(make(map[string]interface{}), parent)
					actv.SetField(alias, act)
					av := value.NewAnnotatedValue(actv)
					av.SetId(entry.PrimaryKey)
					if useCache {
						results = append(results, act)
					}
					if context.UseRequestQuota() {
						err := context.TrackValueSize(av.Size())
						if err != nil {
							context.Error(err)
							av.Recycle()
							ok = false
							break
						}
					}

					av.SetBit(this.bit)
					ok = this.sendItem(av)
					docs++
					if docs > _PHASE_UPDATE_COUNT {
						context.AddPhaseCountWithAgg(this.Phase(), docs)
						docs = 0
					}
				} else {
					ok = false
				}
			} else {
				break
			}

		}
		this.results, results = results, nil
	})
}

func (this *ExternalScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]any) {
		this.marshalTimes(r)
		if this.scanReport != nil {
			if rv := this.scanReport.GetScanReport(); len(rv) > 0 {
				stats, ok := r["#stats"].(map[string]interface{})
				if !ok {
					stats = make(map[string]interface{}, len(rv))
					r["#stats"] = stats
				}
				for k, v := range rv {
					stats[k] = v
				}
			}
		}
	})
	return json.Marshal(r)
}

func (this *ExternalScan) Done() {
	this.baseDone()
	if this.isComplete() {
		this.params = nil
		_EXTERNALSCAN_OP_POOL.Put(this)
	}
}

func (this *ExternalScan) scan(filter expression.Expression, context *Context, parent value.Value, conn *datastore.IndexConnection) {
	defer context.Recover(nil)

	if this.params == nil {
		var snapshotId, snapshotTimestamp string
		if this.plan.SnapshotIdExpr() != nil {
			val, err := this.plan.SnapshotIdExpr().Evaluate(nil, context)
			if err != nil {
				context.Error(errors.NewEvaluationWithCauseError(err, "SNAPSHOT expression"))
				conn.Sender().Close()
				return
			}
			if val != nil && val.Type() != value.MISSING && val.Type() != value.NULL {
				snapshotId = val.ToString()
			}
		}
		if this.plan.SnapshotTimestampExpr() != nil {
			val, err := this.plan.SnapshotTimestampExpr().Evaluate(nil, context)
			if err != nil {
				context.Error(errors.NewEvaluationWithCauseError(err, "TIMESTAMP expression"))
				conn.Sender().Close()
				return
			}
			if val != nil && val.Type() != value.MISSING && val.Type() != value.NULL {
				snapshotTimestamp = val.ToString()
			}
		}

		alias := ""
		if this.plan.Term() != nil {
			alias = this.plan.Term().As()
		}
		projection := this.plan.EarlyProjection()
		var resultObject map[string]any
		if len(projection) > 0 {
			resultObject = value.BuildObjectFromDottedPaths(projection)
		}

		this.params = &datastore.ExternalScanParams{
			RequestId:         context.RequestId(),
			Filter:            filter,
			SnapshotId:        snapshotId,
			SnapshotTimestamp: snapshotTimestamp,
			Alias:             alias,
			Projection:        projection,
			ResultObject:      resultObject,
			ErrTemplate:       make(map[string]any),
			CountOnly:         this.plan.CountOnly(),
			ScanStats:         make(map[string]any),
		}
	}

	this.params.Parent = parent

	this.plan.Keyspace().ExternalScan(this.params, &this.operatorCtx, conn)

	if len(this.params.ScanStats) > 0 {
		conn.AggregateScanReport(aggregateExternalScanStats, this.params.ScanStats)
		// this.params (and its ScanStats map) is reused across invocations on the same
		// operator instance (e.g. once per outer row under a nested-loop join when results
		// aren't cached); clear it so conditionally-set keys from this invocation don't
		// leak into, or get double-counted on, the next one.
		clear(this.params.ScanStats)
	}
}

// aggregateExternalScanStats merges per-invocation ExternalScanParams.ScanStats into the
// operator's shared scan report: numeric counters are summed (an operator instance under a
// nested-loop join may invoke the datastore scan more than once), everything else is
// overwritten since it's expected to be identical across invocations of the same scan.
func aggregateExternalScanStats(v1, v2 map[string]interface{}) {
	for k, v := range v2 {
		if i2, ok := v.(int64); ok {
			if i1, ok := v1[k].(int64); ok {
				v1[k] = i1 + i2
				continue
			}
		}
		v1[k] = v
	}
}
