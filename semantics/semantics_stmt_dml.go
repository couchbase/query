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

func (this *SemChecker) VisitSelect(stmt *algebra.Select) (interface{}, error) {
	return stmt.Subresult().Accept(this)
}

func (this *SemChecker) VisitInsert(stmt *algebra.Insert) (interface{}, error) {
	if stmt.Select() != nil {
		return stmt.Select().Accept(this)
	}
	return nil, nil
}

func (this *SemChecker) VisitUpsert(stmt *algebra.Upsert) (interface{}, error) {
	if stmt.Select() != nil {
		return stmt.Select().Accept(this)
	}
	return nil, nil
}

func (this *SemChecker) VisitDelete(stmt *algebra.Delete) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitUpdate(stmt *algebra.Update) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitMerge(stmt *algebra.Merge) (interface{}, error) {

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
	if source.SubqueryTerm() != nil {
		return source.SubqueryTerm().Accept(this)
	} else if source.ExpressionTerm() != nil {
		return source.ExpressionTerm().Accept(this)
	} else if source.From() != nil {
		return source.From().Accept(this)
	} else {
		return nil, errors.NewMergeMissingSourceError()
	}
	return nil, nil
}

func (this *SemChecker) VisitExplain(stmt *algebra.Explain) (interface{}, error) {
	return stmt.Statement().Accept(this)
}

func (this *SemChecker) VisitPrepare(stmt *algebra.Prepare) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitExecute(stmt *algebra.Execute) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitInferKeyspace(stmt *algebra.InferKeyspace) (interface{}, error) {
	return nil, nil
}
