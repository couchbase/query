//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package decode

import (
	"encoding/binary"
	"io"

	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/planshape"
)

func Decode(i io.Reader, o io.StringWriter) bool {
	buf := make([]byte, 2)
	n, err := i.Read(buf)
	if err != nil || n != 2 || binary.BigEndian.Uint16(buf) != planshape.MAGIC {
		return false
	}
	buf = buf[:1]
	for {
		n, err := i.Read(buf)
		if err != nil || n == 0 {
			return true
		}
		if !decodePSElem(buf, i, o) {
			return false
		}
	}
}

func readVal(buf []byte, i io.Reader, o io.StringWriter) bool {
	for {
		n, err := i.Read(buf)
		if err != nil {
			return false
		} else if buf[0] == 0x0 || n == 0 {
			return true
		}
		o.WriteString(string(buf))
	}
}

func readElems(buf []byte, i io.Reader, o io.StringWriter) bool {
	for c := 0; ; c++ {
		n, err := i.Read(buf)
		if err != nil {
			return false
		} else if buf[0] == 0x0 || n == 0 {
			return true
		}
		if c > 0 {
			o.WriteString(",")
		}
		if !decodePSElem(buf, i, o) {
			return false
		}
	}
}

var simple = map[byte]string{
	planshape.KEYSCAN:              "KeyScan",
	planshape.VALUESCAN:            "ValueScan",
	planshape.DUMMYSCAN:            "DummyScan",
	planshape.EXPRESSIONSCAN:       "ExpressionScan",
	planshape.INDEXFTSSEARCH:       "IndexFtsSearch",
	planshape.DUMMYFETCH:           "DummyFetch",
	planshape.JOIN:                 "Join",
	planshape.NEST:                 "Nest",
	planshape.UNNEST:               "Unnest",
	planshape.LET:                  "Let",
	planshape.FILTER:               "Filter",
	planshape.INITIALGROUP:         "InitialGroup",
	planshape.INTERMEDIATEGROUP:    "IntermediateGroup",
	planshape.FINALGROUP:           "FinalGroup",
	planshape.WINDOWAGGREGATE:      "WindowAggregate",
	planshape.INITIALPROJECT:       "InitialProject",
	planshape.INDEXCOUNTPROJECT:    "IndexCountProject",
	planshape.DISTINCT:             "Distinct",
	planshape.ALL:                  "All",
	planshape.ORDER:                "Order",
	planshape.OFFSET:               "Offset",
	planshape.LIMIT:                "Limit",
	planshape.SENDINSERT:           "SendInsert",
	planshape.SENDUPSERT:           "SendUpsert",
	planshape.SENDDELETE:           "SendDelete",
	planshape.CLONE:                "Clone",
	planshape.SET:                  "Set",
	planshape.UNSET:                "Unset",
	planshape.SENDUPDATE:           "SendUpdate",
	planshape.ALIAS:                "Alias",
	planshape.DISCARD:              "Discard",
	planshape.STREAM:               "Stream",
	planshape.COLLECT:              "Collect",
	planshape.RECEIVE:              "Receive",
	planshape.CHANNEL:              "Channel",
	planshape.CREATEPRIMARYINDEX:   "CreatePrimaryIndex",
	planshape.CREATEINDEX:          "CreateIndex",
	planshape.DROPINDEX:            "DropIndex",
	planshape.ALTERINDEX:           "AlterIndex",
	planshape.BUILDINDEXES:         "BuildIndexes",
	planshape.CREATESCOPE:          "CreateScope",
	planshape.DROPSCOPE:            "DropScope",
	planshape.CREATECOLLECTION:     "CreateCollection",
	planshape.DROPCOLLECTION:       "DropCollection",
	planshape.FLUSHCOLLECTION:      "FlushCollection",
	planshape.GRANTROLE:            "GrantRole",
	planshape.REVOKEROLE:           "RevokeRole",
	planshape.EXPLAIN:              "Explain",
	planshape.EXPLAINFUNCTION:      "ExplainFunction",
	planshape.PREPARE:              "Prepare",
	planshape.INFERKEYSPACE:        "InferKeyspace",
	planshape.INFEREXPRESSION:      "InferExpression",
	planshape.CREATEFUNCTION:       "CreateFunction",
	planshape.DROPFUNCTION:         "DropFunction",
	planshape.EXECUTEFUNCTION:      "ExecuteFunction",
	planshape.INDEXADVICE:          "IndexAdvice",
	planshape.ADVISE:               "Advise",
	planshape.UPDATESTATISTICS:     "UpdateStatistics",
	planshape.STARTTRANSACTION:     "StartTransaction",
	planshape.COMMITTRANSACTION:    "CommitTransaction",
	planshape.ROLLBACKTRANSACTION:  "RollbackTransaction",
	planshape.TRANSACTIONISOLATION: "TransactionIsolation",
	planshape.SAVEPOINT:            "Savepoint",
	planshape.CREATESEQUENCE:       "CreateSequence",
	planshape.DROPSEQUENCE:         "DropSequence",
	planshape.ALTERSEQUENCE:        "AlterSequence",
	planshape.CREATEBUCKET:         "CreateBucket",
	planshape.DROPBUCKET:           "DropBucket",
	planshape.ALTERBUCKET:          "AlterBucket",
	planshape.CREATEGROUP:          "CreateGroup",
	planshape.DROPGROUP:            "DropGroup",
	planshape.ALTERGROUP:           "AlterGroup",
	planshape.CREATEUSER:           "CreateUser",
	planshape.DROPUSER:             "DropUser",
	planshape.ALTERUSER:            "AlterUser",
}

func decodePSElem(buf []byte, i io.Reader, o io.StringWriter) bool {
	logging.Debugf("%#x", buf[0])
	if s, ok := simple[buf[0]]; ok {
		o.WriteString("{\"#operator\":\"")
		o.WriteString(s)
		o.WriteString("\"}")
		return true
	}
	switch buf[0] {
	case planshape.PRIMARYSCAN:
		o.WriteString("{\"#operator\":\"PrimaryScan\",\"index_id\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\",\"keyspace\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\"}")
	case planshape.PRIMARYSCAN3:
		o.WriteString("{\"#operator\":\"PrimaryScan3\",\"index_id\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\",\"keyspace\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\"}")
	case planshape.INDEXSCAN:
		o.WriteString("{\"#operator\":\"IndexScan\",\"index_id\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\",\"index\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\",\"~children\":[")
		if !readElems(buf, i, o) {
			return false
		}
		o.WriteString("]}")
	case planshape.INDEXSCAN2:
		return simpleIndex(buf, i, o, "IndexScan2")
	case planshape.INDEXSCAN3:
		o.WriteString("{\"#operator\":\"IndexScan3\",\"index_id\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\",\"name\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\"")
		n, err := i.Read(buf)
		if err != nil || n == 0 {
			return false
		}
		if buf[0]&planshape.IDX_OFFSET != 0 {
			o.WriteString(",\"offset\":true")
		}
		if buf[0]&planshape.IDX_LIMIT != 0 {
			o.WriteString(",\"limit\":true")
		}
		if buf[0]&planshape.IDX_GROUP != 0 {
			o.WriteString(",\"aggregates\":true")
		}
		if buf[0]&planshape.IDX_COVER != 0 {
			o.WriteString(",\"covering\":true")
		}
		if buf[0]&planshape.IDX_ORDER != 0 {
			o.WriteString(",\"order\":true")
		}
		o.WriteString("}")
	case planshape.COUNTSCAN:
		o.WriteString("{\"#operator\":\"CountScan\",\"alias\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\"}")
	case planshape.INDEXCOUNTSCAN:
		return simpleIndex(buf, i, o, "IndexCountScan")
	case planshape.INDEXCOUNTSCAN2:
		return simpleIndex(buf, i, o, "IndexCountScan2")
	case planshape.INDEXCOUNTDISTINCTSCAN2:
		return simpleIndex(buf, i, o, "IndexCountDistinctScan2")
	case planshape.DISTINCTSCAN:
		return simpleChildren(buf, i, o, "DistinctScan")
	case planshape.UNIONSCAN:
		return simpleChildren(buf, i, o, "UnionScan")
	case planshape.INTERSECTSCAN:
		return simpleChildren(buf, i, o, "IntersectScan")
	case planshape.ORDEREDINTERSECTSCAN:
		return simpleChildren(buf, i, o, "OrderedIntersectScan")
	case planshape.FETCH:
		o.WriteString("{\"#operator\":\"Fetch\",\"keyspace\":\"")
		if !readVal(buf, i, o) {
			return false
		}
		o.WriteString("\"}")
	case planshape.INDEXJOIN:
		return simpleIndex(buf, i, o, "IndexJoin")
	case planshape.NEST:
		return simpleIndex(buf, i, o, "IndexNest")
	case planshape.NLJOIN:
		return simpleChildren(buf, i, o, "NLJoin")
	case planshape.NLNEST:
		return simpleChildren(buf, i, o, "NLNest")
	case planshape.HASHJOIN:
		return simpleChildren(buf, i, o, "HashJoin")
	case planshape.HASHNEST:
		return simpleChildren(buf, i, o, "HashNest")
	case planshape.WITH:
		return simpleChildren(buf, i, o, "With")
	case planshape.UNIONALL:
		return simpleChildren(buf, i, o, "UnionAll")
	case planshape.INTERSECT:
		return simpleChildren(buf, i, o, "Intersect")
	case planshape.INTERSECTALL:
		return simpleChildren(buf, i, o, "IntersectAll")
	case planshape.EXCEPT:
		return simpleChildren(buf, i, o, "Except")
	case planshape.EXCEPTALL:
		return simpleChildren(buf, i, o, "ExceptAll")
	case planshape.MERGE:
		return simpleChildren(buf, i, o, "Merge")
	case planshape.PARALLEL:
		return simpleChildren(buf, i, o, "Parallel")
	case planshape.SEQUENCE:
		return simpleChildren(buf, i, o, "Sequence")
	default:
		logging.Debugf("Invalid plan shape: %#x", buf[0])
		return false
	}
	return true
}

func simpleChildren(buf []byte, i io.Reader, o io.StringWriter, op string) bool {
	o.WriteString("{\"#operator\":\"")
	o.WriteString(op)
	o.WriteString("\",\"~children\":[")
	if !readElems(buf, i, o) {
		return false
	}
	o.WriteString("]}")
	return true
}

func simpleIndex(buf []byte, i io.Reader, o io.StringWriter, op string) bool {
	o.WriteString("{\"#operator\":\"")
	o.WriteString(op)
	o.WriteString("\",\"index_id\":\"")
	if !readVal(buf, i, o) {
		return false
	}
	o.WriteString("\",\"index\":\"")
	if !readVal(buf, i, o) {
		return false
	}
	o.WriteString("\"}")
	return true
}
