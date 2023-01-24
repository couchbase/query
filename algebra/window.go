//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"fmt"
	"sort"

	"github.com/couchbase/query/expression"
)

/*
 Window Term
  windowName    WindowTerm is referenced by WindowName
  partitionBy   PARTITION BY clause
  orderBy       ORDER BY BY clause
  windowFrame   window frame clause
  asWindowName  Alias of window name in WINODW CLAUSE
  flag          referenced by WINDOW NAME
*/

type WindowTerm struct {
	windowName   string
	partitionBy  expression.Expressions
	orderBy      *Order
	windowFrame  *WindowFrame
	asWindowName string
	flag         bool
}

type WindowTerms []*WindowTerm

func NewWindowTerm(name string, pby expression.Expressions, oby *Order,
	windowFrame *WindowFrame, flag bool) *WindowTerm {
	rv := &WindowTerm{
		windowName:  name,
		partitionBy: removeDuplicatePbys(pby),
		orderBy:     oby,
		windowFrame: windowFrame,
		flag:        flag,
	}

	return rv
}

func (this *WindowTerm) SetAsWindowName(as string) {
	this.asWindowName = as
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
	rv.windowName = this.WindowName()
	rv.asWindowName = this.AsWindowName()
	rv.flag = this.flag

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

func (this *WindowTerm) WindowName() string {
	return this.windowName
}

func (this *WindowTerm) AsWindowName() string {
	return this.asWindowName
}

func (this *WindowTerm) RewriteToNewWindowTerm(wterms WindowTerms) (err error) {
	wterm := this
	rv := wterm
	if wterm.WindowName() == "" {
		return nil
	}

	if err = wterm.ValidateWindowTerm(wterms, len(wterms)-1, 0); err != nil {
		return err
	}

	for i := len(wterms) - 1; i >= 0; i-- {
		w := wterms[i]
		if wterm.WindowName() == w.AsWindowName() {
			if w.partitionBy != nil {
				rv.partitionBy = w.partitionBy.Copy()
			}

			if w.orderBy != nil {
				rv.orderBy = w.orderBy.Copy()
			}

			if w.windowFrame != nil {
				rv.windowFrame = w.windowFrame.Copy()
			}

			if w.WindowName() == "" {
				return nil
			}
			wterm = w
		}
	}

	return fmt.Errorf("Not able to translate window %s", rv.WindowName())
}

func (this *WindowTerm) ValidateWindowTerm(wterms WindowTerms, start, end int) (err error) {
	wterm := this
	if wterm.WindowName() == "" {
		return nil
	}

	for i := start; i >= end; i-- {
		w := wterms[i]
		errStr := ""
		if wterm.WindowName() == w.AsWindowName() {
			if len(wterm.PartitionBy()) > 0 {
				errStr = "partitioning"
			} else if wterm.OrderBy() != nil && w.OrderBy() != nil {
				errStr = "ordering"
			} else if !wterm.flag && w.WindowFrame() != nil {
				errStr = "framing"
			}
			if errStr != "" {
				return fmt.Errorf("Window %s shall not have a window %s clause", w.AsWindowName(), errStr)
			}
			if w.WindowName() == "" {
				return nil
			}
			wterm = w
		}
	}
	return fmt.Errorf("Window %s not in the scope of window clause", wterm.WindowName())
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
	return this.wnameString(false)
}

func (this *WindowTerm) wnameString(flag bool) (s string) {
	if flag {
		s += " " + this.AsWindowName() + " AS"
	} else {
		s += " OVER"
	}
	s += " ("
	if flag && this.WindowName() != "" {
		s += " " + this.WindowName() + " "
	}

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

func (this WindowTerms) ValidateWindowTerms() (err error) {
	wnames := make(map[string]bool, len(this))
	for i, w := range this {
		if _, ok := wnames[w.AsWindowName()]; ok {
			return fmt.Errorf("Duplicate window clause alias %s.", w.AsWindowName())
		}
		wnames[w.AsWindowName()] = true
		if w.WindowName() == "" && w.partitionBy == nil && w.orderBy == nil && w.windowFrame != nil {
			return fmt.Errorf("Only window frame clause is not allowed in named window: %s", w.AsWindowName())
		}
		if err = w.ValidateWindowTerm(this, i, 0); err != nil {
			return err
		}
	}

	return nil
}

func (this WindowTerms) Formalize(f *expression.Formalizer) (err error) {
	return this.MapExpressions(f)
}

func (this WindowTerms) MapExpressions(mapper expression.Mapper) (err error) {
	for _, w := range this {
		if err = w.MapExpressions(mapper); err != nil {
			return err
		}
	}
	return nil
}

func (this WindowTerms) Expressions() (rv expression.Expressions) {
	for _, w := range this {
		rv = append(rv, w.Expressions()...)
	}
	return rv
}

func (this WindowTerms) String() (rv string) {
	for i, w := range this {
		if i == 0 {
			rv += "WINDOW "
		} else {
			rv += ", "
		}
		rv += w.wnameString(true)
	}
	return rv
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
