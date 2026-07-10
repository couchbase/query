//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package external

import (
	go_context "context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	iceberg "github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	icebergutils "github.com/apache/iceberg-go/utils"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/extparams"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const IcebergScanTimeout = 2 * time.Minute

// formatBytesRead reports how much of the touched files' on-disk size was actually
// fetched. Range-read scans (column/row-group pruning) fetch less than the full file,
// in which case this reports "read of planned"; a plain whole-file read reports just
// the bytes read since read == planned.
func formatBytesRead(bytesRead, bytesPlanned int64) string {
	if bytesPlanned > bytesRead {
		return fmt.Sprintf("%s of %s read", logging.HumanReadableSize(bytesRead, false),
			logging.HumanReadableSize(bytesPlanned, false))
	}
	return logging.HumanReadableSize(bytesRead, false) + " read"
}

// formatPruningStats summarizes row-group and column pruning across every parquet
// file touched by the scan (row groups summed across files; columns are scan-wide,
// not per-file). Returns "" when no parquet files with these stats were read (e.g.
// Avro/ORC-only scans, or a scan path that doesn't do row-group/column pruning).
func formatPruningStats(scanner *Scanner) string {
	keptRG, totalRG := scanner.RowGroupStats()
	selectedCols, totalCols := scanner.ColumnStats()
	if totalRG == 0 && totalCols == 0 {
		return ""
	}
	return fmt.Sprintf(", %d/%d row groups kept, %d/%d columns selected", keptRG, totalRG, selectedCols, totalCols)
}

// n1qlToIcebergExpr converts a N1QL WHERE-clause expression directly to an
// iceberg-go BooleanExpression for filter pushdown.
// Returns nil when the expression cannot be represented as an Iceberg predicate.
// For AND, convertible children are kept and non-convertible ones are dropped
// (a partial pushdown that only widens the scan is safe).
// For OR, all children must convert; if any fail the whole OR is dropped
// (dropping part of an OR would incorrectly exclude rows that match the failed branch).
func n1qlToIcebergExpr(expr expression.Expression, alias string, parent value.Value) iceberg.BooleanExpression {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *expression.Eq:
		return n1qlCompareExpr(iceberg.OpEQ, e.First(), e.Second(), alias, parent)

	case *expression.LT:
		// LT(field, const) → field < const; LT(const, field) → field > const
		return n1qlCompareExpr(iceberg.OpLT, e.First(), e.Second(), alias, parent)

	case *expression.LE:
		return n1qlCompareExpr(iceberg.OpLTEQ, e.First(), e.Second(), alias, parent)

	case *expression.And:
		var children []iceberg.BooleanExpression
		for _, op := range e.Operands() {
			if child := n1qlToIcebergExpr(op, alias, parent); child != nil {
				children = append(children, child)
			}
		}
		if len(children) == 0 {
			return nil
		}
		result := children[0]
		for _, c := range children[1:] {
			result = iceberg.NewAnd(result, c)
		}
		return result

	case *expression.Or:
		children := make([]iceberg.BooleanExpression, 0, len(e.Operands()))
		for _, op := range e.Operands() {
			child := n1qlToIcebergExpr(op, alias, parent)
			if child == nil {
				return nil // any missing branch makes the OR un-pushable
			}
			children = append(children, child)
		}
		if len(children) == 0 {
			return nil
		}
		result := children[0]
		for _, c := range children[1:] {
			result = iceberg.NewOr(result, c)
		}
		return result

	case *expression.Not:
		// Fold NOT into comparison operators where possible for cleaner predicates.
		switch inner := e.Operand().(type) {
		case *expression.Eq:
			return n1qlCompareExpr(iceberg.OpNEQ, inner.First(), inner.Second(), alias, parent)
		case *expression.LT:
			// NOT(a < b)  →  a >= b
			return n1qlCompareExpr(iceberg.OpGTEQ, inner.First(), inner.Second(), alias, parent)
		case *expression.LE:
			// NOT(a <= b) →  a > b
			return n1qlCompareExpr(iceberg.OpGT, inner.First(), inner.Second(), alias, parent)
		case *expression.In:
			// NOT(field IN (...))  →  native OpNotIn predicate
			field, ok := n1qlFieldPath(inner.First(), alias)
			if !ok {
				return nil
			}
			vals := n1qlArrayValues(inner.Second(), parent)
			if len(vals) == 0 {
				return nil
			}
			ref := iceberg.Reference(field)
			lits := make([]iceberg.Literal, 0, len(vals))
			for _, v := range vals {
				lit := n1qlValueToLiteral(v)
				if lit == nil {
					continue
				}
				lits = append(lits, lit)
			}
			if len(lits) == 0 {
				return nil
			}
			if len(lits) > 200 {
				logging.Warnf("Iceberg filter pushdown: NOT IN predicate on %q has %d values (>200); partition pruning disabled for this filter", field, len(lits))
			}
			return iceberg.SetPredicate(iceberg.OpNotIn, ref, lits)
		}
		if child := n1qlToIcebergExpr(e.Operand(), alias, parent); child != nil {
			return iceberg.NewNot(child)
		}
		return nil

	case *expression.In:
		field, ok := n1qlFieldPath(e.First(), alias)
		if !ok {
			return nil
		}
		vals := n1qlArrayValues(e.Second(), parent)
		if len(vals) == 0 {
			return nil
		}
		ref := iceberg.Reference(field)
		lits := make([]iceberg.Literal, 0, len(vals))
		for _, v := range vals {
			lit := n1qlValueToLiteral(v)
			if lit == nil {
				continue
			}
			lits = append(lits, lit)
		}
		if len(lits) == 0 {
			return nil
		}
		if len(lits) > 200 {
			logging.Warnf("Iceberg filter pushdown: IN predicate on %q has %d values (>200); partition pruning disabled for this filter", field, len(lits))
		}
		// SetPredicate(OpIn) is iceberg-go's native IN. When len > 200 the manifest
		// evaluator skips pruning (returns rowsMightMatch) rather than erroring — safe.
		return iceberg.SetPredicate(iceberg.OpIn, ref, lits)

	case *expression.Between:
		// item BETWEEN low AND high  →  item >= low AND item <= high
		ops := e.Operands()
		if len(ops) != 3 {
			return nil
		}
		field, ok := n1qlFieldPath(ops[0], alias)
		if !ok {
			return nil
		}
		ref := iceberg.Reference(field)
		lowVal := n1qlExtractValue(ops[1], parent)
		highVal := n1qlExtractValue(ops[2], parent)
		if lowVal == nil || highVal == nil {
			return nil
		}
		lowLit := n1qlValueToLiteral(lowVal)
		highLit := n1qlValueToLiteral(highVal)
		if lowLit == nil || highLit == nil {
			return nil
		}
		return iceberg.NewAnd(
			iceberg.LiteralPredicate(iceberg.OpGTEQ, ref, lowLit),
			iceberg.LiteralPredicate(iceberg.OpLTEQ, ref, highLit),
		)

	case *expression.IsNull:
		field, ok := n1qlFieldPath(e.Operand(), alias)
		if !ok {
			return nil
		}
		return iceberg.IsNull(iceberg.Reference(field))

	case *expression.IsNotNull:
		field, ok := n1qlFieldPath(e.Operand(), alias)
		if !ok {
			return nil
		}
		return iceberg.NotNull(iceberg.Reference(field))

	case *expression.Like:
		field, ok := n1qlFieldPath(e.First(), alias)
		if !ok {
			return nil
		}
		val := n1qlExtractValue(e.Second(), parent)
		pattern, ok := val.(string)
		if !ok {
			return nil
		}
		ref := iceberg.Reference(field)
		// Pure prefix pattern "abc%" — no wildcards before the trailing %
		if strings.HasSuffix(pattern, "%") && !strings.ContainsAny(pattern[:len(pattern)-1], "%_") {
			return iceberg.StartsWith(ref, pattern[:len(pattern)-1])
		}
		// Exact string (no wildcards)
		if !strings.ContainsAny(pattern, "%_") {
			return iceberg.EqualTo(ref, pattern)
		}
		return nil
	}
	return nil
}

// n1qlCompareExpr builds an Iceberg comparison predicate from a N1QL binary comparison.
// When first is the field and second is the constant, the op is used as-is.
// When first is the constant and second is the field, the op is flipped (e.g. LT → GT).
func n1qlCompareExpr(op iceberg.Operation, first, second expression.Expression, alias string, parent value.Value) iceberg.BooleanExpression {
	firstVal := n1qlExtractValue(first, parent)
	secondVal := n1qlExtractValue(second, parent)

	var field string
	var val interface{}
	var flip bool

	switch {
	case firstVal == nil && secondVal != nil:
		// field OP const
		var ok bool
		field, ok = n1qlFieldPath(first, alias)
		if !ok {
			return nil
		}
		val = secondVal
	case firstVal != nil && secondVal == nil:
		// const OP field  →  field flipped-OP const
		var ok bool
		field, ok = n1qlFieldPath(second, alias)
		if !ok {
			return nil
		}
		val = firstVal
		flip = true
	default:
		return nil
	}

	if flip {
		op = flipOp(op)
	}

	ref := iceberg.Reference(field)
	lit := n1qlValueToLiteral(val)
	if lit == nil {
		return nil
	}
	return iceberg.LiteralPredicate(op, ref, lit)
}

// flipOp reverses the direction of a comparison operator (swapping operands).
func flipOp(op iceberg.Operation) iceberg.Operation {
	switch op {
	case iceberg.OpLT:
		return iceberg.OpGT
	case iceberg.OpLTEQ:
		return iceberg.OpGTEQ
	case iceberg.OpGT:
		return iceberg.OpLT
	case iceberg.OpGTEQ:
		return iceberg.OpLTEQ
	default:
		return op // EQ, NEQ are symmetric
	}
}

// n1qlFieldPath extracts a dot-separated field path from a N1QL identifier expression.
// Supports both top-level fields (t.name → "name") and nested fields
// (t.address.city → "address.city") using Iceberg's dot-notation for nested access.
// Correlated references from a different alias are rejected.
func n1qlFieldPath(expr expression.Expression, alias string) (string, bool) {
	exprAlias, field, err := expression.PathString(expr)
	if err != nil || field == "" {
		return "", false
	}
	if alias != "" && exprAlias != "" && exprAlias != alias {
		return "", false
	}
	// PathString wraps each segment in backticks: `address`.`city`
	// Strip backticks; keep "." as the Iceberg nested-field separator.
	field = strings.ReplaceAll(field, "`", "")
	return field, true
}

// n1qlExtractValue returns the constant value of a N1QL expression as a plain Go value,
// or nil if the expression is not a constant (or evaluates to NULL/MISSING).
func n1qlExtractValue(expr expression.Expression, parent value.Value) interface{} {
	if v := expr.Value(); v != nil {
		act := v.Actual()
		if act == nil {
			return nil // NULL
		}
		return act
	}
	if parent != nil {
		if v, err := expr.Evaluate(parent, nil); err == nil && v != nil &&
			v.Type() != value.MISSING && v.Type() != value.NULL {
			return v.Actual()
		}
	}
	return nil
}

// n1qlArrayValues collects constant values from an array expression.
// Handles three shapes:
//   - *expression.ArrayConstruct: e.g. literal [1, 2, 3] in the source query
//   - any expression whose Value() returns a pre-computed array — including
//     *expression.Constant, which is how the join-driven runtime filter
//     (execution/external_filter.go) wraps the IN-list values
//   - falls back to evaluating against parent for correlated references
func n1qlArrayValues(expr expression.Expression, parent value.Value) []interface{} {
	if arr, ok := expr.(*expression.ArrayConstruct); ok {
		result := make([]interface{}, 0, len(arr.Operands()))
		for _, op := range arr.Operands() {
			if v := n1qlExtractValue(op, parent); v != nil {
				result = append(result, v)
			}
		}
		return result
	}
	if v := expr.Value(); v != nil {
		if arr, ok := v.Actual().([]interface{}); ok {
			return arr
		}
	}
	if parent != nil {
		if v, err := expr.Evaluate(parent, nil); err == nil && v != nil && v.Type() != value.MISSING {
			if arr, ok := v.Actual().([]interface{}); ok {
				return arr
			}
		}
	}
	return nil
}

// n1qlValueToLiteral converts a plain Go value (from value.Actual()) to an iceberg.Literal.
// N1QL stores all JSON numbers as float64; whole-number floats are converted to int64
// so that iceberg can coerce them to the column's actual integer type during binding.
func n1qlValueToLiteral(val interface{}) iceberg.Literal {
	if v, ok := val.(value.Value); ok {
		val = v.Actual()
	}
	switch v := val.(type) {
	case string:
		return iceberg.StringLiteral(v)
	case bool:
		return iceberg.BoolLiteral(v)
	case float64:
		if v == float64(int64(v)) && v >= math.MinInt64 && v <= math.MaxInt64 {
			return iceberg.Int64Literal(int64(v))
		}
		return iceberg.Float64Literal(v)
	case int32:
		return iceberg.Int32Literal(v)
	case int64:
		return iceberg.Int64Literal(v)
	case float32:
		return iceberg.Float32Literal(v)
	}
	return nil
}

// ScanIcebergCatalog scans an Iceberg table from AWS Glue catalog
func ScanIcebergCatalog(externalEntry *extparams.ExternalCollectionEntry, params *datastore.ExternalScanParams,
	queryContext datastore.QueryContext, conn *datastore.IndexConnection) errors.Error {

	catalogInfo := externalEntry.CatalogInfo

	var opts *ScanOptions
	if params.ScanOptions != nil {
		opts = params.ScanOptions.(*ScanOptions)
	} else {
		awsCfg, err := GetAWSConfig(catalogInfo.Source, params.CatalogCred, catalogInfo.SigV4SigningRegion)
		if err != nil {
			return errors.NewDatastoreExternalCollectionError(nil, err.Error(), params.ErrTemplate)
		}
		// For non-AWS-native catalog sources (NESSIE, REST, …) GetAWSConfig returns nil because
		// IsAWSSource() is false. Without an explicit aws.Config in the context iceberg-go falls
		// back to the ambient credential chain (env vars, ~/.aws, IMDS) for S3 FileIO. Inject the
		// collection credential — the storage credential — so the correct token is used.
		if awsCfg == nil {
			awsCfg, err = GetStorageAWSConfig(params.CollectionCred, catalogInfo.SigV4SigningRegion)
			if err != nil {
				return errors.NewDatastoreExternalCollectionError(nil,
					"Failed to build AWS config from collection credential: "+err.Error(), params.ErrTemplate)
			}
		}

		var snapshotId *int64
		snapshot_id := externalEntry.SnapshotId
		if params.SnapshotId != "" {
			snapshot_id = params.SnapshotId
		}
		if snapshot_id != "" {
			snId, err := strconv.ParseInt(snapshot_id, 10, 64)
			if err != nil {
				return errors.NewDatastoreExternalCollectionError(nil,
					"Failed to parse snapshot ID: "+err.Error(), params.ErrTemplate)
			}
			snapshotId = &snId
		}

		var snapshotAsOf *int64
		snapshotTs := externalEntry.SnapshotTimestamp
		if params.SnapshotTimestamp != "" {
			snapshotTs = params.SnapshotTimestamp
		}
		if snapshotTs != "" {
			snTs, err := strconv.ParseInt(snapshotTs, 10, 64)
			if err != nil {
				t, err1 := time.Parse(time.RFC3339Nano, snapshotTs)
				if err1 == nil {
					milliseconds := t.UnixNano() / int64(time.Millisecond)
					snapshotAsOf = &milliseconds
				}
			} else {
				snapshotAsOf = &snTs
			}
		}

		database := externalEntry.Namespace
		if database == "" {
			return errors.NewDatastoreExternalCollectionError(nil, "Namespace (database) not found in collection metadata", params.ErrTemplate)
		}
		tableName := externalEntry.TableName
		if tableName == "" {
			return errors.NewDatastoreExternalCollectionError(nil, "Table name not found in collection metadata", params.ErrTemplate)
		}
		parallelScans := externalEntry.ParallelScans
		if parallelScans <= 0 {
			parallelScans = 1
		} else if parallelScans > util.NumCPU() {
			parallelScans = util.NumCPU()
		}

		opts = &ScanOptions{
			Database:           database,
			Table:              tableName,
			SnapshotID:         snapshotId,
			SnapshotAsOf:       snapshotAsOf,
			SelectedFields:     params.Projection,
			CaseSensitive:      true,
			Limit:              params.Limit,
			AwsConfig:          awsCfg,
			SourceType:         catalogInfo.Source,
			URI:                catalogInfo.URI,
			Warehouse:          catalogInfo.Warehouse,
			SigV4SigningRegion: catalogInfo.SigV4SigningRegion,
			SigV4SigningName:   catalogInfo.SigV4SigningName,
			Credential:         externalEntry.CredentialId,
			CatalogCred:        params.CatalogCred,
			CollectionCred:     params.CollectionCred,
			QuotaProjectID:     catalogInfo.QuotaProjectID,
			ParallelScans:      parallelScans,
			DecimalToDouble:    externalEntry.DecimalToDouble,
			SQLDialect:         catalogInfo.SQLDialect,
			Branch:             catalogInfo.Branch,
		}
		params.ScanOptions = opts
	}

	opts.FilterExpr = nil
	if params.Filter != nil {
		if expr := n1qlToIcebergExpr(params.Filter, params.Alias, params.Parent); expr != nil {
			opts.FilterExpr = expr
			logging.Infof("scanIcebergCatalog (requestId=%s): Applied filter pushdown: %v", queryContext.RequestId(), expr)
		}
	}

	deadline := time.Now().Add(IcebergScanTimeout)
	if qd := queryContext.GetReqDeadline(); !qd.IsZero() && qd.Before(deadline) {
		deadline = qd
	}
	ctx, cancel := go_context.WithDeadline(go_context.Background(), deadline)
	defer cancel()
	if opts.AwsConfig != nil {
		ctx = icebergutils.WithAwsConfig(ctx, opts.AwsConfig)
	}

	var cat catalog.Catalog
	if params.ScanCatalog != nil {
		cat, _ = params.ScanCatalog.(catalog.Catalog)
	}
	scanner, scanErr := NewScanner(ctx, *opts, cat)
	if scanErr != nil {
		return errors.NewDatastoreExternalCollectionError(scanErr,
			"Failed to create Iceberg scanner", params.ErrTemplate)
	}
	if cat == nil {
		params.ScanCatalog = scanner.Catalog()
	}
	defer scanner.Close()

	if loadErr := scanner.LoadTable(ctx); loadErr != nil {
		return errors.NewDatastoreExternalCollectionError(loadErr,
			"Failed to load Iceberg table", params.ErrTemplate)
	}

	if params.CountOnly {
		count, countErr := scanner.CountRows(ctx)
		if countErr != nil {
			return errors.NewDatastoreExternalCollectionError(countErr,
				"Failed to count Iceberg table rows", params.ErrTemplate)
		}
		conn.Sender().SendEntry(&datastore.IndexEntry{EntryKey: []value.Value{value.NewValue(count)}})
		return nil
	}

	// Stream scan results and send to index connection
	resultChan, errorChan := scanner.ScanAndConvertStream(ctx)

	var rowsSent int64
	var rowsReceived int64
	var rowsSkipped int64
	var conversionErrors int64
	offsetApplied := int64(0)

	for {
		if conn.Sender().IsStopped() || !queryContext.IsActive() {
			cancel()
			return nil
		}

		select {
		case err, ok := <-errorChan:
			if !ok {
				// errorChan closed before resultChan — drain remaining rows
				for row := range resultChan {
					rowsReceived++
					if offsetApplied < params.Offset {
						offsetApplied++
						rowsSkipped++
						continue
					}
					if params.Limit > 0 && rowsSent >= params.Limit {
						continue
					}
					if val, err := value.ObjectToValue(row, params.ResultObject); err != nil {
						conversionErrors++
					} else {
						entry := &datastore.IndexEntry{
							PrimaryKey: "",
							EntryKey:   []value.Value{val},
						}
						conn.Sender().SendEntry(entry)
						rowsSent++
					}
				}
				logging.Infof("Iceberg scan completed (requestId=%s): %d rows sent, %d rows received, %d rows skipped, %d conversion errors (offsetApplied=%d), "+
					"%d files scanned, %s%s",
					queryContext.RequestId(), rowsSent, rowsReceived, rowsSkipped, conversionErrors, offsetApplied,
					scanner.FilesScanned(), formatBytesRead(scanner.BytesRead(), scanner.BytesPlanned()),
					formatPruningStats(scanner))
				return nil
			}
			return errors.NewDatastoreExternalCollectionError(err,
				"Error during Iceberg scan", params.ErrTemplate)

		case row, ok := <-resultChan:
			if !ok {
				logging.Infof("Iceberg scan completed (requestId=%s): %d rows sent, %d rows received, %d rows skipped, %d conversion errors (offsetApplied=%d), "+
					"%d files scanned, %s%s",
					queryContext.RequestId(), rowsSent, rowsReceived, rowsSkipped, conversionErrors, offsetApplied,
					scanner.FilesScanned(), formatBytesRead(scanner.BytesRead(), scanner.BytesPlanned()),
					formatPruningStats(scanner))
				return nil
			}

			rowsReceived++

			if offsetApplied < params.Offset {
				offsetApplied++
				rowsSkipped++
				continue
			}

			if params.Limit > 0 && rowsSent >= params.Limit {
				logging.Infof("Iceberg scan limit reached: %d rows sent, %d rows received total", rowsSent, rowsReceived)
				cancel()
				return nil
			}

			val, err := value.ObjectToValue(row, params.ResultObject)
			if err != nil {
				conversionErrors++
				logging.Warnf("Failed to convert row %d to value (total errors: %d): %v", rowsReceived, conversionErrors, err)
				continue
			}

			entry := &datastore.IndexEntry{
				PrimaryKey: "",
				EntryKey:   []value.Value{val},
			}
			conn.Sender().SendEntry(entry)
			rowsSent++
		}
	}
}
