//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"encoding/json"
	"strconv"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

const _MAX_RECUR_DEPTH = int64(100)

type RecursiveCte struct {
	FunctionBase
}

func NewRecursiveCte(operands ...Expression) Function {
	rv := &RecursiveCte{}
	rv.Init("recursive_cte", operands...)

	rv.setVolatile()
	rv.expr = rv
	return rv
}

func (this *RecursiveCte) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RecursiveCte) Type() value.Type { return value.ARRAY }

func (this *RecursiveCte) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false

	anchorClause, err := this.operands[0].Evaluate(item, context)

	if err != nil {
		return nil, err
	} else if anchorClause.Type() == value.MISSING {
		missing = true
	} else if anchorClause.Type() != value.STRING {
		null = true
	}

	recursiveClause, err := this.operands[1].Evaluate(item, context)

	if err != nil {
		return nil, err
	} else if recursiveClause.Type() == value.MISSING {
		missing = true
	} else if recursiveClause.Type() != value.STRING {
		null = true
	}

	if missing {
		return value.MISSING_VALUE, nil
	}

	if null {
		return value.NULL_VALUE, nil
	}

	// config controls
	configRv := NewConfig()

	if len(this.operands) > 2 {
		// Config Argument Passed
		confRv, _ := this.operands[2].Evaluate(item, context)

		if confRv.Type() == value.OBJECT {
			// config controls received
			configRv.getConfig(confRv, context)
		}
	}

	// cast context to ParkableExecutePreparedContex inorder to use prepareStatements
	pctx, ok := context.(ParkableExecutePreparedContext)
	if !ok {
		return nil, errors.NewEvaluationError(nil, "casting context to ParkableExecutePreparedContext failed")
	}

	// prepare anchorClause
	anchorPlan, errPrepAnchor := pctx.PrepareStatementExt(anchorClause.ToString())
	if errPrepAnchor != nil {
		return nil, errPrepAnchor
	}

	// execute anchor - get root docs
	anchorRv, _, errExecAnchor := pctx.ExecutePreparedExt(anchorPlan, configRv.namedArgs, configRv.posArgs)
	if errExecAnchor != nil {
		return nil, errExecAnchor
	}

	// append recursive results into anchorRvA , for next level's recursive phase
	anchorRvA, ok := anchorRv.Actual().([]interface{})
	if !ok {
		// anchor is not a collection type so can't type cast
		return nil, errors.NewExecutionInternalError("anchor results is not an array")
	}

	if len(anchorRvA) == 0 {
		// early exit as no docs returned from anchor
		return value.NewValue(anchorRvA), nil
	}

	// add visit entries for anchor docs
	var hashTab map[string]bool
	var hashVal map[string]interface{}
	if len(configRv.cycleDetect) > 0 {

		hashTab = make(map[string]bool, 1024)
		hashVal = make(map[string]interface{})

		// clean up
		defer func(hashTab map[string]bool) {
			for key := range hashTab {
				delete(hashTab, key)
			}
		}(hashTab)

		anchorRvA = this.checkCycle(hashTab, hashVal, configRv.cycleDetect, anchorRvA, context)

	}

	// prepare recursiveClause
	recursivePlan, errPrepRecursive := pctx.PrepareStatementExt(recursiveClause.ToString())
	if errPrepRecursive != nil {
		return nil, errPrepRecursive
	}

	// Start recursive phase
	var iLevel int64

	var nextLevel int64 = int64(len(anchorRvA))
	var currLevel int64
	for currLevel < nextLevel {

		if iLevel >= configRv.levelLimit {
			context.Infof("Reached %v recursion depth", iLevel)
			break
		}

		if configRv.docsLimit >= 0 && currLevel >= configRv.docsLimit {
			// trim docs of last level to meet docsLimit
			anchorRvA = anchorRvA[:configRv.docsLimit]
			break
		}

		workDocs := anchorRvA[currLevel:nextLevel]

		// set argument $anchor as workDocs for recursiveClause
		configRv.namedArgs["anchor"] = value.NewValue(workDocs)

		// get children docs for current parent(passed as $anchor expr)
		recursiveRv, _, errExecRecursive := pctx.ExecutePreparedExt(recursivePlan, configRv.namedArgs, configRv.posArgs)
		if errExecRecursive != nil {
			return nil, errExecRecursive
		}

		recursiveRvA, ok := recursiveRv.Actual().([]interface{})
		if !ok {
			continue
		}

		// eliminate cycles
		if len(configRv.cycleDetect) > 0 {
			recursiveRvA = this.checkCycle(hashTab, hashVal, configRv.cycleDetect, recursiveRvA, context)
		}

		// add recursive result docs to final result
		anchorRvA = append(anchorRvA, recursiveRvA...)

		currLevel = nextLevel
		nextLevel = int64(len(anchorRvA))

		iLevel++

	}

	if configRv.explainF {
		this.getExplain(anchorPlan, recursivePlan, context)
	}

	return value.NewValue(anchorRvA), nil

}

func (this *RecursiveCte) MinArgs() int { return 2 }

func (this *RecursiveCte) MaxArgs() int { return 3 }

// factory
func (this *RecursiveCte) Constructor() FunctionConstructor {
	return NewRecursiveCte
}

func (this *RecursiveCte) getExplain(anchorPlan interface{}, recursivePlan interface{}, context Context) {

	ap, _ := json.Marshal(anchorPlan)
	rp, _ := json.Marshal(recursivePlan)
	context.Infof("Explain for anchor clause:")
	context.Infof("%s", string(ap))
	context.Infof("Explain for recursive clause:")
	context.Infof("%s", string(rp))
}

// trim collection based on cycle detection
func (this *RecursiveCte) checkCycle(hashTab map[string]bool, hashValMap map[string]interface{}, cycleFields Expressions, objects []interface{}, context Context) []interface{} {
	noCycleObjects := []interface{}{}

	for _, object := range objects {
		objv := value.NewValue(object)

		hashErr := this.hashVal(objv, hashValMap, cycleFields, context)
		if hashErr != nil {
			continue
		}

		if hashEntry, err := json.Marshal(hashValMap); err == nil {
			// add doc only if not already present
			if _, ok := hashTab[string(hashEntry)]; !ok {
				hashTab[string(hashEntry)] = true
				noCycleObjects = append(noCycleObjects, object)
			}
		}
	}

	return noCycleObjects
}

// given a cycleFields:list of identifiers/field names, modifies hashval map for new values
func (this *RecursiveCte) hashVal(item value.Value, hashValMap map[string]interface{}, cycleFields Expressions, context Context) error {

	for pos, exp := range cycleFields {
		fval, err := exp.Evaluate(item, context)

		if err != nil {
			// skip cycle detection for this doc
			return err
		}

		key := "f" + strconv.Itoa(pos)
		if fval.Type() == value.MISSING {
			delete(hashValMap, key)
		} else {
			hashValMap[key] = fval.Actual()
		}
	}

	return nil
}

func (this *Config) validateCycleFields(cycleFieldsArg value.Value, context Context) {
	cycleDetectA, ok := cycleFieldsArg.Actual().([]interface{})

	validCycleFields := Expressions{}
	if !ok {
		return
	}

	for _, fv := range cycleDetectA {
		fvV := value.NewValue(fv)
		if fvV.Type() != value.STRING {
			continue
		}

		fvr, err := context.Parse(fvV.ToString())
		if err != nil || fvr == nil {
			continue
		}

		// cast to expression
		fExp, ok := fvr.(Expression)
		if ok {
			switch f := fExp.(type) {
			case *Identifier:
				// allow field names as indentifier
				validCycleFields = append(validCycleFields, f)
			case *Field:
				// allow nested field names as
				validCycleFields = append(validCycleFields, f)
			}
		}
	}

	this.cycleDetect = validCycleFields
}

type Config struct {
	levelLimit  int64
	docsLimit   int64
	posArgs     value.Values
	namedArgs   map[string]value.Value
	explainF    bool
	cycleDetect Expressions
}

func NewConfig() *Config {
	// default config
	conRv := Config{
		levelLimit: _MAX_RECUR_DEPTH,
		docsLimit:  -1,
		namedArgs:  make(map[string]value.Value), // init for $anchor expression, in join (recursiveClause)
	}
	return &conRv
}

func (this *Config) getConfig(confVal value.Value, context Context) {

	if confVal == nil {
		// default configRv
		return
	}

	if levelV, levelFlag := confVal.Field("levels"); levelFlag {
		levelA, ok := levelV.Actual().(float64)
		if ok {
			this.levelLimit = int64(levelA)
		}
	}

	if docsV, docsFlag := confVal.Field("documents"); docsFlag {
		docsA, ok := docsV.Actual().(float64)
		if ok {
			this.docsLimit = int64(docsA)
		}
	}

	if explainV, explainFlag := confVal.Field("explain"); explainFlag == true {
		explainA, ok := explainV.Actual().(bool)

		if ok {
			this.explainF = explainA
		}
	}

	if argumentsV, argumentsFlag := confVal.Field("arguments"); argumentsFlag == true {
		switch argT := argumentsV.Actual().(type) {
		case []interface{}:
			this.posArgs = make(value.Values, len(argT))

			for id, v := range argT {
				this.posArgs[id] = value.NewValue(v)
			}

		case map[string]interface{}:
			for k, v := range argT {
				this.namedArgs[k] = value.NewValue(v)
			}
		}
	}

	if cycleDetectV, cycleDetectFlag := confVal.Field("cycle"); cycleDetectFlag {
		this.validateCycleFields(cycleDetectV, context)
	}
}
