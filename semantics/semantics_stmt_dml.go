//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package semantics

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

func (this *SemChecker) VisitSelect(stmt *algebra.Select) (r interface{}, err error) {
	prevStmtType := this.stmtType
	prevObyVectorDist := this.hasSemFlag(_SEM_ORDERBY_VECTOR_DIST)
	defer func() {
		this.stmtType = prevStmtType
		if !prevObyVectorDist {
			this.unsetSemFlag(_SEM_ORDERBY_VECTOR_DIST)
		}
	}()
	this.stmtType = stmt.Type()
	if stmt.Order() != nil {
		for _, term := range stmt.Order().Terms() {
			switch term.Expression().(type) {
			case *expression.VectorDistance, *expression.ApproxVectorDistance:
				if !this.hasSemFlag(_SEM_ENTERPRISE) {
					return nil, errors.NewEnterpriseFeature("Vector Search Function", "semantics.visit_select")
				}
				this.setSemFlag(_SEM_ORDERBY_VECTOR_DIST)
			}
		}
	}

	if r, err = stmt.Subresult().Accept(this); err != nil {
		return r, err
	}

	if stmt.With() != nil {
		if stmt.With().IsRecursive() {
			// if recursive hint is used :- order, limit, offset, group & aggregates are not allowed
			// order limit offset is handled in splitting
			this.setSemFlag(_SEM_WITH_RECURSIVE)
			err = stmt.With().MapExpressions(this)
			this.unsetSemFlag(_SEM_WITH_RECURSIVE)
			if err != nil {
				return nil, err
			}

			for _, binding := range stmt.With().Bindings() {
				if cyc := binding.CycleFields(); cyc != nil {
					var err error
					err = validateCycleFields(cyc)
					if err != nil {
						return nil, errors.NewCycleFieldsValidationFailedError(err, binding.Alias())
					}
				}
			}

		} else {
			if err = stmt.With().MapExpressions(this); err != nil {
				return nil, err
			}
		}
	}

	if stmt.Order() != nil {
		if this.hasSemFlag(_SEM_WITH_RECURSIVE) {
			return nil, errors.NewRecursiveWithSemanticError("Order not allowed")
		}

		// Validate the ORDER BY direction and NULLS position expressions
		err = validateOrderBySemantics(stmt.Order())
		if err != nil {
			return nil, err
		}

		if err = stmt.Order().MapExpressions(this); err != nil {
			return nil, err
		}
		for _, term := range stmt.Order().Terms() {
			switch term.Expression().(type) {
			case *expression.ApproxVectorDistance, *expression.VectorDistance:
				ce := term.DescendingExpr()
				ne := term.NullsPosExpr()
				if ce != nil && ce.Value() == nil {
					return nil, errors.NewVectorOrderConst(term.String(), "collation (ASC/DESC)")
				} else if ne != nil && ne.Value() == nil {
					return nil, errors.NewVectorOrderConst(term.String(), "NULLS position (FIRST/LAST)")
				}
				if term.Descending(nil, nil) {
					return nil, errors.NewVectorOrderOption(term.String(), "DESC")
				} else if !term.NullsLast(nil, nil) {
					return nil, errors.NewVectorOrderOption(term.String(), "NULLS FIRST")
				}
			}
		}
	}

	if stmt.Offset() != nil {
		if this.hasSemFlag(_SEM_WITH_RECURSIVE) {
			return nil, errors.NewRecursiveWithSemanticError("Offset not allowed")
		}
		if _, err = this.Map(stmt.Offset()); err != nil {
			return nil, err
		}
	}

	if stmt.Limit() != nil {
		if this.hasSemFlag(_SEM_WITH_RECURSIVE) {
			return nil, errors.NewRecursiveWithSemanticError("Limit not allowed")
		}
		if _, err = this.Map(stmt.Limit()); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (this *SemChecker) VisitInsert(stmt *algebra.Insert) (interface{}, error) {
	if stmt.Select() != nil {
		if r, err := stmt.Select().Accept(this); err != nil {
			return r, err
		}
	}

	return nil, stmt.MapExpressionsNoSelect(this)
}

func (this *SemChecker) VisitUpsert(stmt *algebra.Upsert) (interface{}, error) {
	if stmt.Select() != nil {
		if r, err := stmt.Select().Accept(this); err != nil {
			return r, err
		}
	}

	return nil, stmt.MapExpressionsNoSelect(this)
}

func (this *SemChecker) VisitDelete(stmt *algebra.Delete) (r interface{}, err error) {
	if stmt.KeyspaceRef().Path() == nil {
		if stmt.Keys() == nil {
			return nil, errors.NewMissingUseKeysError("<placeholder>", "semantic.delete")
		}
		if stmt.Indexes() != nil {
			return nil, errors.NewHasUseIndexesError("<placeholder>", "semantic.delete")
		}
	}

	if stmt.Keys() != nil {
		if _, err = this.Map(stmt.Keys()); err != nil {
			return nil, err
		}
	}

	if stmt.Let() != nil {
		if err = stmt.Let().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	if stmt.Where() != nil {
		this.setSemFlag(_SEM_WHERE)
		_, err = this.Map(stmt.Where())
		this.unsetSemFlag(_SEM_WHERE)
		if err != nil {
			return nil, err
		}
	}

	if stmt.Limit() != nil {
		if _, err = this.Map(stmt.Limit()); err != nil {
			return nil, err
		}
	}

	if stmt.Returning() != nil {
		if err = stmt.Returning().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (this *SemChecker) VisitUpdate(stmt *algebra.Update) (r interface{}, err error) {
	if stmt.KeyspaceRef().Path() == nil {
		if stmt.Keys() == nil {
			return nil, errors.NewMissingUseKeysError("<placeholder>", "semantic.update")
		}
		if stmt.Indexes() != nil {
			return nil, errors.NewHasUseIndexesError("<placeholder>", "semantic.update")
		}
	}

	if stmt.Keys() != nil {
		if _, err = this.Map(stmt.Keys()); err != nil {
			return nil, err
		}
	}

	if stmt.Set() != nil {
		if err = stmt.Set().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	if stmt.Unset() != nil {
		if err = stmt.Unset().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	if stmt.Let() != nil {
		if err = stmt.Let().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	if stmt.Where() != nil {
		this.setSemFlag(_SEM_WHERE)
		_, err = this.Map(stmt.Where())
		this.unsetSemFlag(_SEM_WHERE)
		if err != nil {
			return nil, err
		}
	}

	if stmt.Limit() != nil {
		if _, err = this.Map(stmt.Limit()); err != nil {
			return nil, err
		}
	}

	if stmt.Returning() != nil {
		if err = stmt.Returning().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (this *SemChecker) VisitMerge(stmt *algebra.Merge) (r interface{}, err error) {

	actions := stmt.Actions()
	insert := actions.Insert()
	if stmt.IsOnKey() {
		if stmt.Indexes() != nil {
			return nil, errors.NewMergeNoIndexHintError()
		}
		if insert != nil && insert.Key() != nil {
			return nil, errors.NewMergeInsertNoKeyError()
		}
	} else {
		if insert != nil && insert.Key() == nil {
			return nil, errors.NewMergeInsertMissingKeyError()
		}
	}

	source := stmt.Source()
	if stmt.IsOnKey() {
		if source.SubqueryTerm() != nil {
			if source.SubqueryTerm().JoinHint() != algebra.JOIN_HINT_NONE {
				return nil, errors.NewMergeNoJoinHintError()
			}
		} else if source.ExpressionTerm() != nil {
			if source.ExpressionTerm().JoinHint() != algebra.JOIN_HINT_NONE {
				return nil, errors.NewMergeNoJoinHintError()
			}
		} else if source.From() != nil {
			if source.From().JoinHint() != algebra.JOIN_HINT_NONE {
				return nil, errors.NewMergeNoJoinHintError()
			}
		}
	}

	if source.SubqueryTerm() != nil {
		return source.SubqueryTerm().Accept(this)
	} else if source.ExpressionTerm() != nil {
		return source.ExpressionTerm().Accept(this)
	} else if source.From() != nil {
		return source.From().Accept(this)
	} else {
		return nil, errors.NewMergeMissingSourceError()
	}

	if stmt.On() != nil {
		if !stmt.IsOnKey() {
			this.setSemFlag(_SEM_ON)
		}
		_, err = this.Map(stmt.On())
		this.unsetSemFlag(_SEM_ON)
		if err != nil {
			return nil, err
		}
	}

	if stmt.Let() != nil {
		if err = stmt.Let().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	if err = stmt.Actions().MapExpressions(this); err != nil {
		return nil, err
	}

	if stmt.Limit() != nil {
		if _, err = this.Map(stmt.Limit()); err != nil {
			return nil, err
		}
	}

	if stmt.Returning() != nil {
		if err = stmt.Returning().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	return nil, stmt.MapExpressions(this)
}

// Validate the semantics of the ORDER BY direction and NULLS position
func validateOrderBySemantics(order *algebra.Order) error {

	for _, s := range order.Terms() {

		desc := s.DescendingExpr()
		if desc != nil {
			// ORDER BY direction must be a static expression
			if desc.Static() == nil {
				return errors.NewOrderByValidationError("ORDER BY direction", desc.String())
			}

			// ORDER BY direction can only be a constant or named, positional, function parameters
			switch d := desc.(type) {
			case *expression.Constant:
				break
			case expression.NamedParameter:
				break
			case expression.PositionalParameter:
				break
			case *expression.Identifier:
				{
					// If the ORDER BY direction is an identifier it must be a function variable.
					if !d.IsFuncVariable() {
						return errors.NewOrderByValidationError("ORDER BY direction", d.String())
					}
					break
				}
			default:
				return errors.NewOrderByValidationError("ORDER BY direction", d.String())
			}
		}

		nullsPos := s.NullsPosExpr()
		if nullsPos != nil {

			// NULLS position must be a static expression
			if nullsPos.Static() == nil {
				return errors.NewOrderByValidationError("NULLS position", nullsPos.String())
			}

			// NULLS position can only be a constant or named, positional, function parameters
			switch n := nullsPos.(type) {
			case *expression.Constant:
				break
			case expression.NamedParameter:
				break
			case expression.PositionalParameter:
				break
			case *expression.Identifier:
				{
					// If the NULLS position is an identifier it must be a function variable
					if !n.IsFuncVariable() {
						return errors.NewOrderByValidationError("NULLS position", n.String())
					}
					break
				}
			default:
				return errors.NewOrderByValidationError("NULLS position", n.String())
			}
		}
	}
	return nil
}

// only alow identifier/field expression as cycle expression
func validateCycleFields(cycle expression.Expressions) error {

	for _, cycleFieldExpr := range cycle {
		switch c := cycleFieldExpr.(type) {
		case *expression.Identifier, *expression.Field:
			continue
		default:
			return fmt.Errorf("invalid cycle field expression term: %s only "+
				"identifier/path expressions are allowed", c)
		}
	}
	return nil
}
