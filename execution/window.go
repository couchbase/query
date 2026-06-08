//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

/*
Window operator specific information
*/

type WindowAggregate struct {
	base
	plan         *plan.WindowAggregate
	values       value.AnnotatedValues
	pbyValues    value.Values
	pby          expression.Expressions
	oby          algebra.SortTerms
	pbyTerms     []string
	obyTerms     []string
	obyDesc      []bool
	cItem        int64
	nItems       int64
	aggs         []*AggregateInfo
	newPartition bool
	flags        uint32
}

const (
	_AI_RERANK = 1 << iota
	_AI_SENTIMENT
)

var _AI_ACTIONS = map[string]int{
	"rerank":    _AI_RERANK,
	"sentiment": _AI_SENTIMENT,
}

type AiOptions struct {
	uriObj    *url.URL
	query     string
	cred_id   string
	action    int
	batchSize int64
	model     string
	header    map[string]interface{}
}

/*
Aggregate specific information
*/

type AggregateInfo struct {
	agg               algebra.Aggregate
	id                string
	wTerm             *algebra.WindowTerm
	once              bool
	incremental       bool
	newCollationValue bool
	val               value.Value
	cumVal            value.Value
	sWindowVal        value.Value
	eWindowVal        value.Value
	dupsPreceding     int64
	dupsFollowing     int64
	obyValues         value.Values
	repeats           value.Value
	preVal            value.Value
	options           *AiOptions
	aiValues          value.Values
	preAgg            algebra.Aggregate
	flags             uint32
}

/*
Window frame information
*/
type excludeFrame struct {
	sIndex int64
	eIndex int64
}

type windowFrame struct {
	empty   bool
	sIndex  int64
	cIndex  int64
	eIndex  int64
	exclude []*excludeFrame
}

// Constants for Window operator
const (
	_WINDOW_RELEASE_CURRENTROW = 1 << iota
)

// constants for aggregate information
const (
	_WINDOW_ROW_NUMBER = 1 << iota
	_WINDOW_RANK
	_WINDOW_DENSE_RANK
	_WINDOW_PERCENT_RANK
	_WINDOW_RATIO_TO_REPORT
	_WINDOW_CUME_DIST
	_WINDOW_NTILE
	_WINDOW_FIRST_VALUE
	_WINDOW_LAST_VALUE
	_WINDOW_NTH_VALUE
	_WINDOW_LAG
	_WINDOW_LEAD
	_WINDOW_NOEQUAL_ROWS
	_WINDOW_FL_DUPLICATES
	_WINDOW_AICOMPUTE
	_WINDOW_AIFULLSET
	_WINDOW_AIRERANK
)

var _AGG_FLAGS = map[string]uint32{
	"row_number":      _WINDOW_ROW_NUMBER | _WINDOW_NOEQUAL_ROWS,
	"rank":            _WINDOW_RANK | _WINDOW_NOEQUAL_ROWS,
	"dense_rank":      _WINDOW_DENSE_RANK | _WINDOW_NOEQUAL_ROWS,
	"percent_rank":    _WINDOW_PERCENT_RANK | _WINDOW_NOEQUAL_ROWS,
	"cume_dist":       _WINDOW_CUME_DIST,
	"ratio_to_report": _WINDOW_RATIO_TO_REPORT,
	"ntile":           _WINDOW_NTILE | _WINDOW_NOEQUAL_ROWS,
	"first_value":     _WINDOW_FIRST_VALUE,
	"last_value":      _WINDOW_LAST_VALUE,
	"nth_value":       _WINDOW_NTH_VALUE,
	"lag":             _WINDOW_LAG | _WINDOW_NOEQUAL_ROWS,
	"lead":            _WINDOW_LEAD | _WINDOW_NOEQUAL_ROWS,
	"ai_compute":      _WINDOW_AICOMPUTE,
	"ai_rerank":       _WINDOW_AIRERANK,
}

const _WINDOW_CAP = 512

var _WINDOW_POOL = value.NewAnnotatedPool(_WINDOW_CAP)

func NewWindowAggregate(plan *plan.WindowAggregate, context *Context) *WindowAggregate {
	rv := &WindowAggregate{
		plan:         plan,
		values:       _WINDOW_POOL.Get(),
		newPartition: true,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *WindowAggregate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWindowAggregate(this)
}

func (this *WindowAggregate) Copy() Operator {
	rv := &WindowAggregate{
		plan:         this.plan,
		values:       _WINDOW_POOL.Get(),
		newPartition: true,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *WindowAggregate) PlanOp() plan.Operator {
	return this.plan
}

func (this *WindowAggregate) addFlags(flags uint32) {
	this.flags |= flags
}

func (this *WindowAggregate) removeFlags(flags uint32) {
	this.flags &^= flags
}

func (this *WindowAggregate) hasFlags(flags uint32) bool {
	return (this.flags & flags) != 0
}

/*
Window operator initial setup information
*/
func (this *WindowAggregate) setupTerms(parent value.Value) bool {
	largestOrderAgg := this.plan.Aggregates()[0]
	this.aggs = make([]*AggregateInfo, 0, len(this.plan.Aggregates()))
	this.addFlags(_WINDOW_RELEASE_CURRENTROW)

	/*
	   Setup each aggregate information.
	   Find largest ORDER BY terms aggregate. (every aggregate has same PARTITION BY terms)
	*/

	for _, agg := range this.plan.Aggregates() {
		wTerm := agg.WindowTerm()
		flags, _ := _AGG_FLAGS[agg.Name()]

		// check and reset _WINDOW_RELEASE_CURRENTROW
		if !algebra.AggregateHasProperty(agg.Name(), algebra.AGGREGATE_WINDOW_RELEASE_CURRENTROW) {
			this.removeFlags(_WINDOW_RELEASE_CURRENTROW)
		}

		// setup aggregate information.
		aInfo := &AggregateInfo{agg: agg, id: agg.String(), wTerm: wTerm, flags: flags}
		err := aInfo.setOnce(&this.operatorCtx, parent)
		if err != nil {
			this.operatorCtx.Fatal(errors.NewWindowEvaluationError(err, "Error inital setup"))
			return false
		}
		this.aggs = append(this.aggs, aInfo)

		// find largest ORDER BY terms aggregate
		if wTerm.OrderBy() != nil && (largestOrderAgg.WindowTerm().OrderBy() == nil ||
			len(wTerm.OrderBy().Terms()) > len(largestOrderAgg.WindowTerm().OrderBy().Terms())) {
			largestOrderAgg = agg
		}
	}

	// Setup the PBY terms info
	wTerm := largestOrderAgg.WindowTerm()
	this.pby = wTerm.PartitionBy()

	if len(this.pby) > 0 {
		this.pbyTerms = make([]string, len(this.pby))
		this.pbyValues = make(value.Values, len(this.pby))
		for i, expr := range this.pby {
			this.pbyTerms[i] = expr.String()
		}
	}

	// Setup the OBY terms info
	if wTerm.OrderBy() != nil {
		this.oby = wTerm.OrderBy().Terms()
		this.obyTerms = make([]string, len(this.oby))
		this.obyDesc = make([]bool, len(this.oby))
		for i, t := range this.oby {
			this.obyTerms[i] = t.Expression().String()
			this.obyDesc[i] = t.Descending(parent, &this.operatorCtx)
		}
	}

	return true
}

func (this *AggregateInfo) addFlags(flags uint32) {
	this.flags |= flags
}

func (this *AggregateInfo) hasFlags(flags uint32) bool {
	return (this.flags & flags) != 0
}

/*
 Setup aggregate specific information
*/

func (this *AggregateInfo) setAiCompute(context *opContext, parent value.Value) error {
	optionsVal, err := this.agg.Operands()[1].Evaluate(parent, context)
	if err != nil {
		return err
	}

	if optionsVal.Type() != value.OBJECT {
		return fmt.Errorf("AI_COMPUTE: second argument (options) must be an object")
	}

	// Check for empty options object
	fields := optionsVal.Fields()
	if len(fields) == 0 {
		return fmt.Errorf("AI_COMPUTE: second argument (options) must not be empty")
	}

	this.options = &AiOptions{batchSize: _WINDOW_CAP}
	this.aiValues = make(value.Values, 0, _WINDOW_CAP)

	// Validate action
	if v, ok := optionsVal.Field("action"); !ok || v.Type() != value.STRING {
		return fmt.Errorf("AI_COMPUTE: 'action' field must be a string")
	} else {
		if action, ok := _AI_ACTIONS[strings.ToLower(v.ToString())]; ok {
			this.options.action = action
		} else {
			return fmt.Errorf("AI_COMPUTE: unsupported action %q - supported: rerank", v.ToString())
		}
	}

	// Validate uri
	if v, ok := optionsVal.Field("uri"); !ok || v.Type() != value.STRING || v.ToString() == "" {
		return fmt.Errorf("AI_COMPUTE: 'uri' field is required and must not be empty")
	} else {
		uriObj, err := util.ParseAndValidateURL(v.ToString())
		if err != nil {
			return err
		}
		if err := expression.IsUrlAllowedInCluster(uriObj, context); err != nil {
			return err
		}
		this.options.uriObj = uriObj
	}

	if v, ok := optionsVal.Field("header"); ok && v.Type() == value.OBJECT {
		this.options.header = v.Actual().(map[string]interface{})
	}
	if v, ok := optionsVal.Field("batchSize"); ok && v.Type() == value.NUMBER {
		this.options.batchSize = value.AsNumberValue(v).Int64()
	}
	if v, ok := optionsVal.Field("model"); ok && v.Type() == value.STRING {
		this.options.model = v.ToString()
	}
	if v, ok := optionsVal.Field("cred_id"); ok && v.Type() == value.STRING {
		this.options.cred_id = v.ToString()
	}
	if this.options.batchSize <= 0 || this.options.batchSize > _WINDOW_CAP {
		this.options.batchSize = _WINDOW_CAP
	}

	// Action-specific: rerank requires query as the third argument.
	if this.options.action == _AI_RERANK {
		if len(this.agg.Operands()) < 3 {
			return fmt.Errorf("AI_COMPUTE: action 'rerank' requires a query as the third argument")
		}
		queryVal, err := this.agg.Operands()[2].Evaluate(parent, context)
		if err != nil {
			return err
		}
		if queryVal.Type() != value.STRING {
			return fmt.Errorf("AI_COMPUTE: third argument (query) must be a string")
		}
		if queryVal.ToString() == "" {
			return fmt.Errorf("AI_COMPUTE: third argument (query) must not be empty")
		}
		this.options.query = queryVal.ToString()
		this.addFlags(_WINDOW_AIFULLSET)
		this.options.batchSize = -1
	}

	return nil
}

// setAiRerank initialises options for the AI_RERANK(docs, options, query) window function.
// operand[1] must evaluate to an object (uri, model, cred_id, header).
// operand[2] must evaluate to a non-empty string (the query text).
// The action is always _AI_RERANK — it is not read from the options object.
func (this *AggregateInfo) setAiRerank(context *opContext, parent value.Value) error {
	// Evaluate operand[1]: options object
	optionsVal, err := this.agg.Operands()[1].Evaluate(parent, context)
	if err != nil {
		return err
	}
	if optionsVal.Type() != value.OBJECT {
		return fmt.Errorf("AI_RERANK: second argument (options) must be an object")
	}
	if len(optionsVal.Fields()) == 0 {
		return fmt.Errorf("AI_RERANK: second argument (options) must not be empty")
	}

	// Evaluate operand[2]: query string
	queryVal, err := this.agg.Operands()[2].Evaluate(parent, context)
	if err != nil {
		return err
	}
	if queryVal.Type() != value.STRING {
		return fmt.Errorf("AI_RERANK: third argument (query) must be a string")
	}
	if queryVal.ToString() == "" {
		return fmt.Errorf("AI_RERANK: third argument (query) must not be empty")
	}

	this.options = &AiOptions{
		action:    _AI_RERANK,
		query:     queryVal.ToString(),
		batchSize: -1, // rerank always operates on the full partition
	}

	// Validate and store URI (mandatory).
	v, ok := optionsVal.Field("uri")
	if !ok || v.Type() != value.STRING || v.ToString() == "" {
		return fmt.Errorf("AI_RERANK: 'uri' field is required and must not be empty")
	}
	uriObj, err := util.ParseAndValidateURL(v.ToString())
	if err != nil {
		return err
	}
	if err := expression.IsUrlAllowedInCluster(uriObj, context); err != nil {
		return err
	}
	this.options.uriObj = uriObj

	if v, ok := optionsVal.Field("header"); ok && v.Type() == value.OBJECT {
		this.options.header = v.Actual().(map[string]interface{})
	}
	if v, ok := optionsVal.Field("model"); ok && v.Type() == value.STRING {
		this.options.model = v.ToString()
	}
	if v, ok := optionsVal.Field("cred_id"); ok && v.Type() == value.STRING {
		this.options.cred_id = v.ToString()
	}

	// Rerank always needs the full partition before it can call the API.
	this.addFlags(_WINDOW_AIFULLSET)

	return nil
}

func (this *AggregateInfo) setOnce(context *opContext, parent value.Value) (err error) {
	if this.hasFlags(_WINDOW_AICOMPUTE) {
		return this.setAiCompute(context, parent)
	} else if this.hasFlags(_WINDOW_AIRERANK) {
		return this.setAiRerank(context, parent)
	}

	// No ORDER BY all rows in parttition has same aggregate value. Evaluate once.
	this.once = this.wTerm.OrderBy() == nil && !this.hasFlags(_WINDOW_ROW_NUMBER)
	windowFrame := this.wTerm.WindowFrame()

	/*
	   For RATIO_TO_REPORT() aggregate
	       It requires SUM. Do two phase.
	         Phase 1 SUM (preAgg)
	         Phase 2 RATIO_TO_REPORT
	*/
	if this.hasFlags(_WINDOW_RATIO_TO_REPORT) {
		this.preAgg = algebra.NewSum(this.agg.Operands(), this.agg.Flags(), this.agg.Filter(), this.wTerm)
	}

	this.repeats = value.ONE_VALUE

	// Setup ORDER BY terms info for each aggregate
	if this.wTerm.OrderBy() != nil {
		this.newCollationValue = true
		this.obyValues = make(value.Values, len(this.wTerm.OrderBy().Terms()))
	}

	// aggregate can be incremental
	this.incremental = !this.once && this.agg.Incremental() && !this.agg.Distinct()

	// FIRST_VALUE(), LAST_VALUE(), NTH_VALUE() need special handling for duplicates. Set flag
	if this.wTerm.OrderBy() != nil && this.hasFlags(_WINDOW_FIRST_VALUE|_WINDOW_LAST_VALUE|_WINDOW_NTH_VALUE) &&
		(windowFrame == nil || !windowFrame.RowsWindowFrame()) {
		this.addFlags(_WINDOW_FL_DUPLICATES)
	}

	if !this.once && windowFrame != nil {
		wfes := windowFrame.WindowFrameExtents()
		between := wfes[0].HasModifier(algebra.WINDOW_FRAME_BETWEEN)
		rangeWindow := windowFrame.RangeWindowFrame()
		rowsWindow := windowFrame.RowsWindowFrame()

		/*
		   Reset incremental
		        window exclude is present
		        window frame is current row only
		        RANGE/GROUPS window frame is NOT start to current row
		*/
		this.incremental = this.incremental && !windowFrame.WindowFrameHasExclude()

		if this.incremental {
			this.incremental = !wfes[0].HasModifier(algebra.WINDOW_FRAME_CURRENT_ROW) ||
				(between && !wfes[1].HasModifier(algebra.WINDOW_FRAME_CURRENT_ROW))

			if this.incremental && !rowsWindow {
				this.incremental = wfes[0].HasModifier(algebra.WINDOW_FRAME_UNBOUNDED_PRECEDING) &&
					(!between || wfes[1].HasModifier(algebra.WINDOW_FRAME_CURRENT_ROW))
			}
		}

		// Validate semantics of start frame VALUE expression
		if wfes[0].HasModifier(algebra.WINDOW_FRAME_VALUE_PRECEDING | algebra.WINDOW_FRAME_VALUE_FOLLOWING) {
			this.sWindowVal, err = this.windowValidateValExpr(wfes[0].ValueExpression(), rangeWindow, context, parent)
			if err != nil {
				return
			}
		}

		if wfes[0].HasModifier(algebra.WINDOW_FRAME_BETWEEN) {
			// set once window frame is present
			this.once = wfes[0].HasModifier(algebra.WINDOW_FRAME_UNBOUNDED_PRECEDING) &&
				wfes[1].HasModifier(algebra.WINDOW_FRAME_UNBOUNDED_FOLLOWING) &&
				!windowFrame.WindowFrameHasExclude()

			// Validate semantics of end frame VALUE expression
			if wfes[1].HasModifier(algebra.WINDOW_FRAME_VALUE_PRECEDING | algebra.WINDOW_FRAME_VALUE_FOLLOWING) {
				this.eWindowVal, err = this.windowValidateValExpr(wfes[1].ValueExpression(), rangeWindow, context, parent)
				if err != nil {
					return
				}
			}
		}
	}

	return
}

/*
 evaluate Phase 1 aggregate and store the result in preVal
*/

func (this *AggregateInfo) evaluatePreAggregate(op *WindowAggregate, wf *windowFrame, cItem int64) (err error) {

	// set default value
	this.val, err = this.preAgg.Default(op.values[cItem], &op.operatorCtx)
	if err != nil {
		return err
	}

	// aggregate on all set of rows
	c := wf.sIndex
	if c < 0 {
		c = 0
	}

	for ; c <= wf.eIndex && c < op.nItems; c++ {
		if wf.excludeRow(c) {
			continue
		}

		this.val, err = this.preAgg.CumulateInitial(op.values[c], this.val, &op.operatorCtx)
		if err != nil {
			return err
		}
	}

	// final result
	this.preVal, err = this.agg.ComputeFinal(this.val, &op.operatorCtx)
	return err
}

/*
evaluate the aggregate
*/

func (this *AggregateInfo) evaluate(op *WindowAggregate, wf *windowFrame, cItem int64) (err error) {
	var item value.AnnotatedValue

	cAiItem := cItem
	if this.hasFlags(_WINDOW_AICOMPUTE | _WINDOW_AIRERANK) {
		if !this.hasFlags(_WINDOW_AIFULLSET) {
			si := (cItem / this.options.batchSize) * this.options.batchSize
			cAiItem = cItem % this.options.batchSize
			wf = &windowFrame{sIndex: si, cIndex: cItem, eIndex: si + this.options.batchSize}
		}
		if cAiItem == 0 {
			this.aiValues = this.aiValues[0:0]
		} else {
			this.val = this.aiValues[cAiItem]
			return
		}
	}

	// set the start row and end row reset boundarires
	s := wf.sIndex
	e := wf.eIndex
	empty := wf.empty
	if s < 0 {
		s = 0
	}

	if e > op.nItems-1 {
		e = op.nItems - 1
	}

	if !empty && s <= e {
		if this.incremental {
			// incremental aggregation, start with previous cumVal
			this.val = this.cumVal
			if wf.cIndex > 0 {
				// remove the outgoing row of frame from cumVal
				if (wf.cIndex >= wf.sIndex && wf.sIndex > 0) || (wf.cIndex < wf.sIndex && wf.sIndex < op.nItems) {
					this.val, err = this.agg.CumulateRemove(op.values[wf.sIndex-1], this.val, &op.operatorCtx)
					if err != nil {
						return err
					}
				}
			}

			e1 := wf.eIndex
			s1 := wf.eIndex

			if !this.hasFlags(_WINDOW_CUME_DIST) {
				s1 -= this.dupsFollowing
			}

			if wf.cIndex < wf.eIndex && wf.cIndex == 0 {
				s1 = s
			}

			// add new row to cumVal
			for c := s1; c <= e1 && c < op.nItems; c++ {
				// setup item for ranking functions
				item, err = this.getWindowRow(c, op.values[c], op)
				if err == nil {
					this.val, err = this.agg.CumulateInitial(item, this.val, &op.operatorCtx)
				}
				if err != nil {
					return err
				}
			}

			// store value into cumVal for feature
			this.cumVal = this.val.Copy()
		} else if this.hasFlags(_WINDOW_FIRST_VALUE | _WINDOW_LAST_VALUE | _WINDOW_NTH_VALUE | _WINDOW_LAG | _WINDOW_LEAD) {
			// evaluate Value functions seprately
			empty, err = this.evaluateValueFuncs(op, wf, s, e, cItem)
			if err != nil {
				return err
			}
		} else {
			// non incremental aggregation
			empty = true
			// default aggregation value
			this.val, err = this.agg.Default(op.values[cItem], &op.operatorCtx)
			if err != nil {
				return err
			}

			for c := s; c <= e; c++ {
				// include in aggregation only if not excluded
				if !wf.excludeRow(c) {
					empty = false
					// setup item for ranking functions
					item, err = this.getWindowRow(c, op.values[c], op)
					if err == nil {
						this.val, err = this.agg.CumulateInitial(item, this.val, &op.operatorCtx)
					}
					if err != nil {
						return err
					}
				}
			}
		}
	}

	if empty || s > e {
		// no frame or empty row
		this.val, err = this.agg.Default(op.values[cItem], &op.operatorCtx)
	}

	// final aggregation value
	val, err := this.agg.ComputeFinal(this.val, &op.operatorCtx)
	if err != nil {
		return err
	}
	if this.hasFlags(_WINDOW_AIRERANK) {
		// ai_rerank: direct one-hop call to the canonical rerank implementation.
		err = this.aiRerankEvaluate(val, op)
		if err == nil {
			this.val = this.aiValues[cAiItem]
		}
	} else if this.hasFlags(_WINDOW_AICOMPUTE) {
		// ai_compute: goes through aiEvaluate which may delegate to aiRerankEvaluate.
		err = this.aiEvaluate(val, op)
		if err == nil {
			this.val = this.aiValues[cAiItem]
		}
	} else {
		this.val = val
	}

	return err
}

// aiRerankEvaluate is the canonical implementation of the rerank HTTP call.
// It is called directly by ai_rerank (one hop) and via aiEvaluate by
// ai_compute with action:"rerank" (two hops).
func (this *AggregateInfo) aiRerankEvaluate(val value.Value, op *WindowAggregate) error {
	if val.Type() != value.ARRAY {
		return fmt.Errorf("AI_RERANK: expected an array of document values")
	}

	uriObj := this.options.uriObj

	docs := val.Actual().([]interface{})
	if len(docs) == 0 {
		return nil
	}

	payloadMap := map[string]interface{}{
		"query":     this.options.query,
		"documents": docs,
	}
	if this.options.model != "" {
		payloadMap["model"] = this.options.model
	}

	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return err
	}

	client := http.Client{}
	header := http.Header{}

	if this.options.cred_id != "" {
		client, header, err = expression.HandleCred(uriObj, this.options.cred_id, &op.operatorCtx)
		if err != nil {
			return err
		}
	} else {
		client, err = expression.GetDefaultHttpClient(&op.operatorCtx)
		if err != nil {
			return err
		}
	}
	if client.Timeout == 0 {
		client.Timeout = 30 * time.Second
	}

	// The request body is always a JSON-encoded payload.  Set Content-Type so that
	// APIs that enforce it (returning HTTP 415 otherwise) work without the caller
	// having to add it to config.header manually.  User-provided header entries
	// applied in the loop below can override this default if necessary.
	header.Set("Content-Type", "application/json")

	if this.options.header != nil {
		for k, v := range this.options.header {
			// When config is a N1QL literal object the values in the map are
			// value.Value (e.g. stringValue), not plain Go strings.  Handle both
			// so the header is correctly forwarded regardless of the code path.
			switch sv := v.(type) {
			case string:
				header.Set(k, sv)
			case value.Value:
				if sv.Type() == value.STRING {
					header.Set(k, sv.ToString())
				}
			}
		}
	}

	req, err := http.NewRequest("POST", uriObj.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header = header

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("AI_RERANK: API returned status %d: %s", resp.StatusCode, string(body))
	}

	this.aiValues = make(value.Values, len(docs))
	for i := range this.aiValues {
		this.aiValues[i] = value.NULL_VALUE
	}

	var rawResponse map[string]interface{}
	if err = json.Unmarshal(body, &rawResponse); err != nil {
		return err
	}

	// Accept either "results" or "data" as the top-level array key.
	var resultList []interface{}
	for _, key := range []string{"results", "data"} {
		if v, ok := rawResponse[key]; ok {
			if arr, ok := v.([]interface{}); ok {
				resultList = arr
				break
			}
		}
	}
	if resultList == nil {
		return fmt.Errorf("AI_RERANK: API response does not match expected format: missing required top-level array field \"results\" or \"data\"")
	}

	for _, entry := range resultList {
		respDoc, ok := entry.(map[string]interface{})
		if !ok {
			op.operatorCtx.Warning(errors.NewAiRerankResponseWarning("entry is not a JSON object"))
			continue
		}
		idx, ok := respDoc["index"].(float64)
		if !ok {
			op.operatorCtx.Warning(errors.NewAiRerankResponseWarning(`"index" field is missing or not a number`))
			continue
		}
		if int(idx) < 0 || int(idx) >= len(this.aiValues) {
			return fmt.Errorf("AI_RERANK: invalid index in response: %v", idx)
		}
		score, hasScore := respDoc["relevance_score"]
		if !hasScore || score == nil {
			return fmt.Errorf("AI_RERANK: missing relevance_score in response at index %v", idx)
		}
		this.aiValues[int(idx)] = value.NewValue(score)
	}

	return nil
}

func (this *AggregateInfo) aiEvaluate(val value.Value, op *WindowAggregate) error {
	switch this.options.action {
	case _AI_RERANK:
		// Delegate to the canonical rerank implementation (two-hop path for ai_compute).
		return this.aiRerankEvaluate(val, op)
	default:
		return fmt.Errorf("AI_COMPUTE: unsupported action (internal error)")
	}
}

/*
evaluate the Value aggregates
*/

func (this *AggregateInfo) evaluateValueFuncs(op *WindowAggregate, wf *windowFrame, sItem, eItem, cItem int64) (
	empty bool, err error) {

	empty = false
	// default aggregate value
	this.val, err = this.agg.Default(op.values[cItem], &op.operatorCtx)
	if err != nil {
		return false, err
	}

	// set start and end for each aggregate
	direction := int64(1)
	c := cItem
	s := sItem
	e := eItem

	if this.hasFlags(_WINDOW_FIRST_VALUE) ||
		(this.hasFlags(_WINDOW_NTH_VALUE) && !this.agg.HasFlags(algebra.AGGREGATE_FROMLAST)) {
		c = sItem
	} else if this.hasFlags(_WINDOW_LAST_VALUE) ||
		(this.hasFlags(_WINDOW_NTH_VALUE) && this.agg.HasFlags(algebra.AGGREGATE_FROMLAST)) {
		direction = -1
		c = eItem
	} else if this.hasFlags(_WINDOW_LAG) {
		c = cItem - 1
		direction = -1
	} else if this.hasFlags(_WINDOW_LEAD) {
		c = cItem + 1
	}

	repeats := int64(0)
	for ; (direction > 0 && c <= e) || (direction < 0 && c >= s); c += direction {
		if this.hasFlags(_WINDOW_FL_DUPLICATES) {
			// check how many duplicates at current row
			if repeats == 0 {
				repeats, err = this.windowOrderDuplicatesDirection(op, c, direction)
				if err != nil {
					return false, err
				}
				repeats++
			}
		} else {
			repeats++
		}

		// include in aggregation if not excluded
		if !wf.excludeRow(c) {
			empty = false
			this.val, err = this.agg.CumulateInitial(op.values[c], this.val, &op.operatorCtx)
			if err != nil {
				return false, err
			}
		}
		repeats--
		if repeats == 0 {
			// check if aggregation is done early when no more duplicates
			done, err := this.agg.IsCumulateDone(this.val, &op.operatorCtx)
			if err != nil || done {
				return empty, err
			}
		}
	}
	return empty, nil
}

/*
Setup Window input row aggregate for ranking
*/
func (this *AggregateInfo) getWindowRow(cIndex int64, item value.AnnotatedValue, op *WindowAggregate) (
	value.AnnotatedValue, error) {

	// these aggregates does not need any pre setup
	if !this.hasFlags(_WINDOW_ROW_NUMBER | _WINDOW_DENSE_RANK | _WINDOW_PERCENT_RANK |
		_WINDOW_RANK | _WINDOW_CUME_DIST | _WINDOW_RATIO_TO_REPORT | _WINDOW_NTILE) {
		return item, nil
	}

	// get WINDOW_ATTACHMENT or setup one

	var val value.Value
	v1 := item.GetAttachment(value.ATT_WINDOW_ATTACHMENT)
	if v1 != nil {
		val = v1.(value.Value)
	}

	if v1 == nil || val == nil {
		val = value.NewValue(map[string]interface{}{"part": value.ONE_VALUE, "nrows": value.NewValue(op.nItems)})
		item.SetAttachment(value.ATT_WINDOW_ATTACHMENT, val)
	}

	if this.hasFlags(_WINDOW_ROW_NUMBER) {
		//  how much need to incremented each row. i.e. 1
		val.SetField("part", this.repeats)
		return item, nil
	} else if this.hasFlags(_WINDOW_CUME_DIST) {
		val.SetField("part", value.NewValue(cIndex+1+this.dupsFollowing))
		val.SetField("nrows", value.NewValue(op.nItems))
		return item, nil
	} else if this.hasFlags(_WINDOW_RATIO_TO_REPORT) {
		// setup Phase 1 aggregate value. ie SUM()
		val.SetField("part", this.preVal)
		return item, nil
	} else if this.hasFlags(_WINDOW_NTILE) {
		// Setup current row value and total rows
		val.SetField("part", value.NewValue(cIndex))
		val.SetField("nrows", value.NewValue(op.nItems))
		return item, nil
	} else if this.hasFlags(_WINDOW_DENSE_RANK | _WINDOW_RANK | _WINDOW_PERCENT_RANK) {
		if this.hasFlags(_WINDOW_PERCENT_RANK) {
			val.SetField("nrows", value.NewValue(op.nItems))
		}

		ok, err := this.isNewCollationValue(item, this.obyValues, op)
		if err != nil {
			return nil, err
		}

		if ok {
			err = this.evaluateObyValues(item, op)
			if err != nil {
				return nil, err
			}
			val.SetField("part", this.repeats)
			this.repeats = value.ONE_VALUE
			return item, nil
		} else if this.hasFlags(_WINDOW_RANK | _WINDOW_PERCENT_RANK) {
			this.repeats = value.AsNumberValue(this.repeats).Add(value.AsNumberValue(value.ONE_VALUE))
		}
		val.SetField("part", value.ZERO_VALUE)
		return item, nil
	}

	return item, nil
}

/*
  calculate window frame
*/

func (this *AggregateInfo) windowFramePositions(op *WindowAggregate, c int64) (*windowFrame, error) {

	var err error
	wf := &windowFrame{}
	cIndex := c
	sIndex := int64(0)
	eIndex := c
	empty := false
	this.dupsPreceding = 0
	this.dupsFollowing = 0
	windowFrame := this.wTerm.WindowFrame()

	if this.once || this.hasFlags(_WINDOW_LEAD) || this.hasFlags(_WINDOW_AIFULLSET) {
		eIndex = op.nItems - 1
	} else if windowFrame != nil {
		wfes := windowFrame.WindowFrameExtents()
		wfe := wfes[0]
		if wfe.HasModifier(algebra.WINDOW_FRAME_UNBOUNDED_PRECEDING) {
			sIndex = 0
		} else if wfe.HasModifier(algebra.WINDOW_FRAME_CURRENT_ROW) {
			sIndex = cIndex
			if !windowFrame.RowsWindowFrame() {
				// needs to include duplicates before current row
				sIndex, empty, err = this.windowValuePeerPos(op, nil, int64(0), cIndex, int64(-1), int64(1), true)
			}
		} else if wfe.HasModifier(algebra.WINDOW_FRAME_VALUE_PRECEDING) {
			sIndex, empty, err = this.windowValuePos(op, this.sWindowVal, cIndex, int64(-1), true)
		} else if wfe.HasModifier(algebra.WINDOW_FRAME_VALUE_FOLLOWING) {
			sIndex, empty, err = this.windowValuePos(op, this.sWindowVal, cIndex, int64(1), true)
		}

		if err == nil && !empty {
			if wfe.HasModifier(algebra.WINDOW_FRAME_BETWEEN) {
				wfe = wfes[1]
				if wfe.HasModifier(algebra.WINDOW_FRAME_UNBOUNDED_FOLLOWING) {
					eIndex = op.nItems
				} else if wfe.HasModifier(algebra.WINDOW_FRAME_CURRENT_ROW) {
					eIndex = cIndex
					if !windowFrame.RowsWindowFrame() {
						// needs to include duplicates after current row
						eIndex, empty, err = this.windowValuePeerPos(op, nil, int64(0), cIndex, int64(1), int64(1), false)
					}
				} else if wfe.HasModifier(algebra.WINDOW_FRAME_VALUE_PRECEDING) {
					eIndex, empty, err = this.windowValuePos(op, this.eWindowVal, cIndex, int64(-1), false)
				} else if wfe.HasModifier(algebra.WINDOW_FRAME_VALUE_FOLLOWING) {
					eIndex, empty, err = this.windowValuePos(op, this.eWindowVal, cIndex, int64(1), false)
				}
			} else {
				// default frame
				eIndex = cIndex
				if !windowFrame.RowsWindowFrame() {
					// needs to include duplicates after current row
					eIndex, empty, err = this.windowValuePeerPos(op, nil, int64(0), cIndex, int64(1), int64(1), false)
				}
			}
		}
		if err == nil && !empty && !windowFrame.RowsWindowFrame() && !windowFrame.WindowFrameHasExclude() {
			// calculate duplicates before and after current row
			this.dupsPreceding, this.dupsFollowing, err = this.windowOrderDuplicates(op, cIndex)
		}
	} else if this.wTerm.OrderBy() != nil && !this.hasFlags(_WINDOW_NOEQUAL_ROWS) {
		// calculate duplicates before and after current row
		this.dupsPreceding, this.dupsFollowing, err = this.windowOrderDuplicates(op, cIndex)
		if err == nil {
			if !this.hasFlags(_WINDOW_CUME_DIST) {
				// need to include duplicates after current row
				eIndex, _, err = this.windowValuePeerPos(op, nil, int64(0), cIndex, int64(1), int64(1), false)
			}
		}
	}

	// Setup frame structure
	return wf.setupWindowFrame(op, this, cIndex, sIndex, eIndex, empty, err)

}

/*
  calculate physical pos from the value
*/

func (this *AggregateInfo) windowValuePos(op *WindowAggregate, val value.Value, cIndex, direction int64,
	sframe bool) (pos int64, empty bool, err error) {

	collation := int64(1)
	if this.wTerm.WindowFrame().RangeWindowFrame() {
		pos = cIndex
		if d := op.obyDesc[0]; d {
			collation = int64(-1)
		}

		var rangeVal, currentObyVal value.Value

		currentObyVal, err = getCachedValue(op.values[cIndex], op.oby[0].Expression(), op.obyTerms[0], &op.operatorCtx)
		if err != nil || currentObyVal == nil ||
			!(currentObyVal.Type() == value.NUMBER || currentObyVal.Type() <= value.NULL) {
			return cIndex, true, err
		}

		// range add the logical offset
		if currentObyVal.Type() == value.NUMBER {
			if (direction * collation) < 0 {
				rangeVal = value.AsNumberValue(currentObyVal).Sub(value.AsNumberValue(val))
			} else {
				rangeVal = value.AsNumberValue(currentObyVal).Add(value.AsNumberValue(val))
			}
		} else {
			rangeVal = currentObyVal
		}

		// calcuate physical offset
		return this.windowValuePeerPos(op, rangeVal, int64(0), cIndex, direction, collation, sframe)
	}

	offset := val.(value.NumberValue).Int64()

	if this.wTerm.WindowFrame().RowsWindowFrame() {
		// Physical offset for ROWS
		return cIndex + (direction * offset), empty, nil
	}

	// GROUPS calculate physical offset from logical groups
	if offset == 0 && ((direction > 0 && sframe) || (direction < 0 && !sframe)) {
		return this.windowValuePeerPos(op, nil, offset, cIndex, -direction, collation, sframe)
	}

	return this.windowValuePeerPos(op, nil, offset, cIndex, direction, collation, sframe)
}

/*
 calcuate physical offset from logical offset
 Consider:
          start/end             frame
          ASC/DESC              ORDER
          FOLLOWING/PRECEDING   frame directiion
*/

func (this *AggregateInfo) windowValuePeerPos(op *WindowAggregate, rangeVal value.Value,
	nGroups, cIndex, direction, collation int64, sframe bool) (pos int64, empty bool, err error) {

	if rangeVal != nil {
		// RANGE
		return this.windowValueRangePeerPos(op, rangeVal, cIndex, direction, collation, sframe)
	}

	// GROUPS
	pos = cIndex
	var dups int64

	for g := int64(0); g <= nGroups && pos >= 0 && pos < op.nItems; g++ {
		dups, err = this.windowOrderDuplicatesDirection(op, pos, direction)
		if err != nil {
			break
		}

		if sframe {
			if direction > 0 && g < nGroups {
				pos += dups
			} else if direction < 0 {
				pos -= dups
			}

			if g < nGroups {
				pos += direction
			}
		} else {
			if direction > 0 {
				pos += dups
			} else if direction < 0 && g < nGroups {
				pos -= dups
			}

			if g < nGroups {
				pos += direction
			}
		}
	}

	return pos, false, err
}

/*
calcuate duplicates befor and after current row. current row not included
*/

func (this *AggregateInfo) windowOrderDuplicates(op *WindowAggregate, cIndex int64) (
	dupsPreceding, dupsFollowing int64, err error) {

	dupsPreceding, err = this.windowOrderDuplicatesDirection(op, cIndex, -1)
	if err == nil {
		dupsFollowing, err = this.windowOrderDuplicatesDirection(op, cIndex, 1)
	}
	return
}

/*
Duplicate calculations
*/
func (this *AggregateInfo) windowOrderDuplicatesDirection(op *WindowAggregate, cIndex int64, direction int64) (
	dups int64, err error) {

	oby := this.wTerm.OrderBy()
	if oby == nil {
		return
	}

	cobyValues := make(value.Values, len(this.wTerm.OrderBy().Terms()))
	for i, obyExpr := range oby.Expressions() {
		cobyValues[i], err = getCachedValue(op.values[cIndex], obyExpr, op.obyTerms[i], &op.operatorCtx)
		if err != nil || cobyValues[i] == nil {
			return
		}
	}
	var cc bool

	for pos := cIndex + direction; pos >= 0 && pos < op.nItems; pos = pos + direction {
		cc, err = isNewWindowValues(op.values[pos], false, oby.Expressions(), cobyValues, op.obyTerms, &op.operatorCtx)
		if err != nil || cc {
			return
		}
		dups++
	}
	return
}

/*
 calcuate physical offset from logical offset for RANGE
 Consider:
          start/end             frame
          ASC/DESC              ORDER
          FOLLOWING/PRECEDING   frame directiion
*/

func (this *AggregateInfo) windowValueRangePeerPos(op *WindowAggregate, rangeVal value.Value,
	cIndex, direction, collation int64, sframe bool) (pos int64, empty bool, err error) {

	if rangeVal == nil {
		return cIndex, true, err
	}

	var dups int64
	var otherObyVal value.Value
	for pos = cIndex; (direction < 0 && pos >= 0) || (direction > 0 && pos < op.nItems); pos = pos + direction {
		otherObyVal, err = getCachedValue(op.values[pos], op.oby[0].Expression(), op.obyTerms[0], &op.operatorCtx)

		if err != nil || otherObyVal == nil {
			return pos - direction, false, err
		}

		cc := otherObyVal.Collate(rangeVal)
		if cc == 0 {
			if sframe {
				dups, err = this.windowOrderDuplicatesDirection(op, pos, -1)
				return pos - dups, false, err
			} else {
				dups, err = this.windowOrderDuplicatesDirection(op, pos, 1)
				return pos + dups, false, err
			}
		}

		if (cc * int(direction) * int(collation)) > 0 {
			if direction < 0 && sframe || direction > 0 && !sframe {
				return pos - direction, false, err
			}
			return pos, false, err
		}

	}

	return pos, false, err
}

// value_expr must be a constant or expression and must evaluate to a positive numeric value.
func (this *AggregateInfo) windowValidateValExpr(valExpr expression.Expression,
	rangeWindow bool, context *opContext, parent value.Value) (value.Value, error) {

	val, err := valExpr.Evaluate(parent, context)
	if err != nil {
		return val, err
	}
	if val != nil && val.Type() == value.NUMBER && val.(value.NumberValue).Float64() >= 0.0 &&
		(rangeWindow || value.IsInt(val.(value.NumberValue).Float64())) {
		return val, nil
	}

	return nil, fmt.Errorf("value_expr must be a constant or expression and must evaluate to a positive numeric value.")
}

// evalute aggregate ORDER BY terms
func (this *AggregateInfo) evaluateObyValues(item value.AnnotatedValue, op *WindowAggregate) error {
	if this.wTerm.OrderBy() != nil {
		err := evaluateWindowByValues(item, this.wTerm.OrderBy().Expressions(), this.obyValues, op.obyTerms, &op.operatorCtx)
		if err != nil {
			return err
		}
	}
	this.newCollationValue = false
	return nil
}

// Return item NewCollation form previous one or not

func (this *AggregateInfo) isNewCollationValue(item value.AnnotatedValue, obyValues value.Values, op *WindowAggregate) (
	rv bool, err error) {

	if this.wTerm.OrderBy() == nil {
		return false, nil
	}
	return isNewWindowValues(item, this.newCollationValue, this.wTerm.OrderBy().Expressions(), obyValues, op.obyTerms,
		&op.operatorCtx)
}

// Sets up window frame
func (this *windowFrame) setupWindowFrame(op *WindowAggregate, aInfo *AggregateInfo, cIndex, sIndex, eIndex int64,
	empty bool, err error) (*windowFrame, error) {
	this.empty = empty
	this.cIndex = cIndex
	this.sIndex = sIndex
	this.eIndex = eIndex
	wf := aInfo.wTerm.WindowFrame()
	if err != nil || empty || wf == nil || !wf.WindowFrameHasExclude() {
		return this, err
	}

	if wf.HasModifier(algebra.WINDOW_FRAME_EXCLUDE_CURRENT_ROW) {
		// exclude current row
		if cIndex >= sIndex && cIndex <= eIndex {
			this.exclude = append(this.exclude, &excludeFrame{sIndex: cIndex, eIndex: cIndex})
		}
	} else {
		var s, e int64
		// start is first value in duplicates
		s, _, err = aInfo.windowValuePeerPos(op, nil, int64(0), cIndex, int64(-1), int64(1), true)
		if err == nil {
			// end is last value in duplicates
			e, _, err = aInfo.windowValuePeerPos(op, nil, int64(0), cIndex, int64(1), int64(1), false)
		}

		if err == nil {
			if s < sIndex {
				s = sIndex
			}
			if e > eIndex {
				e = eIndex
			}

			if wf.HasModifier(algebra.WINDOW_FRAME_EXCLUDE_GROUP) {
				// exclude GROUP
				if s <= e {
					this.exclude = append(this.exclude, &excludeFrame{sIndex: s, eIndex: e})
				}
			} else if wf.HasModifier(algebra.WINDOW_FRAME_EXCLUDE_TIES) {
				// exclude duplicates except current row
				if s <= cIndex-1 {
					this.exclude = append(this.exclude, &excludeFrame{sIndex: s, eIndex: cIndex - 1})
				}
				if cIndex+1 <= e {
					this.exclude = append(this.exclude, &excludeFrame{sIndex: cIndex + 1, eIndex: e})
				}
			}
		}
	}
	return this, err
}

// Row is in exclude list or not
func (this *windowFrame) excludeRow(cRow int64) bool {
	for _, ef := range this.exclude {
		if cRow >= ef.sIndex && cRow <= ef.eIndex {
			return true
		}
	}
	return false
}

func (this *WindowAggregate) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, this.releaseValues)
}

func (this *WindowAggregate) beforeItems(context *Context, parent value.Value) bool {
	return this.setupTerms(parent)
}

func (this *WindowAggregate) processItem(item value.AnnotatedValue, context *Context) bool {

	// check if item is in new partition or not
	newPartition, err := this.isNewPartition(item)
	if err != nil {
		return false
	}

	if newPartition {
		// if new partition process all aggregates
		if this.nItems > 0 && !this.afterWindowPartition(true) {
			return false
		}

		// reset partition values for new item
		if !this.evaluatePbyValues(item) {
			return false
		}
		this.newPartition = false
	}

	// process ietm
	if !this.beforeWindowPartition(item) {
		return false
	}

	if !this.processWindowPartition(item) {
		return false
	}

	if !this.afterWindowPartition(false) {
		return false
	}

	return true
}

func (this *WindowAggregate) afterItems(context *Context) {
	// end process all items
	this.afterWindowPartition(true)
	this.releaseValues()
}

func (this *WindowAggregate) isNewPartition(item value.AnnotatedValue) (rv bool, err error) {
	return isNewWindowValues(item, this.newPartition, this.pby, this.pbyValues, this.pbyTerms, &this.operatorCtx)
}

func isNewWindowValues(item value.AnnotatedValue, firstVal bool, exprs expression.Expressions, values value.Values,
	names []string, context *opContext) (bool, error) {

	if firstVal {
		return true, nil
	}

	for i, expr := range exprs {
		oVal, err := getCachedValue(item, expr, names[i], context)
		if err != nil || oVal == nil {
			if oVal == nil {
				err = fmt.Errorf("value is nil")
			}
			context.Fatal(errors.NewWindowEvaluationError(err, "Error during evaluating duplicate oby value."))
			return false, err
		}

		if oVal.Collate(values[i]) != 0 {
			return true, nil
		}
	}

	return false, nil
}

func evaluateWindowByValues(item value.AnnotatedValue, exprs expression.Expressions, values value.Values, names []string,
	context *opContext) error {
	var err error
	for i, expr := range exprs {
		values[i], err = getCachedValue(item, expr, names[i], context)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *WindowAggregate) evaluatePbyValues(item value.AnnotatedValue) bool {
	if err := evaluateWindowByValues(item, this.pby, this.pbyValues, this.pbyTerms, &this.operatorCtx); err != nil {
		this.operatorCtx.Fatal(errors.NewWindowEvaluationError(err, "Error evaluating Window partition value."))
		return false
	}

	for _, aInfo := range this.aggs {
		aInfo.repeats = value.ONE_VALUE
		if aInfo.wTerm.OrderBy() != nil {
			aInfo.newCollationValue = true
		}
		if aInfo.incremental {
			aInfo.cumVal, _ = aInfo.agg.Default(nil, &this.operatorCtx)
		}
	}

	return true
}

// Setup item for aggregation
func (this *WindowAggregate) beforeWindowPartition(item value.AnnotatedValue) bool {

	aggregates := item.GetAttachment(value.ATT_AGGREGATES)
	switch aggregates := aggregates.(type) {
	case map[string]value.Value:
	default:
		aggregates = make(map[string]value.Value, len(this.plan.Aggregates()))
		item.SetAttachment(value.ATT_AGGREGATES, aggregates)
	}

	return true
}

// batch the item

func (this *WindowAggregate) processWindowPartition(item value.AnnotatedValue) bool {
	if len(this.values) == cap(this.values) {
		values := make(value.AnnotatedValues, len(this.values), len(this.values)<<1)
		copy(values, this.values)
		this.releaseValues()
		this.values = values
	}

	this.values = append(this.values, item)
	this.nItems++

	return true
}

func (this *WindowAggregate) afterWindowPartition(all bool) bool {
	if !all {
		// not end of partition
		if this.hasFlags(_WINDOW_RELEASE_CURRENTROW) {
			// aggregat depends on current row only
			defer this.recycleValue(this.cItem)
			av := this.values[this.cItem]
			return !this.stopped && this.processWindowAggregates(this.cItem, av) && this.sendItem(av)
		}
		return true
	}

	// end of partition process aggrgeates and recycle values
	defer this.recycleValues()
	for c, item := range this.values {
		if !this.stopped && !this.processWindowAggregates(int64(c), item) {
			return false
		}
	}

	for _, av := range this.values {
		if !this.stopped && !this.sendItem(av) {
			return false
		}
	}

	return true
}

/*
Aggregate evaluation
*/
func (this *WindowAggregate) processWindowAggregates(c int64, item value.AnnotatedValue) bool {
	var err error
	var wf *windowFrame

	for _, aInfo := range this.aggs {
		// aInfo.dupsFollowing == 0 means new group
		if c == 0 || aInfo.hasFlags(_WINDOW_RATIO_TO_REPORT) || (!aInfo.once && aInfo.dupsFollowing == 0) {
			// setup window frame. aInfo.dupsFollowing calculated by below call
			wf, err = aInfo.windowFramePositions(this, int64(c))
			if aInfo.hasFlags(_WINDOW_RATIO_TO_REPORT) {
				if c == 0 || (!aInfo.once && aInfo.dupsFollowing == 0) {
					err = aInfo.evaluatePreAggregate(this, wf, c)
					if err != nil {
						this.operatorCtx.Fatal(errors.NewWindowEvaluationError(err, "Error evaluating Window function."))
						return false
					}
				} else if aInfo.dupsFollowing > 0 {
					aInfo.dupsFollowing--
				}
				wf = &windowFrame{sIndex: c, cIndex: c, eIndex: c}
			}

			// evalue aggregate
			if err == nil {
				err = aInfo.evaluate(this, wf, c)
			}

			if err != nil {
				this.operatorCtx.Fatal(errors.NewWindowEvaluationError(err, "Error evaluating Window function."))
				return false
			}
		} else if aInfo.dupsFollowing > 0 {
			aInfo.dupsFollowing--
		}

		// set final value of aggregate
		aggregates := item.GetAttachment(value.ATT_AGGREGATES).(map[string]value.Value)
		aggregates[aInfo.id] = aInfo.val
	}

	return true
}

func (this *WindowAggregate) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *WindowAggregate) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	this.values = _WINDOW_POOL.Get()
	this.nItems = 0
	this.cItem = 0
	this.newPartition = true
	return rv
}

func (this *WindowAggregate) recycleValues() {
	this.values = this.values[0:0]
	this.nItems = 0
	this.cItem = 0
	this.newPartition = true
}

func (this *WindowAggregate) recycleValue(c int64) {
	this.values = append(this.values[:c], this.values[c+1:]...)
	this.nItems--
}

func (this *WindowAggregate) releaseValues() {
	if this.values != nil {
		_WINDOW_POOL.Put(this.values)
	}
	this.values = nil
}
