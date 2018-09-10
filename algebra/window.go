//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"sort"

	"github.com/couchbase/query/expression"
)

/*
 Window Term
  partitionBy   PARTITION BY clause
  orderBy       ORDER BY BY clause
  windowframe   window frame clause
*/

type WindowTerm struct {
	partitionBy expression.Expressions
	orderBy     *Order
	windowFrame *WindowFrame
}

func NewWindowTerm(pby expression.Expressions, oby *Order, windowFrame *WindowFrame) *WindowTerm {
	rv := &WindowTerm{
		partitionBy: removeDuplicatePbys(pby),
		orderBy:     oby,
		windowFrame: windowFrame,
	}

	return rv
}

/*
Copy
*/
func (this *WindowTerm) Copy() *WindowTerm {
	rv := &WindowTerm{}

	if this.partitionBy != nil {
		rv.partitionBy = this.partitionBy.Copy()
	}

	if this.orderBy != nil {
		rv.orderBy = this.orderBy.Copy()
	}

	if this.windowFrame != nil {
		rv.windowFrame = this.windowFrame.Copy()
	}

	return rv
}

/*
 Copy WindowTerm
*/
func CopyWindowTerm(wTerm *WindowTerm) *WindowTerm {
	if wTerm == nil {
		return nil
	}
	return wTerm.Copy()
}

/*
 Return PartitionBy Info
*/
func (this *WindowTerm) PartitionBy() expression.Expressions {
	return this.partitionBy
}

/*
 Return OrderBy Info
*/
func (this *WindowTerm) OrderBy() *Order {
	return this.orderBy
}

/*
 Return Window Info
*/
func (this *WindowTerm) WindowFrame() *WindowFrame {
	return this.windowFrame
}

/* Map Expressions
 */
func (this *WindowTerm) MapExpressions(mapper expression.Mapper) (err error) {
	pby := this.PartitionBy()
	if pby != nil {
		err = pby.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	oby := this.OrderBy()
	if oby != nil {
		err = oby.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	windowFrame := this.WindowFrame()
	if windowFrame != nil {
		err = windowFrame.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	return
}

/*
  Return all expressions used in WindowTerm
*/
func (this *WindowTerm) Expressions() expression.Expressions {
	rv := make(expression.Expressions, 0, 4)

	pby := this.PartitionBy()
	if len(pby) > 0 {
		rv = append(rv, pby...)
	}

	oby := this.OrderBy()
	if oby != nil {
		rv = append(rv, oby.Expressions()...)
	}

	windowFrame := this.WindowFrame()
	if windowFrame != nil {
		exprs := windowFrame.Expressions()
		if len(exprs) > 0 {
			rv = append(rv, exprs...)
		}
	}
	return rv
}

/*
String representantion
*/
func (this *WindowTerm) String() (s string) {
	s += " OVER ("
	pby := this.PartitionBy()

	// PARTITION BY
	if len(pby) > 0 {
		s += "PARTITION BY "
		/*
		   order of PARTITION BY expressions are no-impact.
		   Ordered them by names so that it can be comparable
		*/
		names := make([]string, len(pby))
		stringer := expression.NewStringer()
		for i, pk := range pby {
			names[i] = stringer.Visit(pk)
		}
		sort.Strings(names)

		for i, _ := range names {
			if i != 0 {
				s += ", "
			}
			s += names[i]
		}
	}

	// ORDER BY
	oby := this.OrderBy()
	if oby != nil {
		s += oby.String()
	}

	// window clause
	windowFrame := this.WindowFrame()
	if windowFrame != nil {
		s += windowFrame.String()
	}
	s += ")"

	return s
}

/*
 remove duplicate expressions and static expression from
 PARTITION BY clause becaue those are no impact
*/

func removeDuplicatePbys(pby expression.Expressions) expression.Expressions {
	if len(pby) > 0 {
		pbyMap := make(map[string]expression.Expression, len(pby))
		for _, expr := range pby {
			if expr.Static() == nil {
				pbyMap[expr.String()] = expr
			}
		}

		pby = make(expression.Expressions, 0, len(pbyMap))
		for _, expr := range pbyMap {
			pby = append(pby, expr)
		}

		if len(pby) == 0 {
			return nil
		}
	}

	return pby
}

// window frame constants
const (
	WINDOW_FRAME_NONE = 1 << iota
	WINDOW_FRAME_ROWS
	WINDOW_FRAME_RANGE
	WINDOW_FRAME_GROUPS
	WINDOW_FRAME_BETWEEN
	WINDOW_FRAME_CURRENT_ROW
	WINDOW_FRAME_UNBOUNDED_PRECEDING
	WINDOW_FRAME_UNBOUNDED_FOLLOWING
	WINDOW_FRAME_VALUE_PRECEDING
	WINDOW_FRAME_VALUE_FOLLOWING
	WINDOW_FRAME_EXCLUDE_CURRENT_ROW
	WINDOW_FRAME_EXCLUDE_GROUP
	WINDOW_FRAME_EXCLUDE_TIES
	WINDOW_FRAME_EXCLUDE_NO_OTHERS
)

/*
Window frame information
*/
type WindowFrame struct {
	windowFrameExtents   WindowFrameExtents
	windowFrameModifiers uint32
}

/*
 New window frame
*/
func NewWindowFrame(modifiers uint32, wfes WindowFrameExtents) *WindowFrame {
	if len(wfes) == 2 {
		// frame has start and end set BETWEEN on both the frames
		wfes[0].modifiers |= WINDOW_FRAME_BETWEEN
		wfes[1].modifiers |= WINDOW_FRAME_BETWEEN
	}

	return &WindowFrame{
		windowFrameExtents:   wfes,
		windowFrameModifiers: modifiers,
	}
}

/*
 Copy window frame
*/
func (this *WindowFrame) Copy() *WindowFrame {
	return &WindowFrame{
		windowFrameExtents:   this.WindowFrameExtents().Copy(),
		windowFrameModifiers: this.WindowFrameModifiers(),
	}
}

/*
 String representation
*/
func (this *WindowFrame) String() (s string) {
	// window frame unit type
	if this.HasModifier(WINDOW_FRAME_ROWS) {
		s += " ROWS"
	} else if this.HasModifier(WINDOW_FRAME_RANGE) {
		s += " RANGE"
	} else if this.HasModifier(WINDOW_FRAME_GROUPS) {
		s += " GROUPS"
	}

	// window frame extents
	wfes := this.WindowFrameExtents()
	if len(wfes) == 2 {
		s += " BETWEEN" + wfes[0].String() + " AND" + wfes[1].String()
	} else {
		s += wfes[0].String()
	}

	// window frame exclude clause
	if this.HasModifier(WINDOW_FRAME_EXCLUDE_CURRENT_ROW) {
		s += " EXCLUDE CURRENT ROW"
	} else if this.HasModifier(WINDOW_FRAME_EXCLUDE_GROUP) {
		s += " EXCLUDE GROUP"
	} else if this.HasModifier(WINDOW_FRAME_EXCLUDE_TIES) {
		s += " EXCLUDE TIES"
	}

	return
}

/*
 Returns window frame extents
*/
func (this *WindowFrame) WindowFrameExtents() WindowFrameExtents {
	return this.windowFrameExtents
}

/*
 Returns window frame modifiers
*/
func (this *WindowFrame) WindowFrameModifiers() uint32 {
	return this.windowFrameModifiers
}

/*
 Check Any given modifier is set
*/
func (this *WindowFrame) HasModifier(modifier uint32) bool {
	return (this.WindowFrameModifiers() & modifier) != 0
}

/*
  ROWS window frame unit type
*/

func (this *WindowFrame) RowsWindowFrame() bool {
	return this.HasModifier(WINDOW_FRAME_ROWS)
}

/*
  RANGE window frame unit type
*/
func (this *WindowFrame) RangeWindowFrame() bool {
	return this.HasModifier(WINDOW_FRAME_RANGE)
}

/*
  GROUPS window frame unit type
*/
func (this *WindowFrame) GroupsWindowFrame() bool {
	return this.HasModifier(WINDOW_FRAME_GROUPS)
}

/*
  Window Frame has Exclude clause (Required handle of exclude clause)
*/
func (this *WindowFrame) WindowFrameHasExclude() bool {
	return this.HasModifier(WINDOW_FRAME_EXCLUDE_CURRENT_ROW | WINDOW_FRAME_EXCLUDE_GROUP | WINDOW_FRAME_EXCLUDE_TIES)
}

/*
 non-constant Expressions used insdide window frame
*/
func (this *WindowFrame) Expressions() expression.Expressions {
	rv := make(expression.Expressions, 0, 2)
	for _, wfe := range this.WindowFrameExtents() {
		if wfe.ValueExpression() != nil {
			rv = append(rv, wfe.ValueExpression())
		}
	}
	return rv
}

/*
 Map expressions
*/
func (this *WindowFrame) MapExpressions(mapper expression.Mapper) (err error) {
	for _, wfe := range this.WindowFrameExtents() {
		if wfe.ValueExpression() != nil {
			wfe.valueExpr, err = mapper.Map(wfe.valueExpr)
			if err != nil {
				return
			}
		}
	}
	return
}

/*
Represents the Window frame extent in OLAP  window frame clause. Type
WindowFrameExtent is a struct containing the value expression and modifier bits
*/

type WindowFrameExtents []*WindowFrameExtent

type WindowFrameExtent struct {
	valueExpr expression.Expression
	modifiers uint32
}

func NewWindowFrameExtent(valueExpr expression.Expression, modifiers uint32) *WindowFrameExtent {
	return &WindowFrameExtent{
		valueExpr: valueExpr,
		modifiers: modifiers,
	}
}

/*
 Copy of window frame extents
*/

func (this WindowFrameExtents) Copy() WindowFrameExtents {
	rv := make(WindowFrameExtents, len(this))
	for i, v := range this {
		if v != nil {
			rv[i] = v.Copy()
		}
	}

	return rv
}

/*
 String representation of window frame extent
*/
func (this *WindowFrameExtent) String() (s string) {
	if this.HasModifier(WINDOW_FRAME_VALUE_PRECEDING) {
		s += " " + this.ValueExpression().String() + " PRECEDING"
	} else if this.HasModifier(WINDOW_FRAME_VALUE_FOLLOWING) {
		s += " " + this.ValueExpression().String() + " FOLLOWING"
	} else if this.HasModifier(WINDOW_FRAME_UNBOUNDED_PRECEDING) {
		s += " UNBOUNDED PRECEDING"
	} else if this.HasModifier(WINDOW_FRAME_UNBOUNDED_FOLLOWING) {
		s += " UNBOUNDED FOLLOWING"
	} else if this.HasModifier(WINDOW_FRAME_CURRENT_ROW) {
		s += " CURRENT ROW"
	}

	return s
}

/*
 Copy window frame extent
*/
func (this *WindowFrameExtent) Copy() *WindowFrameExtent {
	rv := &WindowFrameExtent{
		modifiers: this.Modifiers(),
	}

	if this.ValueExpression() != nil {
		rv.valueExpr = this.ValueExpression().Copy()
	}

	return rv
}

/*
 Window frame extent value expression
*/
func (this *WindowFrameExtent) ValueExpression() expression.Expression {
	return this.valueExpr
}

/*
 Window frame extent modifiers
*/
func (this *WindowFrameExtent) Modifiers() uint32 {
	return this.modifiers
}

/*
 Window frame extent has Any of the modifier set
*/
func (this *WindowFrameExtent) HasModifier(modifier uint32) bool {
	return (this.modifiers & modifier) != 0
}
