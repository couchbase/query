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
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

func (this *SemChecker) VisitCreatePrimaryIndex(stmt *algebra.CreatePrimaryIndex) (interface{}, error) {
	if stmt.Using() == datastore.FTS {
		return nil, errors.NewIndexNotAllowed("Primary index with USING FTS", "")
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateIndex(stmt *algebra.CreateIndex) (interface{}, error) {
	gsi := stmt.Using() == datastore.GSI || stmt.Using() == datastore.DEFAULT
	if !gsi && stmt.Partition() != nil {
		return nil, errors.NewIndexNotAllowed("PARTITION BY USING FTS", "")
	}

	for _, expr := range stmt.Expressions() {
		if !expr.Indexable() || expr.Value() != nil {
			return nil, errors.NewCreateIndexNotIndexable(expr.String(), expr.ErrorContext())
		}
	}

	nvectors := 0
	nkeys := 0
	for i, term := range stmt.Keys() {
		expr := term.Expression()
		if _, ok := expr.(*expression.Self); ok {
			return nil, errors.NewCreateIndexSelf(expr.String(), expr.ErrorContext())
		}
		all, ok := expr.(*expression.All)
		if !gsi {
			if term.HasAttribute(algebra.IK_MISSING | algebra.IK_ASC | algebra.IK_DESC | algebra.IK_VECTOR) {
				return nil, errors.NewIndexNotAllowed("Index attributes USING FTS", "")
			} else if ok {
				return nil, errors.NewIndexNotAllowed("Array Index USING FTS", "")
			}
		} else if term.HasAttribute(algebra.IK_VECTOR) {
			if !this.hasSemFlag(_SEM_ENTERPRISE) {
				return nil, errors.NewEnterpriseFeature("Index with vector key", "semantics.visit_create_index")
			}

			nvectors++
			indexKey := expr.String()
			if term.HasAttribute(algebra.IK_MISSING) {
				return nil, errors.NewVectorIndexAttrError("INCLUDE MISSING", indexKey)
			}
			if term.HasAttribute(algebra.IK_DESC) {
				return nil, errors.NewVectorIndexAttrError("DESC", indexKey)
			}
			if ok && all.Distinct() {
				return nil, errors.NewVectorDistinctArrayKey()
			}
			switch expr.(type) {
			case *expression.ObjectConstruct, *expression.ArrayConstruct:
				return nil, errors.NewVectorConstantIndexKey(expr.String())
			}
		}

		if ok && all.Flatten() {
			if term.Attributes() != 0 {
				return nil, errors.NewCreateIndexAttribute(expr.String(), expr.ErrorContext())
			}

			fk := all.FlattenKeys()
			for pos, fke := range fk.Operands() {
				nkeys++
				if !fke.Indexable() || fke.Value() != nil {
					return nil, errors.NewCreateIndexNotIndexable(fke.String(), fke.ErrorContext())
				}
				if fk.HasMissing(pos) && (i > 0 || pos > 0 || !gsi) {
					return nil, errors.NewCreateIndexAttributeMissing(fke.String(), fke.ErrorContext())
				}
				if fk.HasVector(pos) {
					return nil, errors.NewIndexNotAllowed("Array Index with FLATTEN_KEYS using Vector Index Key", fke.String())
				}
			}
		} else {
			nkeys++
			if term.HasAttribute(algebra.IK_VECTOR) && ok {
				return nil, errors.NewIndexNotAllowed("Array Index using Vector Index Key", expr.String())
			}

		}
		if term.HasAttribute(algebra.IK_MISSING) && (i > 0 || !gsi) {
			return nil, errors.NewCreateIndexAttributeMissing(expr.String(), expr.ErrorContext())
		}
		if nvectors > 1 {
			return nil, errors.NewVectorIndexSingleVector(stmt.Name())
		}
	}

	if gsi && stmt.Vector() {
		if nkeys > 1 {
			return nil, errors.NewVectorIndexSingleKey(stmt.Name())
		}
		if nvectors == 0 {
			return nil, errors.NewVectorIndexNoVector(stmt.Name())
		}
	}

	if err := semCheckFlattenKeys(stmt.Expressions()); err != nil {
		return nil, err
	}

	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropIndex(stmt *algebra.DropIndex) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitAlterIndex(stmt *algebra.AlterIndex) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitBuildIndexes(stmt *algebra.BuildIndexes) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

type bucketFieldDef struct {
	typ      value.Type
	canAlter bool
	minValue interface{}
	maxValue interface{}
	values   []interface{}
}

var bucketFieldDefinitions = map[string]bucketFieldDef{
	"name":                   {value.STRING, false, nil, nil, nil},
	"bucketType":             {value.STRING, false, nil, nil, []interface{}{"couchbase", "ephemeral", "memcached"}},
	"replicaIndex":           {value.NUMBER, false, 0, 1, []interface{}{0, 1}},
	"conflictResolutionType": {value.STRING, false, nil, nil, []interface{}{"seqno", "lww"}},
	"storageBackend":         {value.STRING, false, nil, nil, []interface{}{"couchstore", "magma"}},

	"evictionPolicy": {value.STRING, true, nil, nil, []interface{}{
		"valueOnly", "fullEviction", "noEviction", "nruEviction"}},
	"durabilityMinLevel": {value.STRING, true, nil, nil, []interface{}{
		"none", "majority", "majorityAndPersistActive", "persistToMajority"}},

	"ramQuota":                                   {value.NUMBER, true, 100, nil, nil},
	"threadsNumber":                              {value.NUMBER, true, 3, 8, []interface{}{3, 8}},
	"replicaNumber":                              {value.NUMBER, true, 0, 3, []interface{}{0, 1, 2, 3}},
	"compressionMode":                            {value.STRING, true, nil, nil, []interface{}{"off", "passive", "active"}},
	"maxTTL":                                     {value.NUMBER, true, 0, 2147483647, nil},
	"flushEnabled":                               {value.NUMBER, true, 0, 1, []interface{}{0, 1}},
	"magmaSeqTreeDataBlockSize":                  {value.NUMBER, true, 4096, 131072, nil},
	"historyRetentionCollectionDefault":          {value.BOOLEAN, true, nil, nil, nil},
	"historyRetentionBytes":                      {value.NUMBER, true, 2147483648, nil, nil},
	"historyRetentionSeconds":                    {value.NUMBER, true, 0, nil, nil},
	"autoCompactionDefined":                      {value.BOOLEAN, true, nil, nil, nil},
	"parallelDBAndViewCompaction":                {value.BOOLEAN, true, nil, nil, nil},
	"databaseFragmentationThreshold[percentage]": {value.NUMBER, true, 0, 100, nil},
	"databaseFragmentationThreshold[size]":       {value.NUMBER, true, 1, nil, nil},
	"viewFragmentationThreshold[percentage]":     {value.NUMBER, true, 0, 100, nil},
	"viewFragmentationThreshold[size]":           {value.NUMBER, true, 1, nil, nil},
	"purgeInterval":                              {value.NUMBER, true, 0.01, 60, nil},
	"allowedTimePeriod[fromHour]":                {value.NUMBER, true, 0, 23, nil},
	"allowedTimePeriod[fromMinute]":              {value.NUMBER, true, 0, 59, nil},
	"allowedTimePeriod[toHour]":                  {value.NUMBER, true, 0, 23, nil},
	"allowedTimePeriod[toMinute]":                {value.NUMBER, true, 0, 59, nil},
	"allowedTimePeriod[abortOutside]":            {value.BOOLEAN, true, nil, nil, nil},
	"magmaFragmentationPercentage":               {value.NUMBER, true, 0, 100, nil},
}

func validateBucketOptions(with value.Value, alter bool) errors.Error {
	if with == nil || with.Type() != value.OBJECT {
		return errors.NewSemanticsWithCauseError(fmt.Errorf("Must be a constant OBJECT"), "Invalid WITH clause value")
	}
	for k, i := range with.Fields() {
		v := value.NewValue(i)
		if def, ok := bucketFieldDefinitions[k]; ok {
			if alter && !def.canAlter {
				return errors.NewWithInvalidOptionError(k)
			}
			if v.Type() != def.typ {
				s := fmt.Sprintf("%s expected", strings.ToLower(def.typ.String()))
				s = strings.ToUpper(s[:1]) + s[1:]
				return errors.NewWithInvalidValueError(k, s)
			}
			if def.typ == value.NUMBER {
				if def.minValue != nil {
					_, expectFloat := def.minValue.(float64)
					if _, ok := value.IsIntValue(v); !ok && !expectFloat {
						return errors.NewWithInvalidValueError(k, "Integer expected")
					}
					min := value.NewValue(def.minValue)
					if v.Compare(min) == value.NEG_ONE_VALUE {
						return errors.NewWithInvalidValueError(k, fmt.Sprintf("Value >= %v expected", def.minValue))
					}
				}
				if def.maxValue != nil {
					max := value.NewValue(def.maxValue)
					if v.Compare(max) == value.ONE_VALUE {
						return errors.NewWithInvalidValueError(k, fmt.Sprintf("Value <= %v expected", def.maxValue))
					}
				}
			}
			if def.values != nil {
				found := false
				for i := range def.values {
					if value.NewValue(def.values[i]) == v {
						found = true
						break
					}
				}
				if !found {
					s := fmt.Sprintf("Value must be one of: %v", def.values[0])
					for i := 1; i < len(def.values); i++ {
						s += fmt.Sprintf(", %v", def.values[i])
					}
					return errors.NewWithInvalidValueError(k, s)
				}
			}
		} else {
			return errors.NewWithInvalidOptionError(k)
		}
	}
	return nil
}

func (this *SemChecker) VisitCreateBucket(stmt *algebra.CreateBucket) (interface{}, error) {
	err := validateBucketOptions(stmt.With(), false)
	if err != nil {
		return nil, err
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitAlterBucket(stmt *algebra.AlterBucket) (interface{}, error) {
	err := validateBucketOptions(stmt.With(), true)
	if err != nil {
		return nil, err
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropBucket(stmt *algebra.DropBucket) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateScope(stmt *algebra.CreateScope) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropScope(stmt *algebra.DropScope) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateCollection(stmt *algebra.CreateCollection) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropCollection(stmt *algebra.DropCollection) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitFlushCollection(stmt *algebra.FlushCollection) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

type CheckFlattenKeys struct {
	expression.MapperBase
	flattenKeys expression.Expression
}

/* FLATTEN_KEYS() function allowed only in
   -   Array indexing key deepest value mapping
   -   Not surounded by any function
   -   No recursive
*/

func NewCheckFlattenKeys() *CheckFlattenKeys {
	rv := &CheckFlattenKeys{}
	rv.SetMapper(rv)
	rv.SetMapFunc(func(expr expression.Expression) (expression.Expression, error) {
		if _, ok := expr.(*expression.FlattenKeys); ok && rv.flattenKeys != expr {
			return expr, errors.NewFlattenKeys(expr.String(), expr.ErrorContext())
		}
		return expr, expr.MapChildren(rv)
	})
	return rv
}

func semCheckFlattenKeys(exprs expression.Expressions) (err error) {
	cfk := NewCheckFlattenKeys()
	for _, expr := range exprs {
		if all, ok := expr.(*expression.All); ok && all.Flatten() {
			cfk.flattenKeys = all.FlattenKeys()
		} else {
			cfk.flattenKeys = nil
		}

		if _, err = cfk.Map(expr); err != nil {
			return err
		}
	}

	return err
}

func (this *SemChecker) VisitCreateSequence(stmt *algebra.CreateSequence) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropSequence(stmt *algebra.DropSequence) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitAlterSequence(stmt *algebra.AlterSequence) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}
