//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// Len

type Len struct {
	UnaryFunctionBase
}

func NewLen(operand Expression) Function {
	rv := &Len{}
	rv.Init("len", operand)

	rv.expr = rv
	return rv
}

func (this *Len) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Len) Type() value.Type { return value.NUMBER }

func (this *Len) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return evaluateLength(arg, false)
}

func (this *Len) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLen(operands[0])
	}
}

// Multi-byte aware variant

type MBLen struct {
	UnaryFunctionBase
}

func NewMBLen(operand Expression) Function {
	rv := &MBLen{}
	rv.Init("mb_len", operand)
	rv.expr = rv
	return rv
}

func (this *MBLen) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MBLen) Type() value.Type { return value.NUMBER }

func (this *MBLen) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return evaluateLength(arg, true)
}

func (this *MBLen) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMBLen(operands[0])
	}
}

func evaluateLength(arg value.Value, runes bool) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING:
		return value.MISSING_VALUE, nil
	case value.STRING:
		if runes {
			return value.NewValue(utf8.RuneCountInString(arg.ToString())), nil
		}
		return value.NewValue(arg.Size()), nil
	case value.OBJECT:
		oa := arg.Actual().(map[string]interface{})
		return value.NewValue(len(oa)), nil
	case value.ARRAY:
		aa := arg.Actual().([]interface{})
		return value.NewValue(len(aa)), nil
	case value.BINARY:
		return value.NewValue(arg.Size()), nil
	case value.BOOLEAN:
		return value.ONE_VALUE, nil
	case value.NUMBER:
		return value.NewValue(len(arg.ToString())), nil
	}
	return value.NULL_VALUE, nil
}

// Evaluate

type Evaluate struct {
	FunctionBase
}

func NewEvaluate(operands ...Expression) Function {
	rv := &Evaluate{}
	rv.Init("evaluate", operands...)

	rv.expr = rv
	return rv
}

func (this *Evaluate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Evaluate) Type() value.Type { return value.OBJECT }

func (this *Evaluate) Evaluate(item value.Value, context Context) (value.Value, error) {
	var stmt string
	var named map[string]value.Value
	var positional value.Values

	null := false
	missing := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			if arg.Type() == value.MISSING {
				missing = true
			} else if arg.Type() != value.STRING {
				null = true
			}
			stmt = arg.ToString()
		} else {
			if arg.Type() == value.OBJECT {
				act := arg.Actual().(map[string]interface{})
				named = make(map[string]value.Value, len(act))
				for k, v := range act {
					named[k] = value.NewValue(v)
				}
			} else if arg.Type() == value.ARRAY {
				act := arg.Actual().([]interface{})
				positional = make(value.Values, 0, len(act))
				for i := range act {
					positional = append(positional, value.NewValue(act[i]))
				}
			} else {
				null = true
			}
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	// only read-only statements are permitted
	pcontext, ok := context.(ParkableContext)
	if !ok {
		return value.NULL_VALUE, nil
	}
	rv, _, err := pcontext.ParkableEvaluateStatement(stmt, named, positional, false, true, false, "")
	if err != nil {
		// to help with diagnosing problems in the provided statement, we return the error encountered and not just the NULL_VALUE
		return value.NULL_VALUE, errors.NewEvaluationError(err, "statement")
	}
	return rv, nil
}

func (this *Evaluate) MinArgs() int { return 1 }

func (this *Evaluate) MaxArgs() int { return 2 }

func (this *Evaluate) Constructor() FunctionConstructor {
	return NewEvaluate
}

func (this *Evaluate) Indexable() bool {
	return false
}

// Finderr

type Finderr struct {
	UnaryFunctionBase
}

func NewFinderr(operand Expression) Function {
	rv := &Finderr{}
	rv.Init("finderr", operand)

	rv.expr = rv
	return rv
}

func (this *Finderr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Finderr) Type() value.Type { return value.OBJECT }

func (this *Finderr) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING && arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	if arg.Type() == value.NUMBER {
		ed := errors.DescribeError(errors.ErrorCode(value.AsNumberValue(arg).Int64()))
		if ed == nil {
			return value.NULL_VALUE, nil
		}
		b, err := json.Marshal(ed)
		if err != nil {
			return value.NULL_VALUE, nil
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(b, &m)
		if err != nil {
			return value.NULL_VALUE, nil
		}
		res := make([]interface{}, 1)
		res[0] = m
		return value.NewValue(res), nil
	} else {
		errs := errors.SearchErrors(arg.ToString())
		if len(errs) == 0 {
			return value.NULL_VALUE, nil
		}
		res := make([]interface{}, 0, len(errs))
		for _, ed := range errs {
			b, err := json.Marshal(ed)
			if err != nil {
				return value.NULL_VALUE, nil
			}
			m := make(map[string]interface{})
			err = json.Unmarshal(b, &m)
			if err != nil {
				return value.NULL_VALUE, nil
			}
			res = append(res, value.NewValue(m))
		}
		return value.NewValue(res), nil
	}
}

func (this *Finderr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewFinderr(operands[0])
	}
}

func (this *Finderr) Indexable() bool {
	return false
}

// ExtractDDL

const (
	_BUCKET_INFO = 1 << iota
	_SCOPE_INFO
	_COLLECTION_INFO
	_INDEX_INFO
	_SEQUENCE_INFO
)

type ExtractDDL struct {
	FunctionBase
}

func NewExtractDDL(operands ...Expression) Function {
	rv := &ExtractDDL{}
	rv.Init("extractddl", operands...)

	rv.expr = rv
	return rv
}

func (this *ExtractDDL) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ExtractDDL) Type() value.Type { return value.ARRAY }

// This relies entirely on the system catalogue permissions to enforce what can and can't be seen: if you could see it in a system
// catalogue query, you can see it in the result of this function
func (this *ExtractDDL) Evaluate(item value.Value, context Context) (value.Value, error) {
	var filter value.Value
	var with value.Value
	var err error

	if len(this.operands) > 0 {
		filter, err = this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if filter.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if filter.Type() == value.NULL {
			filter = nil
		} else if filter.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
	}

	if len(this.operands) > 1 {
		with, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if with.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if with.Type() != value.OBJECT {
			return value.NULL_VALUE, nil
		}
	} else {
		with = value.NewValue(map[string]interface{}{})
	}

	res := make([]interface{}, 0, 32)
	args := make(value.Values, 0, 1)

	var buf strings.Builder
	buf.Grow(128) // Pre-allocate buffer for the initial query
	buf.WriteString("SELECT DISTINCT RAW name FROM system:keyspaces WHERE `namespace` = 'default' AND `bucket` IS NOT VALUED ")
	if filter != nil && filter.ToString() != "" {
		buf.WriteString(" AND name LIKE ?")
		args = append(args, filter)
	}
	buf.WriteString(" ORDER BY name")
	stmt := buf.String()
	v, _, err := context.EvaluateStatement(stmt, nil, args, false, true, false, "")
	if err != nil {
		return value.MISSING_VALUE, err
	}
	buckets := v.Actual().([]interface{})
	if len(buckets) == 0 {
		return value.MISSING_VALUE, errors.NewWarning("No bucket(s) found or missing permissions.")
	}

	flags := _BUCKET_INFO | _SCOPE_INFO | _COLLECTION_INFO | _INDEX_INFO | _SEQUENCE_INFO
	if f, ok := with.Field("flags"); ok {
		if f.Type() == value.STRING {
			f = value.NewValue([]interface{}{f})
		}
		if f.Type() == value.NUMBER {
			flags = int(value.AsNumberValue(f).Int64())
		} else if f.Type() == value.ARRAY {
			act := f.Actual().([]interface{})
			flags = 0
			for i := range act {
				switch t := act[i].(type) {
				case value.Value:
					switch t.ToString() {
					case "bucket":
						flags |= _BUCKET_INFO
					case "scope":
						flags |= _SCOPE_INFO
					case "collection":
						flags |= _COLLECTION_INFO
					case "index":
						flags |= _INDEX_INFO
					case "sequence":
						flags |= _SEQUENCE_INFO
					default:
						return value.MISSING_VALUE, errors.NewWarning(fmt.Sprintf("Invalid flag: %v", act[i]))
					}
				default:
					return value.MISSING_VALUE, errors.NewWarning(fmt.Sprintf("Invalid flag: %v", act[i]))
				}
			}
		} else {
			return value.MISSING_VALUE, errors.NewWarning("Invalid flags.")
		}
	}
	if flags&(_BUCKET_INFO|_SCOPE_INFO|_COLLECTION_INFO|_INDEX_INFO|_SEQUENCE_INFO) == 0 {
		return value.NULL_VALUE, errors.NewWarning("Flags exclude all data.")
	}

	var sb, bucketInfoBuf strings.Builder
	for i := range buckets {
		bucketInfoBuf.Reset()
		bucketInfoBuf.Grow(256) // Pre-allocate buffer for bucket info
		bucket := buckets[i].(string)
		posArg := value.Values{value.NewValue(bucket)}

		if flags&_BUCKET_INFO != 0 {
			stmt = "SELECT bucketType, storageBackend, quota.rawRAM ramQuota, replicaNumber, replicaIndex, maxTTL, compressionMode, " +
				"conflictResolutionType, evictionPolicy, threadsNumber, durabilityMinLevel, purgeInterval," +
				"controllers.`flush` AS flushEnabled, magmaSeqTreeDataBlockSize, historyRetentionCollectionDefault," +
				"historyRetentionBytes, historyRetentionSeconds, numVBuckets, autoCompactionSettings.parallelDBAndViewCompaction," +
				"autoCompactionSettings.databaseFragmentationThreshold.percentage AS " +
				"`databaseFragmentationThreshold[percentage]`," +
				"autoCompactionSettings.databaseFragmentationThreshold.size AS `databaseFragmentationThreshold[size]`," +
				"autoCompactionSettings.viewFragmentationThreshold.percentage AS `viewFragmentationThreshold[percentage]`," +
				"autoCompactionSettings.viewFragmentationThreshold.size AS `viewFragmentationThreshold[size]`," +
				"autoCompactionSettings.allowedTimePeriod.fromHour AS `allowedTimePeriod[fromHour]`," +
				"autoCompactionSettings.allowedTimePeriod.fromMinute AS `allowedTimePeriod[fromMinute]`," +
				"autoCompactionSettings.allowedTimePeriod.toHour AS `allowedTimePeriod[toHour]`," +
				"autoCompactionSettings.allowedTimePeriod.toMinute AS `allowedTimePeriod[toMinute]`," +
				"autoCompactionSettings.allowedTimePeriod.abortOutside AS `allowedTimePeriod[abortOutside]`," +
				"autoCompactionSettings.magmaFragmentationPercentage," +
				"CASE WHEN type(autoCompactionSettings) = 'object' THEN TRUE ELSE MISSING END AS autoCompactionDefined" +
				" FROM system:bucket_info USE KEYS[?]"
			v, _, err = context.EvaluateStatement(stmt, nil, posArg, false, true, false, "")
			if err != nil {
				return value.MISSING_VALUE, err
			}
			bucketInfoBuf.WriteString("CREATE BUCKET `")
			bucketInfoBuf.WriteString(bucket)
			bucketInfoBuf.WriteRune('`')
			v, ok := v.Index(0)
			if ok {
				sb.Reset()
				names := make([]string, 27)
				for n, _ := range v.Fields() {
					names = append(names, n)
				}
				sort.Strings(names)
				for _, n := range names {
					fv, ok := v.Field(n)
					if !ok {
						continue
					}
					// don't include default values
					skip := false
					switch n {
					case "bucketType":
						skip = fv.ToString() == "membase"
					case "storageBackend":
						skip = fv.ToString() == "couchstore"
					case "evictionPolicy":
						skip = fv.ToString() == "valueOnly"
					case "replicaNumber":
						skip = value.AsNumberValue(fv).Int64() == 1
					case "compressionMode":
						skip = fv.ToString() == "passive"
					case "threadsNumber":
						skip = value.AsNumberValue(fv).Int64() == 3
					case "maxTTL", "historyRetentionBytes", "historyRetentionSeconds", "numVBuckets":
						skip = value.AsNumberValue(fv).Int64() == 0
					case "conflictResolutionType":
						skip = fv.ToString() == "seqno"
					case "durabilityMinLevel":
						skip = fv.ToString() == "none"
					case "magmaSeqTreeDataBlockSize":
						skip = value.AsNumberValue(fv).Int64() == 4096
					case "historyRetentionCollectionDefault":
						skip = fv.Truth()
					case "databaseFragmentationThreshold[percentage]", "databaseFragmentationThreshold[size]",
						"viewFragmentationThreshold[percentage]", "viewFragmentationThreshold[size]":
						skip = fv.ToString() == "undefined"
					case "replicaIndex":
						skip = fv.Type() != value.NUMBER || value.AsNumberValue(fv).Int64() == 0
					}
					if !skip {
						sb.WriteRune('\'')
						sb.WriteString(n)
						sb.WriteString("':")
						switch {
						case n == "ramQuota":
							sb.WriteString(value.NewValue(value.AsNumberValue(fv).Int64() / util.MiB).ToString())
						case n == "replicaIndex":
							if fv.Truth() {
								sb.WriteRune('1')
							} else {
								sb.WriteRune('0')
							}
						case n == "flushEnabled":
							sb.WriteRune('1')
						case fv.Type() == value.STRING:
							sb.WriteRune('\'')
							sb.WriteString(fv.ToString())
							sb.WriteRune('\'')
						default:
							sb.WriteString(fv.ToString())
						}
						sb.WriteRune(',')
					}
				}
				if sb.Len() > 0 {
					bucketInfoBuf.WriteString(" WITH {")
					bucketInfoBuf.WriteString(sb.String()[:sb.Len()-1])
					bucketInfoBuf.WriteString("}")
				}
			}
			res = append(res, bucketInfoBuf.String()+";")
		}

		if flags&(_SCOPE_INFO|_COLLECTION_INFO) != 0 {
			stmt = "SELECT RAW name FROM system:scopes WHERE `bucket` = ? ORDER BY name"
			v, _, err := context.EvaluateStatement(stmt, nil, posArg, false, true, false, "")
			if err != nil {
				return value.MISSING_VALUE, err
			}
			scopes := v.Actual().([]interface{})
			for j := range scopes {
				scope := scopes[j].(string)
				sb.Reset()
				if flags&_SCOPE_INFO != 0 {
					sb.WriteString("CREATE SCOPE `")
					sb.WriteString(bucket)
					sb.WriteString("`.`")
					sb.WriteString(scope)
					sb.WriteString("`;")
					res = append(res, sb.String())
				}

				if flags&_COLLECTION_INFO != 0 {
					stmt = "SELECT name, maxTTL FROM system:keyspaces WHERE `bucket` = ? AND `scope` = ? ORDER BY name"
					v, _, err := context.EvaluateStatement(stmt, nil, append(posArg, value.NewValue(scope)), false, true, false, "")
					if err != nil {
						return value.MISSING_VALUE, err
					}
					for k := 0; ; k++ {
						cv, ok := v.Index(k)
						if !ok {
							break
						}
						name, ok := cv.Field("name")
						if !ok {
							return value.NULL_VALUE, nil
						}

						sb.Reset()
						sb.WriteString("CREATE COLLECTION `")
						sb.WriteString(bucket)
						sb.WriteString("`.`")
						sb.WriteString(scope)
						sb.WriteString("`.`")
						sb.WriteString(name.ToString())
						if maxTTL, ok := cv.Field("maxTTL"); ok {
							sb.WriteString(" WITH {'maxTTL':")
							sb.WriteString(maxTTL.ToString())
							sb.WriteRune('}')
						}
						sb.WriteRune(';')
						res = append(res, sb.String())
					}
				}
			}
		}

		if flags&_INDEX_INFO != 0 {
			stmt = "SELECT RAW CONCAT('CREATE INDEX `', s.name, '` ON ', k, ks, p, w, ';')" +
				" FROM system:indexes AS s" +
				" LET bid = CONCAT('`',s.bucket_id, '`')," +
				" sid = CONCAT('`', s.scope_id, '`')," +
				" kid = CONCAT('`', s.keyspace_id, '`')," +
				" k = NVL2(bid, CONCAT2('.', bid, sid, kid), kid)," +
				" ks = CASE WHEN s.`is_primary` THEN '' ELSE '(' || CONCAT2(',',s.`index_key`) || ') ' END," +
				" w = CASE WHEN s.`condition` IS VALUED THEN ' WHERE ' || REPLACE(s.`condition`, '\"','''') ELSE '' END," +
				" p = CASE WHEN s.`partition` IS VALUED THEN ' PARTITION BY ' || s.`partition` ELSE '' END" +
				" WHERE s.namespace_id = 'default'" +
				" AND s.`using` = 'gsi'" +
				" AND NVL(s.bucket_id,s.keyspace_id) = ?" +
				" ORDER BY s.name"
			v, _, err = context.EvaluateStatement(stmt, nil, posArg, false, true, false, "")
			if err != nil {
				return value.MISSING_VALUE, err
			}
			indices := v.Actual().([]interface{})
			res = append(res, indices...)
		}

		if flags&_SEQUENCE_INFO != 0 {
			// Generate such that the sequence continues from the current point.  Since we don't keep a history of alterations we
			// couldn't ever replay an exact sequence values generated and this approach allows the generated DDL to function with
			// the existing data (at least as well as the active sequence would).
			stmt = "SELECT RAW 'CREATE SEQUENCE '||`path`" +
				"||' START WITH '||TO_STRING(`value`.`~next_block`)" +
				"||' CACHE '||TO_STRING(`cache`)" +
				"||CASE WHEN `cycle` = false THEN ' NO CYCLE' ELSE ' CYCLE' END" +
				"||' INCREMENT BY '||TO_STRING(`increment`)" +
				"||CASE WHEN `min` != -9223372036854775808 THEN ' MINVALUE '||TO_STRING(`min`) ELSE '' END" +
				"||CASE WHEN `max` != 9223372036854775807 THEN ' MAXVALUE '||TO_STRING(`max`) ELSE '' END" +
				"||';'" +
				" FROM system:all_sequences" +
				" WHERE `bucket` = ?" +
				" ORDER BY `path`"
			v, _, err = context.EvaluateStatement(stmt, nil, posArg, false, true, false, "")
			if err != nil {
				return value.MISSING_VALUE, err
			}
			sequences := v.Actual().([]interface{})
			res = append(res, sequences...)
		}
	}

	return value.NewValue(res), nil
}

func (this *ExtractDDL) Constructor() FunctionConstructor {
	return NewExtractDDL
}

func (this *ExtractDDL) Indexable() bool {
	return false
}

func (this *ExtractDDL) MinArgs() int {
	return 1
}

func (this *ExtractDDL) MaxArgs() int {
	return 2
}
