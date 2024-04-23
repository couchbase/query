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
				isUnion := with.IsUnion()

				// track config options
				ilevel := int64(0)
				idoc := int64(0)

				implicitMaxDocs := int64(-1)
				if config.document == -1 && !context.UseRequestQuota() {
					implicitMaxDocs = _MAX_IMPLICIT_DOCS
				}

				// CYCLE CLAUSE
				var cycleFields expression.Expressions
				trackCycle := newwMap()

				if cycleFields = with.CycleFields(); cycleFields != nil {
					cycleFields = validateCycleFields(cycleFields)
				}

				finalRes := []interface{}{}
				workRes, ok := v.Actual().([]interface{})
				if !ok {
					context.Error(errors.NewExecutionInternalError("Anchor value is not an array"))
					this.notify()

					// MB-31605 have to start the child for the output and stop
					// operators to be set properly by sequences
					break
				}

				// cycle detection for anchor
				if cycleFields != nil {
					workRes = cycleRestrict(workRes, cycleFields, trackCycle, context)
				}

				// track duplicates for anchor
				if isUnion {
					workRes = removeDuplicates(workRes, trackUnion)
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

					if ilevel > config.level {
						// exit on level limit
						context.Infof("Reached %v recursion depth", ilevel)
						break
					}

					// exit on docs limit
					if config.document > -1 && idoc > config.document {
						finalRes = finalRes[:config.document]
						break
					} else if implicitMaxDocs > -1 && idoc > implicitMaxDocs {
						e = errors.NewRecursiveImplicitDocLimitError(with.Alias(), implicitMaxDocs)
						finalRes = finalRes[:implicitMaxDocs]
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
					if !ok {
						e = errors.NewExecutionInternalError("couldn't convert recursive result to array")
						break
					}

					// cycle detection for anchor
					if cycleFields != nil {
						workRes = cycleRestrict(workRes, cycleFields, trackCycle, context)
					}

					// update alias
					e = wv.SetField(with.Alias(), v)
					if e != nil {
						break
					}

					if isUnion {
						workRes = removeDuplicates(workRes, trackUnion)
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
	this.recycleBindings(false) // field values may still be referenced so don't recycle them
	return rv
}

func (this *With) Done() {
	this.baseDone()
	if this.child != nil {
		child := this.child
		this.child = nil
		child.Done()
	}
	this.recycleBindings(true)
}

func (this *With) recycleBindings(clearArray bool) {
	if this.wv == nil {
		return
	}

	m := this.wv.Fields()
	if m != nil {
		for _, v := range m {
			if val, ok := v.(value.Value); ok {
				// When possible provide GC hint for array elements...
				if clearArray == true && val.Type() == value.ARRAY {
					// NOTE: this affects the underlying type so references to the value are all affected too so we must be sure
					// we're done with it before doing this
					if arr, ok := val.Actual().([]interface{}); ok && arr != nil {
						clear(arr)
						arr = arr[0:0:0]
					}
				}
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
	m map[interface{}]interface{}
}

func newwMap() *wMap {
	return &wMap{m: make(map[interface{}]interface{})}
}

func (this *wMap) put(key, value interface{}) {
	b, _ := json.Marshal(key)
	this.m[string(b)] = value
}

func (this *wMap) get(key interface{}) (value interface{}, pres bool) {
	b, _ := json.Marshal(key)
	value, pres = this.m[string(b)]
	return
}

func (this *wMap) clear() {
	for k := range this.m {
		delete(this.m, k)
	}
	this.m = nil
}

func removeDuplicates(list []interface{}, set *wMap) []interface{} {
	newList := []interface{}{}
	for _, v := range list {
		if _, pres := set.get(v); !pres {
			//add as not seen before
			newList = append(newList, v)
			set.put(v, true)
		}
	}

	return newList
}

type ConfigOptions struct {
	level    int64 // exit on level N
	document int64 // exit on accumulating N docs
}

func processConfig(config value.Value) (*ConfigOptions, errors.Error) {
	configOptions := ConfigOptions{
		level:    _MAX_RECUR_DEPTH,
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

func validateCycleFields(cycle expression.Expressions) expression.Expressions {

	validateCycleFields := expression.Expressions{}
	for _, cycleFieldExpr := range cycle {

		switch c := cycleFieldExpr.(type) {
		case *expression.Identifier:
			validateCycleFields = append(validateCycleFields, c)
		case *expression.Field:
			// allow nested fieldnames
			validateCycleFields = append(validateCycleFields, c)
		}
	}

	return validateCycleFields
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

func cycleRestrict(items []interface{}, cycleFields expression.Expressions, trackCycle *wMap, context *Context) []interface{} {
	result := []interface{}{}

	for _, item := range items {
		hv, err := hopVal(value.NewValue(item), cycleFields, context)
		if err != nil {
			//skip item for cycle detection
			continue
		} else {
			if _, pres := trackCycle.get(hv); !pres {
				// if not present add
				result = append(result, item)
				trackCycle.put(hv, true)
			}
		}
	}

	return result
}
