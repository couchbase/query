//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package semantics

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
)

func (this *SemChecker) VisitSelect(stmt *algebra.Select) (r interface{}, err error) {
	prevStmtType := this.stmtType
	defer func() {
		this.stmtType = prevStmtType
	}()
	this.stmtType = stmt.Type()

	if r, err = stmt.Subresult().Accept(this); err != nil {
		return r, err
	}

	if stmt.Order() != nil {
		if err = stmt.Order().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	if stmt.Offset() != nil {
		if _, err = this.Map(stmt.Offset()); err != nil {
			return nil, err
		}
	}

	if stmt.Limit() != nil {
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

	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitUpsert(stmt *algebra.Upsert) (interface{}, error) {
	if stmt.Select() != nil {
		if r, err := stmt.Select().Accept(this); err != nil {
			return r, err
		}
	}

	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDelete(stmt *algebra.Delete) (r interface{}, err error) {
	if stmt.Keys() != nil {
		if _, err = this.Map(stmt.Keys()); err != nil {
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
