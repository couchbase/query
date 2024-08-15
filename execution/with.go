//  Copyright 2018-Present Couchbase, Inc.
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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

const _MAX_RECUR_DEPTH = int64(100)
const _MAX_IMPLICIT_DOCS = int64(10000)

type With struct {
	base
	plan  *plan.With
	child Operator
	wv    value.AnnotatedValue
}

func NewWith(plan *plan.With, context *Context, child Operator) *With {
	rv := &With{
		plan:  plan,
		child: child,
	}

	newBase(&rv.base, context)
	rv.base.setInline()
	rv.output = rv
	return rv
}

func (this *With) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWith(this)
}

func (this *With) Copy() Operator {
	rv := &With{plan: this.plan, child: this.child.Copy()}
	this.base.copy(&rv.base)
	return rv
}

func (this *With) PlanOp() plan.Operator {
	return this.plan
}

func (this *With) Child() Operator {
	return this.child
}

func (this *With) IsParallel() bool {
	return this.child.IsParallel()
}

func (this *With) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.SetKeepAlive(1, context) // terminate early
		this.switchPhase(_EXECTIME)
		this.setExecPhase(RUN, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time

		if !active || !context.assert(this.child != nil, "With has no child") {
			this.notify()
			this.fail(context)
			return
		}

		this.child.SetInput(this.input)
		this.child.SetOutput(this.output)
		this.child.SetStop(nil)
		this.child.SetParent(this)
		this.stashOutput()

		var wv value.AnnotatedValue

		if parent != nil {
			wv = value.NewAnnotatedValue(parent.Copy())
		} else {
			wv = value.NewAnnotatedValue(make(map[string]interface{}, 1))
		}

		withs := this.plan.Bindings()

		for _, with := range withs.Bindings() {
			if !this.isRunning() {
				this.notify()
				break
			}
			v, e := with.Expression().Evaluate(wv, &this.operatorCtx)
			if e != nil {
				context.Error(errors.NewEvaluationError(e, "WITH"))
				this.notify()

				// MB-31605 have to start the child for the output and stop
				// operators to be set properly by sequences
				break
			}
			wv.SetField(with.Alias(), v)
			if with.IsRecursive() {

				// read options
				config, confErr := processConfig(with.Config())
				if confErr != nil {
					context.Error(confErr)
					this.notify()

					// MB-31605 have to start the child for the output and stop
					// operators to be set properly by sequences
					break
				}

				// UNION: don't allow duplicates
				trackUnion := newwMap()

				// track config options
				ilevel := int64(0)
				idoc := int64(0)

				implicitMaxDocs := int64(-1)
				if config.document == -1 && !context.UseRequestQuota() {
					implicitMaxDocs = _MAX_IMPLICIT_DOCS
				}

				implicitMaxDepth := int64(-1)
				if config.level == -1 && !context.UseRequestQuota() {
					implicitMaxDepth = _MAX_RECUR_DEPTH
				}

				// CYCLE CLAUSE
				trackCycle := newwMap()

				finalRes := []interface{}{}
				workRes, ok := v.Actual().([]interface{})
				if !ok {
					context.Error(errors.NewExecutionInternalError("Anchor value is not an array"))
					this.notify()

					// MB-31605 have to start the child for the output and stop
					// operators to be set properly by sequences
					break
				}

				// dedup+cycle detection for anchor
				workRes, e = dedupAndCycleRestrict(workRes, with.CycleFields(), with.IsUnion(), trackCycle, trackUnion, context)
				if e != nil {
					context.Error(errors.NewEvaluationError(e, "WITH"))
					this.notify()

					// MB-31605 have to start the child for the output and stop
					// operators to be set properly by sequences
					break
				}

				for {
					if !this.isRunning() {
						this.notify()
						break
					}
					if len(workRes) == 0 {
						// naive exit
						break
					}

					if config.level > -1 && ilevel > config.level {
						// exit on level limit
						break
					} else if implicitMaxDepth > -1 && ilevel > implicitMaxDepth {
						// set to finalRes
						wv.SetField(with.Alias(), value.NewValue(finalRes))
						context.Warning(errors.NewRecursiveImplicitDepthLimitError(with.Alias(), implicitMaxDepth))
						break
					}

					// exit on docs limit
					if config.document > -1 && idoc > config.document {
						finalRes = finalRes[:config.document]
						break
					} else if implicitMaxDocs > -1 && idoc > implicitMaxDocs {
						finalRes = finalRes[:implicitMaxDocs]
						// set to finalRes
						wv.SetField(with.Alias(), value.NewValue(finalRes))
						context.Warning(errors.NewRecursiveImplicitDocLimitError(with.Alias(), implicitMaxDocs))
						break
					}

					// append workres to final
					finalRes = append(finalRes, workRes...)

					// update doc count
					idoc += int64(len(workRes))
					// update level count
					ilevel += int64(1)

					v, e = with.RecursiveExpression().Evaluate(wv, &this.operatorCtx)
					if e != nil {
						break
					}

					// create workRes
					workRes, ok = v.Actual().([]interface{})
					orgLen := len(workRes)
					if !ok {
						e = errors.NewExecutionInternalError("couldn't convert recursive result to array")
						break
					}

					workRes, e = dedupAndCycleRestrict(workRes, with.CycleFields(), with.IsUnion(), trackCycle, trackUnion, context)
					if e != nil {
						break
					}

					if len(workRes) < orgLen {
						v = value.NewValue(workRes)
					}

					// update alias
					e = wv.SetField(with.Alias(), v)
					if e != nil {
						break
					}
				}

				// clean up
				trackUnion.clear()
				trackCycle.clear()

				if e != nil {
					context.Error(errors.NewEvaluationError(e, "WITH"))
					this.notify()

					// MB-31605 have to start the child for the output and stop
					// operators to be set properly by sequences
					break
				}
				// set to finalRes
				wv.SetField(with.Alias(), value.NewValue(finalRes))
			}
		}
		this.wv = wv // keep a copy for later recycling

		this.fork(this.child, context, wv)
	})
}

func (this *With) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	r["~child"] = this.child
	return json.Marshal(r)
}

func (this *With) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*With)
	this.child.accrueTimes(copy.child)
}

func (this *With) SendAction(action opAction) {
	this.baseSendAction(action)
	child := this.child
	if child != nil {
		child.SendAction(action)
	}
}

func (this *With) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.child != nil {
		rv = this.child.reopen(context)
	}
	this.recycleBindings()
	return rv
}

func (this *With) Done() {
	this.baseDone()
	if this.child != nil {
		child := this.child
		this.child = nil
		child.Done()
	}
	this.recycleBindings()
}

func (this *With) recycleBindings() {
	if this.wv == nil {
		return
	}

	m := this.wv.Fields()
	if m != nil {
		for _, v := range m {
			if val, ok := v.(value.Value); ok {
				// double recycle here to account for SetField's tracking
				val.Recycle()
				val.Recycle()
			}
		}
	}
	this.wv.Recycle()
	this.wv = nil
}

type wMap struct {
	m map[string]interface{}
}

func newwMap() *wMap {
	return &wMap{m: make(map[string]interface{})}
}

func (this *wMap) clear() {
	for k := range this.m {
		delete(this.m, k)
	}
	this.m = nil
}

type ConfigOptions struct {
	level    int64 // exit on level N
	document int64 // exit on accumulating N docs
}

func processConfig(config value.Value) (*ConfigOptions, errors.Error) {
	configOptions := ConfigOptions{
		level:    -1,
		document: -1,
	}

	if config == nil {
		return &configOptions, nil
	}

	if config.Type() != value.OBJECT {
		return nil, errors.NewExecutionInternalError(fmt.Sprintf("Configuration (%v) is not an object", config.Type().String()))
	}

	for fieldName := range config.Fields() {
		fv, _ := config.Field(fieldName)

		if fv.Type() != value.NUMBER {
			// all options are numeric for now
			return nil, errors.NewExecutionInternalError(fmt.Sprintf("Configuration options must be numeric ('%v' is %v)",
				fieldName, fv.Type().String()))
		}

		v, ok := fv.Actual().(float64)
		if !ok {
			return nil, errors.NewExecutionInternalError(fmt.Sprintf("Value for '%v' is invalid", fieldName))
		}

		switch fieldName {
		case "levels":
			configOptions.level = int64(v)
		case "documents":
			configOptions.document = int64(v)
		default:
			return nil, errors.NewInvalidConfigOptions(fieldName)
		}
	}
	return &configOptions, nil
}

func hopVal(item value.Value, cycleFields expression.Expressions, context *Context) (map[string]interface{}, error) {
	val := map[string]interface{}{}
	for _, exp := range cycleFields {
		fval, err := exp.Evaluate(item, context)

		if err != nil {
			// skip cycle detection for this doc
			return nil, err
		}

		if fval.Type() != value.MISSING {
			val[exp.String()] = fval.Actual()
		}
	}
	return val, nil
}

func dedupAndCycleRestrict(items []interface{}, cycleFields expression.Expressions, isUnion bool,
	trackCycle *wMap, trackSet *wMap, context *Context) ([]interface{}, errors.Error) {
	if !isUnion && cycleFields == nil {
		return items, nil
	}

	newItemListEnd := 0

	for _, item := range items {
		keep := true
		if isUnion {
			b, _ := json.Marshal(item)
			if _, pres := trackSet.m[string(b)]; pres {
				keep = false
			} else {
				trackSet.m[string(b)] = true
			}
		}

		if cycleFields != nil {
			v := value.NewValue(item)
			hv, err := hopVal(v, cycleFields, context)
			if err != nil {
				// keep = false
				return nil, errors.NewExecutionInternalError(fmt.Sprintf("failed to create encoded value from "+
					"cycle fields provided for item: %v", item))
			}

			if keep {
				b, _ := json.Marshal(hv)
				if _, pres := trackCycle.m[string(b)]; pres {
					keep = false
				} else {
					trackCycle.m[string(b)] = true
				}
			}
		}

		if keep {
			items[newItemListEnd] = item
			newItemListEnd++
		} else {
			discardv := value.NewValue(item)
			if context.UseRequestQuota() {
				context.ReleaseValueSize(discardv.Size())
			}
			// This is objectValue so, recycle doesn't do anything
			discardv.Recycle()
		}
	}

	return items[:newItemListEnd:newItemListEnd], nil
}
