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
	"strconv"
	"strings"
	"time"

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

// convertN1QLToIcebergFilter converts N1QL expression to iceberg filter
// Returns nil filter if conversion fails (instead of error)
func convertN1QLToIcebergFilter(expr expression.Expression, alias string, parent value.Value) IcebergFilter {
	if expr == nil {
		return IcebergFilter{}
	}

	// Process the expression directly and handle alias in field extraction
	switch e := expr.(type) {
	case *expression.Eq:
		// equality comparison (commutative)
		field, val, _, valid := extractComparison(e.First(), e.Second(), alias, parent)
		if !valid {
			return IcebergFilter{} // conversion failed
		}
		return IcebergFilter{Op: "=", Field: field, Value: val}

	case *expression.LT:
		// less than
		field, val, inverted, valid := extractComparison(e.First(), e.Second(), alias, parent)
		if !valid {
			return IcebergFilter{} // conversion failed
		}
		// If operands are inverted (constant < field), change to field > constant
		if inverted {
			return IcebergFilter{Op: ">", Field: field, Value: val}
		}
		return IcebergFilter{Op: "<", Field: field, Value: val}

	case *expression.LE:
		// less than or equal
		field, val, inverted, valid := extractComparison(e.First(), e.Second(), alias, parent)
		if !valid {
			return IcebergFilter{} // conversion failed
		}
		// If operands are inverted (constant <= field), change to field >= constant
		if inverted {
			return IcebergFilter{Op: ">=", Field: field, Value: val}
		}
		return IcebergFilter{Op: "<=", Field: field, Value: val}

	case *expression.And:
		// logical AND
		filters := make([]IcebergFilter, 0, len(e.Operands()))
		for _, op := range e.Operands() {
			f := convertN1QLToIcebergFilter(op, alias, parent)
			if f.Op != "" || len(f.Children) > 0 {
				filters = append(filters, f)
			}
		}
		if len(filters) == 0 {
			return IcebergFilter{} // conversion failed for all children
		}
		return IcebergFilter{Op: "and", Children: filters}

	case *expression.Or:
		// logical OR
		filters := make([]IcebergFilter, 0, len(e.Operands()))
		for _, op := range e.Operands() {
			f := convertN1QLToIcebergFilter(op, alias, parent)
			if f.Op != "" || len(f.Children) > 0 {
				filters = append(filters, f)
			}
		}
		if len(filters) == 0 {
			return IcebergFilter{} // conversion failed for all children
		}
		return IcebergFilter{Op: "or", Children: filters}

	case *expression.Not:
		// logical NOT or inverted comparison
		// Check if the operand is a comparison operator and invert it
		if op, ok := e.Operand().(*expression.Eq); ok {
			// NOT(Eq(a, b)) -> NE(a, b)
			field, val, _, valid := extractComparison(op.First(), op.Second(), alias, parent)
			if !valid {
				return IcebergFilter{}
			}
			return IcebergFilter{Op: "!=", Field: field, Value: val}
		}
		if op, ok := e.Operand().(*expression.LT); ok {
			// NOT(LT(a, b)) -> GE(a, b)
			field, val, inverted, valid := extractComparison(op.First(), op.Second(), alias, parent)
			if !valid {
				return IcebergFilter{}
			}
			// NOT(LT(field, const)) -> GE(field, const)
			// NOT(LT(const, field)) -> NOT(GT(field, const)) -> LE(field, const)
			if inverted {
				return IcebergFilter{Op: "<=", Field: field, Value: val}
			}
			return IcebergFilter{Op: ">=", Field: field, Value: val}
		}
		if op, ok := e.Operand().(*expression.LE); ok {
			// NOT(LE(a, b)) -> GT(a, b)
			field, val, inverted, valid := extractComparison(op.First(), op.Second(), alias, parent)
			if !valid {
				return IcebergFilter{}
			}
			// NOT(LE(field, const)) -> GT(field, const)
			// NOT(LE(const, field)) -> NOT(GE(field, const)) -> LT(field, const)
			if inverted {
				return IcebergFilter{Op: "<", Field: field, Value: val}
			}
			return IcebergFilter{Op: ">", Field: field, Value: val}
		}
		// For other NOT cases, handle as logical NOT
		f := convertN1QLToIcebergFilter(e.Operand(), alias, parent)
		if f.Op != "" || len(f.Children) > 0 {
			return IcebergFilter{Op: "not", Children: []IcebergFilter{f}}
		}
		return IcebergFilter{} // conversion failed

	case *expression.In:
		// IN operator
		field, valid := extractFieldName(e.First(), alias, parent)
		if !valid {
			return IcebergFilter{} // conversion failed
		}
		vals := extractArrayValues(e.Second(), parent)
		if vals == nil {
			return IcebergFilter{} // conversion failed
		}
		return IcebergFilter{Op: "in", Field: field, Value: vals}

	case *expression.IsNull:
		// IS NULL check
		field, valid := extractFieldName(e.Operand(), alias, parent)
		if !valid {
			return IcebergFilter{} // conversion failed
		}
		return IcebergFilter{Op: "is_null", Field: field, Value: nil}

	case *expression.IsNotNull:
		// IS NOT NULL check
		field, valid := extractFieldName(e.Operand(), alias, parent)
		if !valid {
			return IcebergFilter{} // conversion failed
		}
		return IcebergFilter{Op: "is_not_null", Field: field, Value: nil}

	case *expression.Like:
		// LIKE operator
		field, valid := extractFieldName(e.First(), alias, parent)
		if !valid {
			return IcebergFilter{} // conversion failed
		}
		val := extractValue(e.Second(), parent)
		if strVal, ok := val.(string); ok {
			return IcebergFilter{Op: "like", Field: field, Value: strVal}
		}
		return IcebergFilter{} // conversion failed

	default:
		// Unsupported expression type
		return IcebergFilter{} // conversion failed
	}
}

// extractFieldName extracts a field name from a simple path expression.
// Returns (fieldName, valid).
// Only a single-segment path is accepted: t.col or bare col.
// Multi-segment paths (t.a.b), compound expressions (t.c1+t.c2), and
// correlated references from a different alias are all rejected.
func extractFieldName(expr expression.Expression, alias string, parent value.Value) (path string, valid bool) {
	exprAlias, field, err := expression.PathString(expr)
	if err != nil || field == "" {
		return "", false
	}
	// Reject expressions that belong to a different scope (correlated outer reference).
	if alias != "" && exprAlias != "" && exprAlias != alias {
		return "", false
	}
	// PathString wraps each segment in backticks: `a`.`b`.`c`
	// Remove all backticks so Iceberg receives the plain field name.
	field = strings.ReplaceAll(field, "`", "")
	// Reject multi-segment paths such as address.city — must be a single field name.
	if strings.Contains(field, ".") {
		return "", false
	}
	return field, true
}

// extractComparison extracts field and value from a binary comparison expression
// Returns (field, value, inverted, valid)
// - field: the field name
// - value: the constant value (or parent-evaluated value)
// - inverted: true if we need to invert the operator due to operand swap
// - valid: true if extraction succeeded
func extractComparison(first, second expression.Expression, alias string, parent value.Value) (path string, val interface{}, inverted bool, valid bool) {
	firstVal := extractValue(first, parent)
	secondVal := extractValue(second, parent)

	// Case 1: field = constant/parent-value
	if firstVal == nil && secondVal != nil {
		path, ok := extractFieldName(first, alias, parent)
		if !ok || path == "" {
			return "", nil, false, false
		}
		return path, secondVal, false, true
	}

	// Case 2: constant/parent-value = field - need to invert operator
	if firstVal != nil && secondVal == nil {
		path, ok := extractFieldName(second, alias, parent)
		if !ok || path == "" {
			return "", nil, false, false
		}
		return path, firstVal, true, true
	}

	return "", nil, false, false
}

// extractValue extracts a value from an expression — constant first, then evaluated against parent.
func extractValue(expr expression.Expression, parent value.Value) interface{} {
	if v := expr.Value(); v != nil {
		return v.Actual()
	}
	if parent != nil {
		if v, err := expr.Evaluate(parent, nil); err == nil && v != nil &&
			v.Type() != value.MISSING && v.Type() != value.NULL {
			return v.Actual()
		}
	}
	return nil
}

// extractArrayValues extracts values from an array expression, using parent for non-constant elements.
func extractArrayValues(expr expression.Expression, parent value.Value) []interface{} {
	if arrConstruct, ok := expr.(*expression.ArrayConstruct); ok {
		result := make([]interface{}, 0, len(arrConstruct.Operands()))
		for _, op := range arrConstruct.Operands() {
			val := extractValue(op, parent)
			if val != nil {
				result = append(result, val)
			}
		}
		return result
	}
	// Fall back to evaluating the whole expression against parent (e.g. a parameter array).
	if parent != nil {
		if v, err := expr.Evaluate(parent, nil); err == nil && v != nil && v.Type() != value.MISSING {
			if arr, ok := v.Actual().([]interface{}); ok {
				return arr
			}
		}
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
		}
		params.ScanOptions = opts
	}

	opts.Filters = nil
	if params.Filter != nil {
		icebergFilter := convertN1QLToIcebergFilter(params.Filter, params.Alias, params.Parent)
		if icebergFilter.Op != "" || len(icebergFilter.Children) > 0 {
			opts.Filters = []IcebergFilter{icebergFilter}
			logging.Infof("scanIcebergCatalog: Applied filter pushdown: op=%s, field=%s", icebergFilter.Op, icebergFilter.Field)
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
				logging.Infof("Iceberg scan completed: %d rows sent, %d rows received, %d rows skipped, %d conversion errors (offsetApplied=%d)",
					rowsSent, rowsReceived, rowsSkipped, conversionErrors, offsetApplied)
				return nil
			}
			return errors.NewDatastoreExternalCollectionError(err,
				"Error during Iceberg scan", params.ErrTemplate)

		case row, ok := <-resultChan:
			if !ok {
				logging.Infof("Iceberg scan completed: %d rows sent, %d rows received, %d rows skipped, %d conversion errors (offsetApplied=%d)",
					rowsSent, rowsReceived, rowsSkipped, conversionErrors, offsetApplied)
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
